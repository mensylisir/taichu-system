package main

import (
	"infra-management/internal/api"
	"infra-management/internal/config"
	"infra-management/internal/db"
	"infra-management/internal/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := db.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	router := gin.Default()

	apiHandler := api.NewHandler(db)

	routes.SetupRoutes(router, apiHandler)

	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}