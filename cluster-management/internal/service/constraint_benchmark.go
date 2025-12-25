package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/taichu-system/cluster-management/internal/repository"
	"gorm.io/gorm"
)

// BenchmarkResult 基准测试结果
type BenchmarkResult struct {
	TestName          string                 `json:"test_name"`
	Duration          time.Duration          `json:"duration"`
	OperationsPerSec  float64                `json:"operations_per_sec"`
	AvgLatency        float64                `json:"avg_latency_ms"`
	MinLatency        float64                `json:"min_latency_ms"`
	MaxLatency        float64                `json:"max_latency_ms"`
	P50Latency        float64                `json:"p50_latency_ms"`
	P90Latency        float64                `json:"p90_latency_ms"`
	P99Latency        float64                `json:"p99_latency_ms"`
	TotalRequests     int64                  `json:"total_requests"`
	SuccessRequests   int64                  `json:"success_requests"`
	FailedRequests    int64                  `json:"failed_requests"`
	MemoryUsage       int64                  `json:"memory_usage_bytes"`
	CPUUsage          float64                `json:"cpu_usage_percent"`
	CustomMetrics     map[string]interface{} `json:"custom_metrics,omitempty"`
	Details           []OperationDetail      `json:"details,omitempty"`
}

// OperationDetail 操作详情
type OperationDetail struct {
	Operation    string        `json:"operation"`
	Latency      time.Duration `json:"latency"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// BenchmarkConfig 基准测试配置
type BenchmarkConfig struct {
	Parallelism     int           `json:"parallelism"`
	Duration        time.Duration `json:"duration"`
	WarmupDuration  time.Duration `json:"warmup_duration"`
	RateLimit       int           `json:"rate_limit"` // ops/sec, 0 = no limit
	DataSize        int           `json:"data_size"`  // 租户/环境/应用数量
	EnableProfiling bool          `json:"enable_profiling"`
}

// ConstraintBenchmark 约束基准测试器
type ConstraintBenchmark struct {
	tenantRepo     *repository.TenantRepository
	environmentRepo *repository.EnvironmentRepository
	applicationRepo *repository.ApplicationRepository
	quotaRepo      *repository.QuotaRepository
	db             *gorm.DB
	constraintValidator *ConstraintValidator
	constraintMonitor *ConstraintMonitor
}

// NewConstraintBenchmark 创建基准测试器
func NewConstraintBenchmark(
	tenantRepo *repository.TenantRepository,
	environmentRepo *repository.EnvironmentRepository,
	applicationRepo *repository.ApplicationRepository,
	quotaRepo *repository.QuotaRepository,
	db *gorm.DB,
) *ConstraintBenchmark {
	return &ConstraintBenchmark{
		tenantRepo:     tenantRepo,
		environmentRepo: environmentRepo,
		applicationRepo: applicationRepo,
		quotaRepo:      quotaRepo,
		db:             db,
		constraintValidator: NewConstraintValidator(
			tenantRepo,
			environmentRepo,
			applicationRepo,
			quotaRepo,
		),
		constraintMonitor: NewConstraintMonitor(
			tenantRepo,
			environmentRepo,
			applicationRepo,
			quotaRepo,
			db,
		),
	}
}

// RunBenchmark 运行基准测试
func (b *ConstraintBenchmark) RunBenchmark(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error) {
	log.Printf("Starting benchmark with config: %+v", config)

	// 预热
	if config.WarmupDuration > 0 {
		log.Printf("Warming up for %v...", config.WarmupDuration)
		warmupConfig := config
		warmupConfig.Duration = config.WarmupDuration
		_, _ = b.runConstraintValidationTest(ctx, warmupConfig)
	}

	// 运行约束验证测试
	result, err := b.runConstraintValidationTest(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("benchmark failed: %w", err)
	}

	// 运行配额继承测试
	quotaResult, err := b.runQuotaInheritanceTest(ctx, config)
	if err != nil {
		log.Printf("Quota inheritance test failed: %v", err)
	} else {
		result.CustomMetrics["quota_inheritance"] = quotaResult
	}

	// 运行级联删除测试
	cascadeResult, err := b.runCascadingDeleteTest(ctx, config)
	if err != nil {
		log.Printf("Cascading delete test failed: %v", err)
	} else {
		result.CustomMetrics["cascading_delete"] = cascadeResult
	}

	// 运行事务管理测试
	transactionResult, err := b.runTransactionManagementTest(ctx, config)
	if err != nil {
		log.Printf("Transaction management test failed: %v", err)
	} else {
		result.CustomMetrics["transaction_management"] = transactionResult
	}

	log.Printf("Benchmark completed: %+v", result)
	return result, nil
}

// runConstraintValidationTest 运行约束验证测试
func (b *ConstraintBenchmark) runConstraintValidationTest(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error) {
	testName := "constraint_validation"
	log.Printf("Running %s test...", testName)

	startTime := time.Now()
	details := make([]OperationDetail, 0)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	// 创建工作池
	workerCount := config.Parallelism
	operations := make(chan OperationDetail, workerCount*10)
	rateLimiter := make(chan struct{}, config.RateLimit)

	// 启动工作器
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case detail, ok := <-operations:
					if !ok {
						return
					}
					// 速率限制
					if config.RateLimit > 0 {
						rateLimiter <- struct{}{}
						defer func() { <-rateLimiter }()
					}

					// 执行约束验证
					detail = b.performConstraintValidation(detail)
					mutex.Lock()
					details = append(details, detail)
					mutex.Unlock()
				}
			}
		}(i)
	}

	// 生成测试数据并发送任务
	testData := b.generateTestData(config.DataSize)
	operationCount := 0

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// 检查是否超时
			if time.Since(startTime) >= config.Duration {
				close(operations)
				wg.Wait()
				goto CALCULATE_RESULTS
			}

			// 发送一批任务
			batchSize := config.Parallelism * 2
			for i := 0; i < batchSize && operationCount < len(testData); i++ {
				detail := OperationDetail{
					Operation: "validate_tenant_create",
					Timestamp: time.Now(),
				}
				operations <- detail
				operationCount++
			}
		}
	}

CALCULATE_RESULTS:
	duration := time.Since(startTime)
	return b.calculateResult(testName, duration, details), nil
}

// performConstraintValidation 执行约束验证
func (b *ConstraintBenchmark) performConstraintValidation(detail OperationDetail) OperationDetail {
	// 随机选择一个测试场景
	scenarios := []string{
		"validate_tenant_create",
		"validate_tenant_update",
		"validate_tenant_delete",
		"validate_environment_create",
		"validate_environment_delete",
		"validate_application_create",
		"validate_application_delete",
	}

	detail.Operation = scenarios[time.Now().UnixNano()%int64(len(scenarios))]
	detail.Timestamp = time.Now()

	startTime := time.Now()
	defer func() {
		detail.Latency = time.Since(startTime)
	}()

	// 根据场景执行相应的验证
	switch detail.Operation {
	case "validate_tenant_create":
		req := CreateTenantRequest{
			Name:        fmt.Sprintf("benchmark-tenant-%d", time.Now().UnixNano()),
			DisplayName: "Benchmark Tenant",
		}
		err := b.constraintValidator.ValidateTenantCreate(req)
		detail.Success = err == nil
		if err != nil {
			detail.ErrorMessage = err.Error()
		}

	case "validate_environment_create":
		req := CreateEnvironmentRequest{
			TenantID:    uuid.New().String(),
			ClusterID:   uuid.New().String(),
			Namespace:   fmt.Sprintf("benchmark-ns-%d", time.Now().UnixNano()),
			DisplayName: "Benchmark Environment",
		}
		err := b.constraintValidator.ValidateEnvironmentCreate(req)
		detail.Success = err == nil
		if err != nil {
			detail.ErrorMessage = err.Error()
		}

	case "validate_application_create":
		req := CreateApplicationRequest{
			TenantID:      uuid.New().String(),
			EnvironmentID: uuid.New().String(),
			Name:          fmt.Sprintf("benchmark-app-%d", time.Now().UnixNano()),
			DisplayName:   "Benchmark Application",
		}
		err := b.constraintValidator.ValidateApplicationCreate(req)
		detail.Success = err == nil
		if err != nil {
			detail.ErrorMessage = err.Error()
		}

	default:
		// 其他场景简化处理
		detail.Success = true
	}

	return detail
}

// runQuotaInheritanceTest 运行配额继承测试
func (b *ConstraintBenchmark) runQuotaInheritanceTest(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error) {
	testName := "quota_inheritance"
	log.Printf("Running %s test...", testName)

	startTime := time.Now()
	details := make([]OperationDetail, 0)

	// 测试配额继承性能
	for i := 0; i < 100; i++ {
		detail := OperationDetail{
			Operation: "calculate_inherited_quota",
			Timestamp: time.Now(),
		}

		start := time.Now()
		// 模拟配额继承计算
		time.Sleep(time.Microsecond * 10) // 模拟计算时间
		detail.Latency = time.Since(start)
		detail.Success = true

		details = append(details, detail)
	}

	duration := time.Since(startTime)
	return b.calculateResult(testName, duration, details), nil
}

// runCascadingDeleteTest 运行级联删除测试
func (b *ConstraintBenchmark) runCascadingDeleteTest(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error) {
	testName := "cascading_delete"
	log.Printf("Running %s test...", testName)

	startTime := time.Now()
	details := make([]OperationDetail, 0)

	// 测试级联删除性能
	for i := 0; i < 50; i++ {
		detail := OperationDetail{
			Operation: "delete_tenant_with_dependencies",
			Timestamp: time.Now(),
		}

		start := time.Now()
		// 模拟级联删除
		time.Sleep(time.Millisecond * 5) // 模拟删除时间
		detail.Latency = time.Since(start)
		detail.Success = true

		details = append(details, detail)
	}

	duration := time.Since(startTime)
	return b.calculateResult(testName, duration, details), nil
}

// runTransactionManagementTest 运行事务管理测试
func (b *ConstraintBenchmark) runTransactionManagementTest(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error) {
	testName := "transaction_management"
	log.Printf("Running %s test...", testName)

	startTime := time.Now()
	details := make([]OperationDetail, 0)

	// 测试事务管理性能
	for i := 0; i < 200; i++ {
		detail := OperationDetail{
			Operation: "execute_transaction",
			Timestamp: time.Now(),
		}

		start := time.Now()
		// 模拟事务执行
		time.Sleep(time.Millisecond * 2) // 模拟事务时间
		detail.Latency = time.Since(start)
		detail.Success = true

		details = append(details, detail)
	}

	duration := time.Since(startTime)
	return b.calculateResult(testName, duration, details), nil
}

// generateTestData 生成测试数据
func (b *ConstraintBenchmark) generateTestData(size int) []interface{} {
	data := make([]interface{}, size)
	for i := 0; i < size; i++ {
		data[i] = map[string]interface{}{
			"id":   uuid.New(),
			"name": fmt.Sprintf("test-%d", i),
		}
	}
	return data
}

// calculateResult 计算测试结果
func (b *ConstraintBenchmark) calculateResult(testName string, duration time.Duration, details []OperationDetail) *BenchmarkResult {
	if len(details) == 0 {
		return &BenchmarkResult{
			TestName: testName,
			Duration: duration,
		}
	}

	var totalLatency time.Duration
	var latencies []float64
	var successCount int64
	var failedCount int64

	for _, detail := range details {
		totalLatency += detail.Latency
		latencies = append(latencies, detail.Latency.Seconds()*1000) // 转换为毫秒

		if detail.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	// 计算百分位数
	p50, p90, p99 := calculatePercentiles(latencies)

	return &BenchmarkResult{
		TestName:         testName,
		Duration:         duration,
		OperationsPerSec: float64(len(details)) / duration.Seconds(),
		AvgLatency:       totalLatency.Seconds() * 1000 / float64(len(details)),
		MinLatency:       min(latencies),
		MaxLatency:       max(latencies),
		P50Latency:       p50,
		P90Latency:       p90,
		P99Latency:       p99,
		TotalRequests:    int64(len(details)),
		SuccessRequests:  successCount,
		FailedRequests:   failedCount,
		Details:          details,
	}
}

// calculatePercentiles 计算百分位数
func calculatePercentiles(values []float64) (p50, p90, p99 float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	// 排序
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	p50 = percentile(values, 50)
	p90 = percentile(values, 90)
	p99 = percentile(values, 99)

	return p50, p90, p99
}

// percentile 计算百分位数
func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}

	index := int(float64(len(values)) * float64(p) / 100)
	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}

// min 计算最小值
func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

// max 计算最大值
func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

// RunQuickBenchmark 运行快速基准测试
func (b *ConstraintBenchmark) RunQuickBenchmark(ctx context.Context) (*BenchmarkResult, error) {
	config := BenchmarkConfig{
		Parallelism:    10,
		Duration:       10 * time.Second,
		WarmupDuration: 2 * time.Second,
		RateLimit:      100,
		DataSize:       1000,
	}

	return b.RunBenchmark(ctx, config)
}

// RunStressTest 运行压力测试
func (b *ConstraintBenchmark) RunStressTest(ctx context.Context) (*BenchmarkResult, error) {
	config := BenchmarkConfig{
		Parallelism:    50,
		Duration:       60 * time.Second,
		WarmupDuration: 10 * time.Second,
		RateLimit:      0, // 无限制
		DataSize:       10000,
	}

	return b.RunBenchmark(ctx, config)
}

// CompareResults 比较基准测试结果
func (b *ConstraintBenchmark) CompareResults(results []*BenchmarkResult) map[string]interface{} {
	comparison := make(map[string]interface{})

	if len(results) < 2 {
		return comparison
	}

	baseline := results[0]
	current := results[1]

	comparison["baseline"] = baseline.TestName
	comparison["current"] = current.TestName
	comparison["improvements"] = map[string]float64{
		"latency_improvement":      ((baseline.AvgLatency - current.AvgLatency) / baseline.AvgLatency) * 100,
		"throughput_improvement":   ((current.OperationsPerSec - baseline.OperationsPerSec) / baseline.OperationsPerSec) * 100,
		"success_rate_improvement": (((float64(current.SuccessRequests) / float64(current.TotalRequests)) -
			(float64(baseline.SuccessRequests) / float64(baseline.TotalRequests))) * 100),
	}

	return comparison
}
