package api

import (
	"database/sql"
	"infra-management/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetVMs(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, os, status, cpu, memory, storage, ip, cluster, created_at
		FROM vms ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var vms []models.VM
	for rows.Next() {
		var vm models.VM
		err := rows.Scan(&vm.ID, &vm.Name, &vm.OS, &vm.Status, &vm.CPU, &vm.Memory,
			&vm.Storage, &vm.IP, &vm.Cluster, &vm.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		vms = append(vms, vm)
	}

	var response []models.VMResponse
	for _, vm := range vms {
		response = append(response, *vm.ToResponse())
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetStorages(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, type, capacity, iops, status, mounted_to, created_at
		FROM storages ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var storages []models.Storage
	for rows.Next() {
		var storage models.Storage
		var mountedTo sql.NullString
		err := rows.Scan(&storage.ID, &storage.Name, &storage.Type, &storage.Capacity,
			&storage.IOPS, &storage.Status, &mountedTo, &storage.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if mountedTo.Valid {
			storage.MountedTo = &mountedTo.String
		}
		storages = append(storages, storage)
	}

	var response []models.StorageResponse
	for _, s := range storages {
		response = append(response, *s.ToResponse())
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetFirewallRules(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, protocol, port, source_ip, target_ip, action, status, created_at
		FROM firewall_rules ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rules []models.FirewallRule
	for rows.Next() {
		var rule models.FirewallRule
		err := rows.Scan(&rule.ID, &rule.Name, &rule.Protocol, &rule.Port,
			&rule.SourceIP, &rule.TargetIP, &rule.Action, &rule.Status, &rule.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rules = append(rules, rule)
	}

	var response []models.FirewallResponse
	for _, rule := range rules {
		response = append(response, *rule.ToResponse())
	}

	c.JSON(http.StatusOK, response)
}
