package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/constants"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
)

// ConstraintViolation 约束违反记录
type ConstraintViolation struct {
	ID            uuid.UUID              `json:"id"`
	TenantID      *uuid.UUID             `json:"tenant_id,omitempty"`
	EnvironmentID *uuid.UUID             `json:"environment_id,omitempty"`
	ApplicationID *uuid.UUID             `json:"application_id,omitempty"`
	ViolationType string                 `json:"violation_type"`
	Message       string                 `json:"message"`
	Severity      string                 `json:"severity"` // info/warning/critical
	Details       map[string]interface{} `json:"details"`
	DetectedAt    time.Time              `json:"detected_at"`
	Resolved      bool                   `json:"resolved"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
}

// ConstraintMetrics 约束指标
type ConstraintMetrics struct {
	TotalViolations   int64            `json:"total_violations"`
	ActiveViolations  int64            `json:"active_violations"`
	ResolvedToday     int64            `json:"resolved_today"`
	ByType            map[string]int64 `json:"by_type"`
	BySeverity        map[string]int64 `json:"by_severity"`
	ByTenant          map[string]int64 `json:"by_tenant"`
	AvgResolutionTime float64          `json:"avg_resolution_time_seconds"`
	Trend             []MetricPoint    `json:"trend"`
}

// MetricPoint 指标趋势点
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// ViolationStats 违反统计
type ViolationStats struct {
	TotalCount    int            `json:"total_count"`
	CriticalCount int            `json:"critical_count"`
	WarningCount  int            `json:"warning_count"`
	InfoCount     int            `json:"info_count"`
	ByTenant      map[string]int `json:"by_tenant"`
	RecentTrends  []MetricPoint  `json:"recent_trends"`
}

// ConstraintMonitor 约束监控器
type ConstraintMonitor struct {
	tenantRepo      *repository.TenantRepository
	environmentRepo *repository.EnvironmentRepository
	applicationRepo *repository.ApplicationRepository
	quotaRepo       *repository.QuotaRepository
	db              *gorm.DB
	alertThresholds map[string]int
}

// NewConstraintMonitor 创建约束监控器
func NewConstraintMonitor(
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	db *gorm.DB,
) *ConstraintMonitor {
	return &ConstraintMonitor{
		tenantRepo:      tenantRepo,
		environmentRepo: environmentRepo,
		applicationRepo: applicationRepo,
		quotaRepo:       quotaRepo,
		db:              db,
		alertThresholds: map[string]int{
			"max_violations_per_hour":       100,
			"max_critical_violations":       10,
			"max_violations_per_tenant":     50,
			"avg_resolution_time_threshold": 3600, // 1小时
		},
	}
}

// CheckConstraints 检查所有约束
func (m *ConstraintMonitor) CheckConstraints(ctx context.Context) ([]ConstraintViolation, error) {
	violations := make([]ConstraintViolation, 0)

	// 1. 检查租户配额约束
	tenantViolations, err := m.checkTenantQuotaConstraints(ctx)
	if err != nil {
		log.Printf("Failed to check tenant quota constraints: %v", err)
	} else {
		violations = append(violations, tenantViolations...)
	}

	// 2. 检查环境配额约束
	envViolations, err := m.checkEnvironmentQuotaConstraints(ctx)
	if err != nil {
		log.Printf("Failed to check environment quota constraints: %v", err)
	} else {
		violations = append(violations, envViolations...)
	}

	// 3. 检查应用配额约束
	appViolations, err := m.checkApplicationQuotaConstraints(ctx)
	if err != nil {
		log.Printf("Failed to check application quota constraints: %v", err)
	} else {
		violations = append(violations, appViolations...)
	}

	// 4. 检查层级约束
	hierarchyViolations, err := m.checkHierarchyConstraints(ctx)
	if err != nil {
		log.Printf("Failed to check hierarchy constraints: %v", err)
	} else {
		violations = append(violations, hierarchyViolations...)
	}

	return violations, nil
}

// checkTenantQuotaConstraints 检查租户配额约束
func (m *ConstraintMonitor) checkTenantQuotaConstraints(ctx context.Context) ([]ConstraintViolation, error) {
	violations := make([]ConstraintViolation, 0)

	// 获取所有租户
	tenants, err := m.tenantRepo.ListAll()
	if err != nil {
		return nil, err
	}

	for _, tenant := range tenants {
		// 获取租户配额
		tenantQuota, err := m.quotaRepo.GetTenantQuotaByTenantID(tenant.ID.String())
		if err != nil {
			continue
		}

		// 检查配额超限
		if exceeded, details := m.checkQuotaExceeded(tenantQuota.HardLimits, tenantQuota.Allocated); exceeded {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				TenantID:      &tenant.ID,
				ViolationType: "tenant_quota_exceeded",
				Message:       fmt.Sprintf("租户 %s 配额超限", tenant.Name),
				Severity:      "critical",
				Details:       details,
				DetectedAt:    time.Now(),
				Resolved:      false,
			})
		}

		// 检查可用配额不足
		if insufficient, details := m.checkQuotaInsufficient(tenantQuota.Available); insufficient {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				TenantID:      &tenant.ID,
				ViolationType: "tenant_quota_insufficient",
				Message:       fmt.Sprintf("租户 %s 可用配额不足", tenant.Name),
				Severity:      "warning",
				Details:       details,
				DetectedAt:    time.Now(),
				Resolved:      false,
			})
		}
	}

	return violations, nil
}

// checkEnvironmentQuotaConstraints 检查环境配额约束
func (m *ConstraintMonitor) checkEnvironmentQuotaConstraints(ctx context.Context) ([]ConstraintViolation, error) {
	violations := make([]ConstraintViolation, 0)

	// 获取所有环境
	environments, _, err := m.environmentRepo.ListAll(0, 0)
	if err != nil {
		return nil, err
	}

	for _, env := range environments {
		// 获取环境配额
		resourceQuota, err := m.quotaRepo.GetResourceQuotaByEnvironmentID(env.ID.String())
		if err != nil {
			continue
		}

		// 检查配额使用率过高
		if highUsage, details := m.checkQuotaHighUsage(resourceQuota.HardLimits, resourceQuota.Used); highUsage {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				EnvironmentID: &env.ID,
				ViolationType: "environment_quota_high_usage",
				Message:       fmt.Sprintf("环境 %s 配额使用率过高", env.Namespace),
				Severity:      "warning",
				Details:       details,
				DetectedAt:    time.Now(),
				Resolved:      false,
			})
		}

		// 检查配额状态异常
		if resourceQuota.Status != constants.StatusActive {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				EnvironmentID: &env.ID,
				ViolationType: "environment_quota_status_error",
				Message:       fmt.Sprintf("环境 %s 配额状态异常: %s", env.Namespace, resourceQuota.Status),
				Severity:      "critical",
				Details:       map[string]interface{}{"status": resourceQuota.Status},
				DetectedAt:    time.Now(),
				Resolved:      false,
			})
		}
	}

	return violations, nil
}

// checkApplicationQuotaConstraints 检查应用配额约束
func (m *ConstraintMonitor) checkApplicationQuotaConstraints(ctx context.Context) ([]ConstraintViolation, error) {
	violations := make([]ConstraintViolation, 0)

	// 获取所有应用
	applications, err := m.applicationRepo.ListAll()
	if err != nil {
		return nil, err
	}

	for _, app := range applications {
		// 获取应用资源规格
		resourceSpec, err := m.quotaRepo.GetApplicationResourceSpecByApplicationID(app.ID.String())
		if err != nil {
			continue
		}

		// 检查副本数超限
		if resourceSpec.CurrentReplicas > resourceSpec.MaxReplicas {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				ApplicationID: &app.ID,
				ViolationType: "application_replica_exceeded",
				Message:       fmt.Sprintf("应用 %s 副本数超限: %d/%d", app.Name, resourceSpec.CurrentReplicas, resourceSpec.MaxReplicas),
				Severity:      "critical",
				Details: map[string]interface{}{
					"current_replicas": resourceSpec.CurrentReplicas,
					"max_replicas":     resourceSpec.MaxReplicas,
				},
				DetectedAt: time.Now(),
				Resolved:   false,
			})
		}
	}

	return violations, nil
}

// checkHierarchyConstraints 检查层级约束
func (m *ConstraintMonitor) checkHierarchyConstraints(ctx context.Context) ([]ConstraintViolation, error) {
	violations := make([]ConstraintViolation, 0)

	// 检查孤立的环境（没有关联租户）
	orphanedEnvs, err := m.environmentRepo.FindOrphanedEnvironments()
	if err != nil {
		log.Printf("Failed to find orphaned environments: %v", err)
	} else {
		for _, env := range orphanedEnvs {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				EnvironmentID: &env.ID,
				ViolationType: "orphaned_environment",
				Message:       fmt.Sprintf("环境 %s 没有关联租户", env.Namespace),
				Severity:      "critical",
				Details:       map[string]interface{}{"namespace": env.Namespace},
				DetectedAt:    time.Now(),
				Resolved:      false,
			})
		}
	}

	// 检查孤立的应用（没有关联环境）
	orphanedApps, err := m.applicationRepo.FindOrphanedApplications()
	if err != nil {
		log.Printf("Failed to find orphaned applications: %v", err)
	} else {
		for _, app := range orphanedApps {
			violations = append(violations, ConstraintViolation{
				ID:            uuid.New(),
				ApplicationID: &app.ID,
				ViolationType: "orphaned_application",
				Message:       fmt.Sprintf("应用 %s 没有关联环境", app.Name),
				Severity:      "critical",
				Details:       map[string]interface{}{"application_name": app.Name},
				DetectedAt:    time.Now(),
				Resolved:      false,
			})
		}
	}

	return violations, nil
}

// GetConstraintMetrics 获取约束指标
func (m *ConstraintMonitor) GetConstraintMetrics(ctx context.Context) (*ConstraintMetrics, error) {
	metrics := &ConstraintMetrics{
		ByType:     make(map[string]int64),
		BySeverity: make(map[string]int64),
		ByTenant:   make(map[string]int64),
		Trend:      make([]MetricPoint, 0),
	}

	// 获取违反记录
	violations, err := m.GetAllViolations(ctx)
	if err != nil {
		return nil, err
	}

	// 计算总违反数
	metrics.TotalViolations = int64(len(violations))

	// 计算活跃违反数
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	metrics.ResolvedToday = 0

	for _, v := range violations {
		if !v.Resolved {
			metrics.ActiveViolations++
		}
		if v.Resolved && v.ResolvedAt != nil && v.ResolvedAt.After(today) {
			metrics.ResolvedToday++
		}

		// 按类型统计
		metrics.ByType[v.ViolationType]++

		// 按严重程度统计
		metrics.BySeverity[v.Severity]++

		// 按租户统计
		if v.TenantID != nil {
			tenantID := v.TenantID.String()
			metrics.ByTenant[tenantID]++
		}
	}

	// 计算趋势数据（最近24小时）
	metrics.Trend = m.calculateTrendData(violations)

	return metrics, nil
}

// GetAllViolations 获取所有违反记录
func (m *ConstraintMonitor) GetAllViolations(ctx context.Context) ([]ConstraintViolation, error) {
	// 从数据库查询违反记录
	var dbViolations []struct {
		ID            uuid.UUID
		TenantID      *uuid.UUID
		EnvironmentID *uuid.UUID
		ApplicationID *uuid.UUID
		ViolationType string
		Message       string
		Severity      string
		Details       string
		DetectedAt    time.Time
		Resolved      bool
		ResolvedAt    *time.Time
	}

	// 这里应该查询数据库，简化实现
	// 实际实现需要创建 constraint_violations 表
	// err := m.db.Model(&ConstraintViolation{}).Find(&dbViolations).Error

	// 转换并返回
	violations := make([]ConstraintViolation, 0)
	for _, dbV := range dbViolations {
		var details map[string]interface{}
		if dbV.Details != "" {
			json.Unmarshal([]byte(dbV.Details), &details)
		}
		violations = append(violations, ConstraintViolation{
			ID:            dbV.ID,
			TenantID:      dbV.TenantID,
			EnvironmentID: dbV.EnvironmentID,
			ApplicationID: dbV.ApplicationID,
			ViolationType: dbV.ViolationType,
			Message:       dbV.Message,
			Severity:      dbV.Severity,
			Details:       details,
			DetectedAt:    dbV.DetectedAt,
			Resolved:      dbV.Resolved,
			ResolvedAt:    dbV.ResolvedAt,
		})
	}

	return violations, nil
}

// GetViolationStats 获取违反统计
func (m *ConstraintMonitor) GetViolationStats(ctx context.Context) (*ViolationStats, error) {
	stats := &ViolationStats{
		ByTenant:     make(map[string]int),
		RecentTrends: make([]MetricPoint, 0),
	}

	violations, err := m.GetAllViolations(ctx)
	if err != nil {
		return nil, err
	}

	// 统计各种严重程度
	for _, v := range violations {
		stats.TotalCount++
		switch v.Severity {
		case "critical":
			stats.CriticalCount++
		case "warning":
			stats.WarningCount++
		case "info":
			stats.InfoCount++
		}

		// 按租户统计
		if v.TenantID != nil {
			tenantID := v.TenantID.String()
			stats.ByTenant[tenantID]++
		}
	}

	// 计算最近趋势（每小时）
	stats.RecentTrends = m.calculateHourlyTrend(violations)

	return stats, nil
}

// ReportViolation 报告违反
func (m *ConstraintMonitor) ReportViolation(violation ConstraintViolation) error {
	// 保存到数据库
	// 实际实现需要使用数据库操作
	log.Printf("Constraint violation reported: %s - %s", violation.ViolationType, violation.Message)

	// 检查是否需要告警
	if m.shouldAlert(violation) {
		m.sendAlert(violation)
	}

	return nil
}

// shouldAlert 判断是否需要告警
func (m *ConstraintMonitor) shouldAlert(violation ConstraintViolation) bool {
	// 严重违反总是需要告警
	if violation.Severity == "critical" {
		return true
	}

	// 检查告警阈值
	metrics, err := m.GetConstraintMetrics(context.Background())
	if err != nil {
		return false
	}

	// 违反数量超过阈值
	if metrics.ActiveViolations > int64(m.alertThresholds["max_violations_per_hour"]) {
		return true
	}

	// 特定租户违反过多
	if violation.TenantID != nil {
		tenantID := violation.TenantID.String()
		if count, ok := metrics.ByTenant[tenantID]; ok {
			if count > int64(m.alertThresholds["max_violations_per_tenant"]) {
				return true
			}
		}
	}

	return false
}

// sendAlert 发送告警
func (m *ConstraintMonitor) sendAlert(violation ConstraintViolation) {
	alert := map[string]interface{}{
		"type":      "constraint_violation",
		"severity":  violation.Severity,
		"message":   violation.Message,
		"timestamp": violation.DetectedAt,
		"details":   violation.Details,
	}

	alertJSON, _ := json.Marshal(alert)
	log.Printf("ALERT: %s", string(alertJSON))

	// 实际实现可以集成告警系统（如Prometheus Alertmanager、钉钉、企微等）
}

// ResolveViolation 解决违反
func (m *ConstraintMonitor) ResolveViolation(violationID uuid.UUID) error {
	// 更新数据库
	// 实际实现需要更新数据库操作
	log.Printf("Resolving constraint violation: %s", violationID.String())

	return nil
}

// checkQuotaExceeded 检查配额是否超限
func (m *ConstraintMonitor) checkQuotaExceeded(hardLimits, allocated map[string]interface{}) (bool, map[string]interface{}) {
	exceeded := false
	details := make(map[string]interface{})
	exceededResources := make([]string, 0)

	for resource, limit := range hardLimits {
		if alloc, ok := allocated[resource]; ok {
			if limitFloat, limitOk := limit.(float64); limitOk {
				if allocFloat, allocOk := alloc.(float64); allocOk {
					if allocFloat > limitFloat {
						exceeded = true
						exceededResources = append(exceededResources, resource)
					}
				}
			}
		}
	}

	if exceeded {
		details["exceeded_resources"] = exceededResources
	}

	return exceeded, details
}

// checkQuotaInsufficient 检查配额是否不足
func (m *ConstraintMonitor) checkQuotaInsufficient(available map[string]interface{}) (bool, map[string]interface{}) {
	insufficient := false
	details := make(map[string]interface{})
	lowResources := make([]string, 0)

	for resource, avail := range available {
		if availFloat, ok := avail.(float64); ok {
			// 如果可用配额少于10%，认为不足
			if availFloat < 0.1 {
				insufficient = true
				lowResources = append(lowResources, resource)
			}
		}
	}

	if insufficient {
		details["low_resources"] = lowResources
	}

	return insufficient, details
}

// checkQuotaHighUsage 检查配额使用率是否过高
func (m *ConstraintMonitor) checkQuotaHighUsage(hardLimits, used map[string]interface{}) (bool, map[string]interface{}) {
	highUsage := false
	details := make(map[string]interface{})
	highUsageResources := make([]string, 0)

	for resource, limit := range hardLimits {
		if use, ok := used[resource]; ok {
			if limitFloat, limitOk := limit.(float64); limitOk {
				if useFloat, useOk := use.(float64); useOk && limitFloat > 0 {
					usagePercent := (useFloat / limitFloat) * 100
					if usagePercent > 80 {
						highUsage = true
						highUsageResources = append(highUsageResources, resource)
					}
				}
			}
		}
	}

	if highUsage {
		details["high_usage_resources"] = highUsageResources
	}

	return highUsage, details
}

// calculateTrendData 计算趋势数据
func (m *ConstraintMonitor) calculateTrendData(violations []ConstraintViolation) []MetricPoint {
	trend := make([]MetricPoint, 0)
	now := time.Now()

	// 按小时统计最近24小时
	for i := 23; i >= 0; i-- {
		hourStart := now.Add(-time.Duration(i) * time.Hour)
		hourEnd := hourStart.Add(time.Hour)

		count := int64(0)
		for _, v := range violations {
			if v.DetectedAt.After(hourStart) && v.DetectedAt.Before(hourEnd) {
				count++
			}
		}

		trend = append(trend, MetricPoint{
			Timestamp: hourStart,
			Value:     count,
			Label:     hourStart.Format("15:04"),
		})
	}

	return trend
}

// calculateHourlyTrend 计算每小时趋势
func (m *ConstraintMonitor) calculateHourlyTrend(violations []ConstraintViolation) []MetricPoint {
	return m.calculateTrendData(violations)
}

// AutoResolve 自动解决违反
func (m *ConstraintMonitor) AutoResolve(ctx context.Context) (int, error) {
	resolvedCount := 0
	violations, err := m.GetAllViolations(ctx)
	if err != nil {
		return 0, err
	}

	for _, v := range violations {
		if v.Resolved {
			continue
		}

		// 检查是否可以自动解决
		if m.canAutoResolve(v) {
			if err := m.ResolveViolation(v.ID); err == nil {
				resolvedCount++
			}
		}
	}

	return resolvedCount, nil
}

// canAutoResolve 判断是否可以自动解决
func (m *ConstraintMonitor) canAutoResolve(violation ConstraintViolation) bool {
	// 只有信息级别的违反可以自动解决
	if violation.Severity != "info" {
		return false
	}

	// 检查是否超过自动解决时间（24小时）
	if time.Since(violation.DetectedAt) > 24*time.Hour {
		return true
	}

	return false
}
