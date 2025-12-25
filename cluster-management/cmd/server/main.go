package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/config"
	"github.com/taichu-system/cluster-management/internal/handler"
	"github.com/taichu-system/cluster-management/internal/middleware"
	"github.com/taichu-system/cluster-management/internal/repository"
	"github.com/taichu-system/cluster-management/internal/service"
	"github.com/taichu-system/cluster-management/internal/service/worker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Database.AutoCreateDB {
		log.Printf("Auto-creating database '%s' if not exists...", cfg.Database.DBName)
		if err := createDatabase(cfg); err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
	}

	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if cfg.Database.AutoMigrate {
		log.Println("Running database migration...")
		if err := autoMigrate(db); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}
	}

	encryptionService, err := service.NewEncryptionService(cfg.Encryption.Key)
	if err != nil {
		log.Fatalf("Failed to initialize encryption service: %v", err)
	}

	clusterManager := service.NewClusterManager(cfg.ClusterManager.ClientTimeout, cfg.ClusterManager.MaxClients)

	clusterRepo := repository.NewClusterRepository(db)
	stateRepo := repository.NewClusterStateRepository(db)
	nodeRepo := repository.NewNodeRepository(db)
	eventRepo := repository.NewEventRepository(db)
	securityPolicyRepo := repository.NewSecurityPolicyRepository(db)
	autoscalingPolicyRepo := repository.NewAutoscalingPolicyRepository(db)
	backupRepo := repository.NewBackupRepository(db)
	backupScheduleRepo := repository.NewBackupScheduleRepository(db)
	clusterEnvironmentRepo := repository.NewClusterEnvironmentRepository(db)
	importRepo := repository.NewImportRecordRepository(db)
	machineRepo := repository.NewMachineRepository(db)
	createTaskRepo := repository.NewCreateTaskRepository(db)
	clusterResourceRepo := repository.NewClusterResourceRepository(db)

	tenantRepo := repository.NewTenantRepository(db)
	environmentRepo := repository.NewEnvironmentRepository(db)
	applicationRepo := repository.NewApplicationRepository(db)
	quotaRepo := repository.NewQuotaRepository(db)
	classificationRepo := repository.NewResourceClassificationRepository(db)
	alertRepo := repository.NewAlertRepository(db)

	healthCheckWorker := worker.NewHealthCheckWorker(
		clusterRepo,
		stateRepo,
		clusterManager,
		encryptionService,
	)

	resourceClassificationWorker := worker.NewResourceClassificationWorker(
		clusterRepo,
		encryptionService,
		clusterManager,
		tenantRepo,
		environmentRepo,
		applicationRepo,
		quotaRepo,
		classificationRepo,
	)

	resourceSyncWorker := worker.NewResourceSyncWorker(
		clusterRepo,
		nodeRepo,
		eventRepo,
		clusterResourceRepo,
		stateRepo,
		autoscalingPolicyRepo,
		securityPolicyRepo,
		clusterManager,
		encryptionService,
		resourceClassificationWorker,
	)

	log.Printf("Worker.Enabled: %v", cfg.Worker.Enabled)
	if cfg.Worker.Enabled {
		log.Println("Starting health check worker...")
		healthCheckWorker.Start()
		defer healthCheckWorker.Stop()

		log.Println("Starting resource sync worker...")
		if cfg.Worker.UseInformerMode {
			log.Println("Using Informer-based resource sync worker")
			informerResourceSyncWorker := worker.NewInformerResourceSyncWorker(
				clusterRepo,
				nodeRepo,
				eventRepo,
				clusterResourceRepo,
				stateRepo,
				autoscalingPolicyRepo,
				securityPolicyRepo,
				clusterManager,
				encryptionService,
			)
			informerResourceSyncWorker.Start()
			defer informerResourceSyncWorker.Stop()
		} else {
			log.Println("Using traditional polling-based resource sync worker")
			resourceSyncWorker.Start()
			defer resourceSyncWorker.Stop()
		}

		log.Println("Starting resource classification worker...")
		resourceClassificationWorker.Start()
		defer resourceClassificationWorker.Stop()
	} else {
		log.Println("Worker is disabled in configuration")
	}

	auditRepo := repository.NewAuditRepository(db)
	auditService := service.NewAuditService(auditRepo)
	alertService := service.NewAlertService(alertRepo)

	clusterService := service.NewClusterService(
		clusterRepo,
		stateRepo,
		clusterResourceRepo,
		encryptionService,
		clusterManager,
		auditService,
	)

	nodeService := service.NewNodeService(
		clusterManager,
		encryptionService,
		nodeRepo,
		clusterRepo,
	)

	eventService := service.NewEventService(
		clusterManager,
		encryptionService,
		eventRepo,
		clusterRepo,
	)

	// 创建KubernetesService实例
	kubernetesService := service.NewKubernetesService()

	securityPolicyService := service.NewSecurityPolicyService(
		securityPolicyRepo,
		clusterRepo,
		clusterManager,
		encryptionService,
		kubernetesService,
	)

	autoscalingPolicyService := service.NewAutoscalingPolicyService(
		clusterManager,
		encryptionService,
		autoscalingPolicyRepo,
		clusterRepo,
	)

	backupService := service.NewBackupService(
		backupRepo,
		backupScheduleRepo,
		clusterRepo,
		encryptionService,
		clusterManager,
		alertService,
	)

	restoreService := service.NewRestoreService(
		backupRepo,
		backupScheduleRepo,
		clusterRepo,
		encryptionService,
		clusterManager,
	)

	topologyService := service.NewTopologyService(
		clusterRepo,
		stateRepo,
		clusterEnvironmentRepo,
	)

	importService := service.NewImportService(
		importRepo,
		clusterRepo,
		encryptionService,
		clusterManager,
		clusterService,
		tenantRepo,
		environmentRepo,
		applicationRepo,
		quotaRepo,
	)

	expansionRepo := repository.NewExpansionRepository(db)
	expansionService := service.NewExpansionService(
		expansionRepo,
		clusterRepo,
		stateRepo,
		clusterResourceRepo,
	)

	// 创建新服务
	machineService := service.NewMachineService(machineRepo)
	configGenerator := service.NewConfigGenerator()
	createClusterService := service.NewCreateClusterService(
		createTaskRepo,
		machineService,
		configGenerator,
	)

	// 创建认证服务和处理器
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, "your-secret-key", 24*time.Hour)
	authHandler := handler.NewAuthHandler(authService, auditService)

	// 三级分类模型相关服务
	tenantService := service.NewTenantService(tenantRepo, quotaRepo, environmentRepo, auditService)
	environmentService := service.NewEnvironmentService(environmentRepo, quotaRepo, auditService)
	applicationService := service.NewApplicationService(applicationRepo, quotaRepo, auditService)
	kubeClient, _ := kubernetesService.GetClientset("")
	quotaService := service.NewQuotaService(kubeClient, quotaRepo, environmentRepo, tenantRepo)
	constraintValidator := service.NewConstraintValidator(tenantRepo, environmentRepo, applicationRepo, quotaRepo)
	constraintMonitor := service.NewConstraintMonitor(tenantRepo, environmentRepo, applicationRepo, quotaRepo, db)
	quotaInheritance := service.NewQuotaInheritance(tenantRepo, environmentRepo, quotaRepo)

	clusterHandler := handler.NewClusterHandler(
		clusterService,
		encryptionService,
		healthCheckWorker,
		createClusterService,
		nodeService,
		auditService,
	)

	nodeHandler := handler.NewNodeHandler(nodeService)
	eventHandler := handler.NewEventHandler(eventService)
	securityPolicyHandler := handler.NewSecurityPolicyHandler(securityPolicyService)
	autoscalingPolicyHandler := handler.NewAutoscalingPolicyHandler(autoscalingPolicyService)
	backupHandler := handler.NewBackupHandler(backupService, restoreService, auditService)
	topologyHandler := handler.NewTopologyHandler(topologyService)
	importHandler := handler.NewImportHandler(importService, healthCheckWorker, resourceSyncWorker, auditService)
	auditHandler := handler.NewAuditHandler(auditService)
	expansionHandler := handler.NewExpansionHandler(expansionService)
	machineHandler := handler.NewMachineHandler(machineService, auditService)

	// 三级分类模型相关Handler
	tenantHandler := handler.NewTenantHandler(tenantService, constraintValidator)
	environmentHandler := handler.NewEnvironmentHandler(environmentService, constraintValidator, quotaInheritance)
	applicationHandler := handler.NewApplicationHandler(applicationService, constraintValidator)
	constraintHandler := handler.NewConstraintHandler(constraintValidator, constraintMonitor, nil)
	resourceClassificationHandler := handler.NewResourceClassificationHandler(
		clusterRepo,
		tenantRepo,
		environmentRepo,
		applicationRepo,
		quotaRepo,
		resourceClassificationWorker,
		quotaService,
		nil,
	)

	r := setupRoutes(clusterHandler, nodeHandler, eventHandler, securityPolicyHandler, autoscalingPolicyHandler, backupHandler, topologyHandler, importHandler, auditHandler, expansionHandler, machineHandler, authHandler, tenantHandler, environmentHandler, applicationHandler, constraintHandler, resourceClassificationHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s options='%s'",
		cfg.Database.Host,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode,
		cfg.Database.Timezone,
		cfg.Database.Options,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	return db, nil
}

func createDatabase(cfg *config.Config) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=%s TimeZone=%s options='%s'",
		cfg.Database.Host,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Port,
		cfg.Database.SSLMode,
		cfg.Database.Timezone,
		cfg.Database.Options,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres for database creation: %w", err)
	}

	var count int64
	err = db.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", cfg.Database.DBName).Scan(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if count == 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}

		_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8' LC_COLLATE='zh_CN.UTF-8' LC_CTYPE='zh_CN.UTF-8'", cfg.Database.DBName))
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}

		log.Printf("Database '%s' created successfully", cfg.Database.DBName)
	} else {
		log.Printf("Database '%s' already exists, skipping creation", cfg.Database.DBName)
	}

	return nil
}

func autoMigrate(db *gorm.DB) error {
	// 创建migrations表（如果不存在）
	if err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW())").Error; err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// 获取已执行的migration列表
	var executedMigrations []string
	if err := db.Raw("SELECT version FROM schema_migrations").Scan(&executedMigrations).Error; err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}
	executedMap := make(map[string]bool)
	for _, v := range executedMigrations {
		executedMap[v] = true
	}

	// 获取所有migration文件
	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	// 按文件名排序
	sort.Strings(migrationFiles)

	// 执行未执行的migration
	for _, file := range migrationFiles {
		filename := filepath.Base(file)
		if executedMap[filename] {
			continue
		}

		log.Printf("Executing migration: %s", filename)
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// 执行SQL
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		// 记录migration已执行
		if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", filename).Error; err != nil {
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		log.Printf("Successfully executed migration: %s", filename)
	}

	log.Println("All migrations completed successfully!")
	return nil
}

func setupRoutes(
	clusterHandler *handler.ClusterHandler,
	nodeHandler *handler.NodeHandler,
	eventHandler *handler.EventHandler,
	securityPolicyHandler *handler.SecurityPolicyHandler,
	autoscalingPolicyHandler *handler.AutoscalingPolicyHandler,
	backupHandler *handler.BackupHandler,
	topologyHandler *handler.TopologyHandler,
	importHandler *handler.ImportHandler,
	auditHandler *handler.AuditHandler,
	expansionHandler *handler.ExpansionHandler,
	machineHandler *handler.MachineHandler,
	authHandler *handler.AuthHandler,
	// 新增：三级分类模型Handler
	tenantHandler *handler.TenantHandler,
	environmentHandler *handler.EnvironmentHandler,
	applicationHandler *handler.ApplicationHandler,
	constraintHandler *handler.ConstraintHandler,
	resourceClassificationHandler *handler.ResourceClassificationHandler,
) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 认证相关路由（不需要认证）
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.GET("/profile", middleware.JWTMiddleware("your-secret-key"), authHandler.Profile)
		auth.POST("/logout", middleware.JWTMiddleware("your-secret-key"), authHandler.Logout)
		auth.GET("/token", authHandler.GenerateToken) // 测试用，移除了userId参数
	}

	// 需要认证的路由
	v1 := r.Group("/api/v1")
	// v1.Use(middleware.JWTMiddleware("your-secret-key"))
	{
		// 全局审计日志接口（查看所有操作）
		v1.GET("/audit", auditHandler.ListAllAuditEvents)
		v1.POST("/audit", auditHandler.CreateAuditEvent)

		// 集群拓扑接口（不需要cluster ID）
		v1.GET("/clusters/topology", topologyHandler.GetTopology)

		clusters := v1.Group("/clusters")
		{
			clusters.POST("", clusterHandler.CreateCluster)
			clusters.POST("/create", clusterHandler.CreateClusterByMachines)
			clusters.GET("", clusterHandler.ListClusters)
			clusters.GET(":id", clusterHandler.GetCluster)
			clusters.DELETE(":id", clusterHandler.DeleteCluster)

			// 集群导入接口
			clusters.POST("/import", importHandler.ImportCluster)
			clusters.GET("/imports", importHandler.ListImports)

			// 节点相关接口
			nodes := clusters.Group(":id/nodes")
			{
				nodes.GET("", nodeHandler.ListNodes)
				nodes.GET(":nodeName", nodeHandler.GetNode)
			}

			// 事件相关接口
			events := clusters.Group(":id/events")
			{
				events.GET("", eventHandler.ListEvents)
			}

			// 安全策略相关接口
			security := clusters.Group(":id/security-policies")
			{
				security.GET("", securityPolicyHandler.GetSecurityPolicy)
			}

			// 自动伸缩策略相关接口
			autoscaling := clusters.Group(":id/autoscaling-policies")
			{
				autoscaling.GET("", autoscalingPolicyHandler.GetAutoscalingPolicy)
			}

			// 备份相关接口
			backups := clusters.Group(":id/backups")
			{
				backups.POST("", backupHandler.CreateBackup)
				backups.GET("", backupHandler.ListBackups)
				backups.GET(":backupId", backupHandler.GetBackup)
				backups.POST(":backupId/restore", backupHandler.RestoreBackup)
				backups.GET(":backupId/restore/:restoreId", backupHandler.GetRestoreProgress)
				backups.DELETE(":backupId", backupHandler.DeleteBackup)
			}

			// 独立的etcd备份接口
			etcdBackups := clusters.Group(":id/etcd/backups")
			{
				etcdBackups.POST("", backupHandler.CreateEtcdBackup)
				etcdBackups.GET("", backupHandler.ListBackups)
				etcdBackups.GET(":backupId", backupHandler.GetBackup)
			}

			// 独立的资源备份接口
			resourceBackups := clusters.Group(":id/resources/backups")
			{
				resourceBackups.POST("", backupHandler.CreateResourceBackup)
				resourceBackups.GET("", backupHandler.ListBackups)
				resourceBackups.GET(":backupId", backupHandler.GetBackup)
			}

			// 备份计划接口
			backupSchedules := clusters.Group(":id/backup-schedules")
			{
				backupSchedules.GET("", backupHandler.ListBackupSchedules)
				backupSchedules.POST("", backupHandler.CreateBackupSchedule)
				backupSchedules.PUT(":scheduleId", backupHandler.UpdateBackupSchedule)
				backupSchedules.DELETE(":scheduleId", backupHandler.DeleteBackupSchedule)
			}

			// 审计相关接口
			audit := clusters.Group(":id/audit")
			{
				audit.GET("", auditHandler.ListAuditEvents)
			}

			// 扩展相关接口
			expansion := clusters.Group(":id/expansion")
			{
				expansion.POST("", expansionHandler.RequestExpansion)
				expansion.GET("/history", expansionHandler.GetExpansionHistory)
			}
		}

		// 导入状态接口（独立路径）
		imports := v1.Group("/imports")
		{
			imports.GET(":importId/status", importHandler.GetImportStatus)
		}

		// 机器管理接口
		machines := v1.Group("/machines")
		{
			machines.POST("", machineHandler.CreateMachine)
			machines.GET("", machineHandler.ListMachines)
			machines.GET(":id", machineHandler.GetMachine)
			machines.PUT(":id", machineHandler.UpdateMachine)
			machines.DELETE(":id", machineHandler.DeleteMachine)
			machines.PUT(":id/status", machineHandler.UpdateMachineStatus)
		}

		// 创建任务接口
		createTasks := v1.Group("/create-tasks")
		{
			createTasks.GET("", clusterHandler.ListCreateTasks)
			createTasks.GET(":taskId", clusterHandler.GetCreateTask)
		}

		// 三级分类模型接口
		tenants := v1.Group("/tenants")
		{
			tenants.POST("", tenantHandler.CreateTenant)
			tenants.GET("", tenantHandler.ListTenants)
			tenants.GET(":id", tenantHandler.GetTenant)
			tenants.PUT(":id", tenantHandler.UpdateTenant)
			tenants.DELETE(":id", tenantHandler.DeleteTenant)
			tenants.GET(":id/quota", tenantHandler.GetTenantQuota)
			tenants.PUT(":id/quota", tenantHandler.UpdateTenantQuota)
			tenants.GET(":id/environments", tenantHandler.GetTenantEnvironments)
			tenants.GET("/predefined", tenantHandler.PredefinedTenants)
			tenants.POST("/predefined/initialize", tenantHandler.InitializePredefinedTenants)
		}

		environments := v1.Group("/environments")
		{
			environments.POST("", environmentHandler.CreateEnvironment)
			environments.GET("", environmentHandler.ListEnvironments)
			environments.GET(":id", environmentHandler.GetEnvironment)
			environments.PUT(":id", environmentHandler.UpdateEnvironment)
			environments.DELETE(":id", environmentHandler.DeleteEnvironment)
			environments.GET(":id/quota", environmentHandler.GetEnvironmentQuota)
			environments.PUT(":id/quota", environmentHandler.UpdateEnvironmentQuota)
			environments.POST(":id/quota/inherit", environmentHandler.InheritQuota)
			environments.GET(":id/applications", environmentHandler.GetEnvironmentApplications)
			environments.GET(":id/namespace-info", environmentHandler.GetNamespaceInfo)
			environments.POST(":id/quota/sync", environmentHandler.SyncResourceQuota)
		}

		applications := v1.Group("/applications")
		{
			applications.POST("", applicationHandler.CreateApplication)
			applications.GET("", applicationHandler.ListApplications)
			applications.GET(":id", applicationHandler.GetApplication)
			applications.PUT(":id", applicationHandler.UpdateApplication)
			applications.DELETE(":id", applicationHandler.DeleteApplication)
			applications.GET(":id/resource-spec", applicationHandler.GetApplicationResourceSpec)
			applications.PUT(":id/resource-spec", applicationHandler.UpdateApplicationResourceSpec)
			applications.POST(":id/scale", applicationHandler.ScaleApplication)
			applications.POST("/discover", applicationHandler.DiscoverApplications)
			applications.POST("/aggregate", applicationHandler.AggregateApplications)
			applications.GET(":id/metrics", applicationHandler.GetApplicationMetrics)
			applications.GET("/tenant/:tenantId", applicationHandler.GetApplicationsByTenant)
		}
	}

	return r
}
