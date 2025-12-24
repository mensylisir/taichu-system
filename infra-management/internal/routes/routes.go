package routes

import (
	"infra-management/internal/api"
	"infra-management/internal/ssh"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, apiHandler *api.Handler) {
	// 健康检查接口
	router.GET("/health", apiHandler.HealthCheck)
	router.GET("/ready", apiHandler.ReadinessCheck)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/vms", apiHandler.GetVMs)
		v1.GET("/storages", apiHandler.GetStorages)
		v1.GET("/firewall-rules", apiHandler.GetFirewallRules)
		v1.POST("/vms/import", apiHandler.ImportVMs) // 添加导入虚拟机接口

		// 删除接口
		v1.DELETE("/vms/:name", apiHandler.DeleteVM)
		v1.DELETE("/storages/:name", apiHandler.DeleteStorage)
		v1.DELETE("/firewall-rules/:name", apiHandler.DeleteFirewallRule)
	}

	// WebSocket SSH终端路由
	wsHandler := ssh.NewWebSocketSSHHandler()
	router.GET("/ws/ssh", func(c *gin.Context) {
		wsHandler.HandleWebSocket(c.Writer, c.Request)
	})

	// 设置静态文件服务（用于前端终端页面）
	router.Static("/terminal", "./static/terminal")
}
