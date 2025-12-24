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

	// 获取筛选参数
	name := c.Query("name")
	os := c.Query("os")
	status := c.Query("status")
	cpu := c.Query("cpu")
	memory := c.Query("memory")
	storage := c.Query("storage")
	ip := c.Query("ip")
	cluster := c.Query("cluster")
	createdAt := c.Query("createdAt")

	// 构建查询语句
	query := "SELECT name, os, status, cpu, memory, storage, ip, cluster, created_at FROM vms WHERE 1=1"
	params := []interface{}{}

	if name != "" {
		query += " AND name ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+name+"%")
	}
	if os != "" {
		query += " AND os ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+os+"%")
	}
	if status != "" {
		query += " AND status ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+status+"%")
	}
	if cpu != "" {
		query += " AND cpu ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+cpu+"%")
	}
	if memory != "" {
		query += " AND memory ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+memory+"%")
	}
	if storage != "" {
		query += " AND storage ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+storage+"%")
	}
	if ip != "" {
		query += " AND ip ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+ip+"%")
	}
	if cluster != "" {
		query += " AND cluster ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+cluster+"%")
	}
	if createdAt != "" {
		query += " AND created_at ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+createdAt+"%")
	}

	// 查询总数（带筛选条件）
	countQuery := "SELECT COUNT(*) FROM vms WHERE 1=1"
	countParams := []interface{}{}

	if name != "" {
		countQuery += " AND name ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+name+"%")
	}
	if os != "" {
		countQuery += " AND os ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+os+"%")
	}
	if status != "" {
		countQuery += " AND status ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+status+"%")
	}
	if cpu != "" {
		countQuery += " AND cpu ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+cpu+"%")
	}
	if memory != "" {
		countQuery += " AND memory ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+memory+"%")
	}
	if storage != "" {
		countQuery += " AND storage ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+storage+"%")
	}
	if ip != "" {
		countQuery += " AND ip ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+ip+"%")
	}
	if cluster != "" {
		countQuery += " AND cluster ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+cluster+"%")
	}
	if createdAt != "" {
		countQuery += " AND created_at ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+createdAt+"%")
	}

	var total int
	err := h.db.QueryRow(countQuery, countParams...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 添加排序和分页限制
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(params)+1) + " OFFSET $" + strconv.Itoa(len(params)+2)
	params = append(params, limit, offset)

	// 查询分页数据
	rows, err := h.db.Query(query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var vms []models.VM
	for rows.Next() {
		var vm models.VM
		err := rows.Scan(&vm.Name, &vm.OS, &vm.Status, &vm.CPU, &vm.Memory,
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

	// 获取筛选参数
	name := c.Query("name")
	typ := c.Query("type")
	capacity := c.Query("capacity")
	iops := c.Query("iops")
	status := c.Query("status")
	mountedTo := c.Query("mountedTo")
	createdAt := c.Query("createdAt")

	// 构建查询语句
	query := "SELECT name, type, capacity, iops, status, mounted_to, created_at FROM storages WHERE 1=1"
	params := []interface{}{}

	if name != "" {
		query += " AND name ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+name+"%")
	}
	if typ != "" {
		query += " AND type ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+typ+"%")
	}
	if capacity != "" {
		query += " AND capacity ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+capacity+"%")
	}
	if iops != "" {
		query += " AND iops ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+iops+"%")
	}
	if status != "" {
		query += " AND status ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+status+"%")
	}
	if mountedTo != "" {
		query += " AND mounted_to ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+mountedTo+"%")
	}
	if createdAt != "" {
		query += " AND created_at ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+createdAt+"%")
	}

	// 查询总数（带筛选条件）
	countQuery := "SELECT COUNT(*) FROM storages WHERE 1=1"
	countParams := []interface{}{}

	if name != "" {
		countQuery += " AND name ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+name+"%")
	}
	if typ != "" {
		countQuery += " AND type ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+typ+"%")
	}
	if capacity != "" {
		countQuery += " AND capacity ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+capacity+"%")
	}
	if iops != "" {
		countQuery += " AND iops ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+iops+"%")
	}
	if status != "" {
		countQuery += " AND status ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+status+"%")
	}
	if mountedTo != "" {
		countQuery += " AND mounted_to ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+mountedTo+"%")
	}
	if createdAt != "" {
		countQuery += " AND created_at ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+createdAt+"%")
	}

	var total int
	err := h.db.QueryRow(countQuery, countParams...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 添加排序和分页限制
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(params)+1) + " OFFSET $" + strconv.Itoa(len(params)+2)
	params = append(params, limit, offset)

	// 查询分页数据
	rows, err := h.db.Query(query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var storages []models.Storage
	for rows.Next() {
		var storage models.Storage
		var mountedTo sql.NullString
		var createdAt sql.NullTime
		err := rows.Scan(&storage.Name, &storage.Type, &storage.Capacity,
			&storage.IOPS, &storage.Status, &mountedTo, &createdAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 处理 mounted_to 字段
		if mountedTo.Valid {
			storage.MountedTo = &mountedTo.String
		}
		// 处理 created_at 字段
		if createdAt.Valid {
			storage.CreatedAt = createdAt.Time
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

	// 获取筛选参数
	name := c.Query("name")
	protocol := c.Query("protocol")
	port := c.Query("port")
	sourceIp := c.Query("sourceIp")
	targetIp := c.Query("targetIp")
	action := c.Query("action")
	status := c.Query("status")
	createdAt := c.Query("createdAt")

	// 构建查询语句
	query := "SELECT name, protocol, port, source_ip, target_ip, action, status, created_at FROM firewall_rules WHERE 1=1"
	params := []interface{}{}

	if name != "" {
		query += " AND name ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+name+"%")
	}
	if protocol != "" {
		query += " AND protocol ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+protocol+"%")
	}
	if port != "" {
		query += " AND port ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+port+"%")
	}
	if sourceIp != "" {
		query += " AND source_ip ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+sourceIp+"%")
	}
	if targetIp != "" {
		query += " AND target_ip ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+targetIp+"%")
	}
	if action != "" {
		query += " AND action ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+action+"%")
	}
	if status != "" {
		query += " AND status ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+status+"%")
	}
	if createdAt != "" {
		query += " AND created_at ILIKE $" + strconv.Itoa(len(params)+1)
		params = append(params, "%"+createdAt+"%")
	}

	// 查询总数（带筛选条件）
	countQuery := "SELECT COUNT(*) FROM firewall_rules WHERE 1=1"
	countParams := []interface{}{}

	if name != "" {
		countQuery += " AND name ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+name+"%")
	}
	if protocol != "" {
		countQuery += " AND protocol ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+protocol+"%")
	}
	if port != "" {
		countQuery += " AND port ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+port+"%")
	}
	if sourceIp != "" {
		countQuery += " AND source_ip ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+sourceIp+"%")
	}
	if targetIp != "" {
		countQuery += " AND target_ip ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+targetIp+"%")
	}
	if action != "" {
		countQuery += " AND action ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+action+"%")
	}
	if status != "" {
		countQuery += " AND status ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+status+"%")
	}
	if createdAt != "" {
		countQuery += " AND created_at ILIKE $" + strconv.Itoa(len(countParams)+1)
		countParams = append(countParams, "%"+createdAt+"%")
	}

	var total int
	err := h.db.QueryRow(countQuery, countParams...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 添加排序和分页限制
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(params)+1) + " OFFSET $" + strconv.Itoa(len(params)+2)
	params = append(params, limit, offset)

	// 查询分页数据
	rows, err := h.db.Query(query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rules []models.FirewallRule
	for rows.Next() {
		var rule models.FirewallRule
		err := rows.Scan(&rule.Name, &rule.Protocol, &rule.Port,
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

// DeleteVM 删除虚拟机
func (h *Handler) DeleteVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "虚拟机名称不能为空",
		})
		return
	}

	// 检查虚拟机是否存在
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM vms WHERE name = $1)", vmName).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "虚拟机不存在",
		})
		return
	}

	// 删除虚拟机
	_, err = h.db.Exec("DELETE FROM vms WHERE name = $1", vmName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("虚拟机 '%s' 删除成功", vmName),
	})
}

// DeleteStorage 删除存储
func (h *Handler) DeleteStorage(c *gin.Context) {
	storageName := c.Param("name")
	if storageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "存储名称不能为空",
		})
		return
	}

	// 检查存储是否存在
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM storages WHERE name = $1)", storageName).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "存储不存在",
		})
		return
	}

	// 删除存储
	_, err = h.db.Exec("DELETE FROM storages WHERE name = $1", storageName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("存储 '%s' 删除成功", storageName),
	})
}

// DeleteFirewallRule 删除防火墙规则
func (h *Handler) DeleteFirewallRule(c *gin.Context) {
	ruleName := c.Param("name")
	if ruleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "防火墙规则名称不能为空",
		})
		return
	}

	// 检查防火墙规则是否存在
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM firewall_rules WHERE name = $1)", ruleName).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "防火墙规则不存在",
		})
		return
	}

	// 删除防火墙规则
	_, err = h.db.Exec("DELETE FROM firewall_rules WHERE name = $1", ruleName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("防火墙规则 '%s' 删除成功", ruleName),
	})
}
