package routes

import (
	"infra-management/internal/api"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, apiHandler *api.Handler) {
	v1 := router.Group("/api/v1")
	{
		v1.GET("/vms", apiHandler.GetVMs)
		v1.GET("/storages", apiHandler.GetStorages)
		v1.GET("/firewall-rules", apiHandler.GetFirewallRules)
	}
}