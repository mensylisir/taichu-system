package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
	"gorm.io/gorm"
)

// ConstraintMonitorWorker 约束监控工作器
type ConstraintMonitorWorker struct {
	tenantRepo       *repository.TenantRepository
	environmentRepo  *repository.EnvironmentRepository
	applicationRepo  *repository.ApplicationRepository
	quotaRepo        *repository.QuotaRepository
	db               *gorm.DB
	constraintMonitor *service.ConstraintMonitor
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	checkInterval    time.Duration
	alertInterval    time.Duration
}

// NewConstraintMonitorWorker 创建约束监控工作器
func NewConstraintMonitorWorker(
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	db *gorm.DB,
) *ConstraintMonitorWorker {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := service.NewConstraintMonitor(
		tenantRepo,
		environmentRepo,
		applicationRepo,
		quotaRepo,
		db,
	)

	return &ConstraintMonitorWorker{
		tenantRepo:       tenantRepo,
		environmentRepo:  environmentRepo,
		applicationRepo:  applicationRepo,
		quotaRepo:        quotaRepo,
		db:               db,
		constraintMonitor: monitor,
		ctx:              ctx,
		cancel:           cancel,
		checkInterval:    5 * time.Minute,  // 每5分钟检查一次
		alertInterval:    1 * time.Hour,    // 每小时发送告警汇总
	}
}

// Start 启动工作器
func (w *ConstraintMonitorWorker) Start() {
	log.Println("Starting constraint monitor worker...")

	// 执行一次初始检查
	w.performConstraintCheck()

	w.wg.Add(1)
	go w.scheduler()
}

// Stop 停止工作器
func (w *ConstraintMonitorWorker) Stop() {
	log.Println("Stopping constraint monitor worker...")
	w.cancel()
	w.wg.Wait()
}

// scheduler 调度器
func (w *ConstraintMonitorWorker) scheduler() {
	defer w.wg.Done()

	checkTicker := time.NewTicker(w.checkInterval)
	alertTicker := time.NewTicker(w.alertInterval)
	defer func() {
		checkTicker.Stop()
		alertTicker.Stop()
	}()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-checkTicker.C:
			w.performConstraintCheck()
		case <-alertTicker.C:
			w.sendAlertSummary()
		}
	}
}

// performConstraintCheck 执行约束检查
func (w *ConstraintMonitorWorker) performConstraintCheck() {
	log.Println("Starting constraint check...")

	startTime := time.Now()
	violations, err := w.constraintMonitor.CheckConstraints(w.ctx)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Failed to perform constraint check: %v", err)
		return
	}

	log.Printf("Constraint check completed in %v, found %d violations", duration, len(violations))

	// 报告所有违反
	for _, violation := range violations {
		if err := w.constraintMonitor.ReportViolation(violation); err != nil {
			log.Printf("Failed to report violation %s: %v", violation.ID, err)
		}
	}

	// 获取指标
	metrics, err := w.constraintMonitor.GetConstraintMetrics(w.ctx)
	if err != nil {
		log.Printf("Failed to get constraint metrics: %v", err)
	} else {
		log.Printf("Constraint metrics - Total: %d, Active: %d, Critical: %d",
			metrics.TotalViolations,
			metrics.ActiveViolations,
			metrics.BySeverity["critical"])
	}

	// 自动解决可解决的违反
	if len(violations) > 0 {
		w.performAutoResolve()
	}
}

// sendAlertSummary 发送告警汇总
func (w *ConstraintMonitorWorker) sendAlertSummary() {
	log.Println("Sending constraint alert summary...")

	stats, err := w.constraintMonitor.GetViolationStats(w.ctx)
	if err != nil {
		log.Printf("Failed to get violation stats: %v", err)
		return
	}

	if stats.TotalCount > 0 {
		summary := map[string]interface{}{
			"type":              "constraint_alert_summary",
			"timestamp":         time.Now(),
			"total_violations":  stats.TotalCount,
			"critical_count":    stats.CriticalCount,
			"warning_count":     stats.WarningCount,
			"info_count":        stats.InfoCount,
			"top_tenants":       w.getTopViolatingTenants(stats.ByTenant),
		}

		log.Printf("Constraint Alert Summary: %+v", summary)

		// 实际实现可以发送邮件、短信或集成到告警系统
	}
}

// getTopViolatingTenants 获取违反最多的租户
func (w *ConstraintMonitorWorker) getTopViolatingTenants(byTenant map[string]int) []map[string]interface{} {
	type tenantCount struct {
		TenantID string
		Count    int
	}

	var counts []tenantCount
	for tenantID, count := range byTenant {
		counts = append(counts, tenantCount{TenantID: tenantID, Count: count})
	}

	// 排序并取前5
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[i].Count < counts[j].Count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	var top []map[string]interface{}
	for i := 0; i < len(counts) && i < 5; i++ {
		top = append(top, map[string]interface{}{
			"tenant_id": counts[i].TenantID,
			"count":     counts[i].Count,
		})
	}

	return top
}

// performAutoResolve 执行自动解决
func (w *ConstraintMonitorWorker) performAutoResolve() {
	resolvedCount, err := w.constraintMonitor.AutoResolve(w.ctx)
	if err != nil {
		log.Printf("Failed to auto-resolve violations: %v", err)
	} else if resolvedCount > 0 {
		log.Printf("Auto-resolved %d violations", resolvedCount)
	}
}

// TriggerCheck 手动触发检查
func (w *ConstraintMonitorWorker) TriggerCheck() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.performConstraintCheck()
	}()
}

// GetMetrics 获取当前指标
func (w *ConstraintMonitorWorker) GetMetrics() (*service.ConstraintMetrics, error) {
	return w.constraintMonitor.GetConstraintMetrics(w.ctx)
}

// GetStats 获取当前统计
func (w *ConstraintMonitorWorker) GetStats() (*service.ViolationStats, error) {
	return w.constraintMonitor.GetViolationStats(w.ctx)
}
