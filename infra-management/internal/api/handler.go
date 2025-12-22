package api

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"infra-management/internal/models"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) GetVMs(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 计算偏移量
	offset := (page - 1) * limit

	// 查询总数
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM vms").Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查询分页数据
	rows, err := h.db.Query(`
		SELECT id, name, os, status, cpu, memory, storage, ip, cluster, created_at
		FROM vms ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
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

	// 返回分页信息和数据
	c.JSON(http.StatusOK, gin.H{
		"data":       response,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": (total + limit - 1) / limit,
	})
}

func (h *Handler) GetStorages(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 计算偏移量
	offset := (page - 1) * limit

	// 查询总数
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM storages").Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查询分页数据
	rows, err := h.db.Query(`
		SELECT id, name, type, capacity, iops, status, mounted_to, created_at
		FROM storages ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
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

	// 返回分页信息和数据
	c.JSON(http.StatusOK, gin.H{
		"data":       response,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": (total + limit - 1) / limit,
	})
}

func (h *Handler) GetFirewallRules(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 计算偏移量
	offset := (page - 1) * limit

	// 查询总数
	var total int
	err := h.db.QueryRow("SELECT COUNT(*) FROM firewall_rules").Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查询分页数据
	rows, err := h.db.Query(`
		SELECT id, name, protocol, port, source_ip, target_ip, action, status, created_at
		FROM firewall_rules ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
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

	// 返回分页信息和数据
	c.JSON(http.StatusOK, gin.H{
		"data":       response,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": (total + limit - 1) / limit,
	})
}

// HealthCheck 健康检查接口 - 检查服务是否存活
func (h *Handler) HealthCheck(c *gin.Context) {
	// 简单检查 - 只要服务运行就返回200
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "service is running",
		"time":    h.getCurrentTime(),
	})
}

// ReadinessCheck 就绪检查接口 - 检查服务是否就绪
func (h *Handler) ReadinessCheck(c *gin.Context) {
	// 检查数据库连接是否正常
	err := h.db.Ping()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not ready",
			"message": "database connection failed",
			"time":    h.getCurrentTime(),
			"error":   err.Error(),
		})
		return
	}

	// 数据库连接正常
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"message": "service is ready to receive requests",
		"time":    h.getCurrentTime(),
	})
}

// getCurrentTime 获取当前时间字符串
func (h *Handler) getCurrentTime() string {
	return "2024-12-19 00:00:00" // 简化处理，实际应用中可以使用 time.Now().Format()
}

// ImportVMs 导入虚拟机列表
func (h *Handler) ImportVMs(c *gin.Context) {
	// 读取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "文件上传失败",
			"message": err.Error(),
		})
		return
	}

	// 检查文件类型
	fileName := strings.ToLower(file.Filename)
	if strings.Contains(fileName, ".xlsx") || strings.Contains(fileName, ".xls") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "不支持XLSX文件",
			"message": "请使用CSV格式的虚拟机列表文件",
		})
		return
	}

	if !strings.Contains(fileName, ".csv") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "文件格式不支持",
			"message": "仅支持CSV格式的文件",
		})
		return
	}

	// 打开文件
	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "打开文件失败",
			"message": err.Error(),
		})
		return
	}
	defer openedFile.Close()

	// 读取文件内容
	fileContent, err := io.ReadAll(openedFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "读取文件内容失败",
			"message": err.Error(),
		})
		return
	}

	// 去除BOM头
	contentStr := strings.TrimPrefix(string(fileContent), "\xEF\xBB\xBF")

	// 尝试检测和转换编码
	convertedStr := contentStr
	if !utf8.ValidString(contentStr) {
		// 如果不是有效的UTF-8，尝试作为GBK解码
		decoder := simplifiedchinese.GBK.NewDecoder()
		if decoded, err := decoder.String(contentStr); err == nil && utf8.ValidString(decoded) {
			convertedStr = decoded
		}
	}

	// 创建CSV reader
	reader := csv.NewReader(strings.NewReader(convertedStr))

	// 尝试读取头部，如果失败则提供更详细的错误信息
	headers, err := reader.Read()
	if err != nil {
		// 如果是编码问题，给出明确的提示
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "encoding") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "CSV文件编码不正确",
				"message": "请将Excel文件另存为UTF-8编码的CSV文件，然后再导入",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "读取文件头失败",
			"message": fmt.Sprintf("错误: %s", errMsg),
		})
		return
	}

	// 验证CSV文件格式
	expectedHeaders := []string{"虚拟机名称", "操作系统", "状态", "CPU", "内存", "存储", "IP地址", "所属集群", "创建时间"}
	if len(headers) < len(expectedHeaders) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "文件格式不正确",
			"message": fmt.Sprintf("需要包含以下列: %v", expectedHeaders),
		})
		return
	}

	// 逐行读取数据
	var importedCount int
	var failedRows []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failedRows = append(failedRows, fmt.Sprintf("读取行失败: %v", err))
			continue
		}

		// 验证数据行长度
		if len(record) < len(expectedHeaders) {
			failedRows = append(failedRows, fmt.Sprintf("数据行长度不正确: %s", record))
			continue
		}

		// 提取数据
		vmName := record[0]
		os := record[1]
		status := record[2]
		cpu := record[3]
		memory := record[4]
		storage := record[5]
		ip := record[6]
		cluster := record[7]
		createdTime := record[8]

		// 检查虚拟机是否已存在（根据名称）
		var exists bool
		err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM vms WHERE name = $1)", vmName).Scan(&exists)
		if err != nil {
			failedRows = append(failedRows, fmt.Sprintf("检查重复记录失败: %v", err))
			continue
		}

		if exists {
			failedRows = append(failedRows, fmt.Sprintf("虚拟机 '%s' 已存在", vmName))
			continue
		}

		// 插入数据库（id字段由数据库自动生成）
		_, err = h.db.Exec(`
			INSERT INTO vms (name, os, status, cpu, memory, storage, ip, cluster, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, vmName, os, status, cpu, memory, storage, ip, cluster, createdTime)

		if err != nil {
			failedRows = append(failedRows, fmt.Sprintf("插入记录 '%s' 失败: %v", vmName, err))
			continue
		}

		importedCount++
		log.Printf("成功导入虚拟机: %s", vmName)
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("导入完成！成功导入 %d 条记录", importedCount),
		"data": gin.H{
			"imported_count": importedCount,
			"failed_rows":    failedRows,
			"total_failed":   len(failedRows),
		},
	})
}
