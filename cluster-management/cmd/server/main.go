package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/taichu-system/cluster-management/internal/config"
	"github.com/taichu-system/cluster-management/internal/handler"
	"github.com/taichu-system/cluster-management/internal/model"
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

	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if cfg.Database.AutoMigrate {
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

	healthCheckWorker := worker.NewHealthCheckWorker(
		clusterRepo,
		stateRepo,
		clusterManager,
		encryptionService,
	)

	if cfg.Worker.Enabled {
		healthCheckWorker.Start()
		defer healthCheckWorker.Stop()
	}

	clusterService := service.NewClusterService(
		clusterRepo,
		stateRepo,
		encryptionService,
		clusterManager,
	)

	clusterHandler := handler.NewClusterHandler(
		clusterService,
		encryptionService,
		healthCheckWorker,
	)

	r := setupRoutes(clusterHandler)

	srv := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := r.Run(":" + cfg.Server.Port); err != nil {
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
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		cfg.Database.Host,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.Port,
		cfg.Database.SSLMode,
		cfg.Database.Timezone,
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

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Cluster{},
		&model.ClusterState{},
	)
}

func setupRoutes(clusterHandler *handler.ClusterHandler) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	v1 := r.Group("/api/v1")
	{
		clusters := v1.Group("/clusters")
		{
			clusters.POST("", clusterHandler.CreateCluster)
			clusters.GET("", clusterHandler.ListClusters)
			clusters.GET(":id", clusterHandler.GetCluster)
		}
	}

	return r
}
