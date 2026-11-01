package handlers

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"smartdns-manager/database"
	"smartdns-manager/models"
	"smartdns-manager/services"
)

// GetNodeConfig 获取节点配置
func GetNodeConfig(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	// 连接到节点
	client, err := services.NewSSHClient(&node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "连接节点失败",
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	// 读取配置文件
	content, err := client.ReadFile(node.ConfigPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "读取配置文件失败",
			"error":   err.Error(),
		})
		return
	}

	// 解析配置
	parser := services.NewConfigParser()
	config, err := parser.Parse(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "解析配置失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"data":       config,
		"rawContent": content,
		"checksum":   fmt.Sprintf("%x", md5.Sum([]byte(content))),
	})
}

// SaveNodeConfig 保存节点配置
func SaveNodeConfig(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	var request struct {
		Config     *models.SmartDNSConfig `json:"config"`
		RawContent string                 `json:"raw_content"`
		Checksum   string                 `json:"checksum"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 连接到节点
	client, err := services.NewSSHClient(&node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "连接节点失败",
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	current, err := client.ReadFile(node.ConfigPath)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "读取当前配置失败"})
		return
	}
	if request.RawContent != "" && request.Checksum == "" {
		c.JSON(http.StatusPreconditionRequired, gin.H{"success": false, "message": "请刷新控制台并重新加载配置后再保存"})
		return
	}
	if request.Checksum != "" && request.Checksum != fmt.Sprintf("%x", md5.Sum([]byte(current))) {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "节点配置已变化，请重新加载后合并修改"})
		return
	}
	if request.RawContent == "" {
		if request.Config == nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "配置不能为空"})
			return
		}
		request.Config.SourceContent = current
		candidate := services.NewConfigParser().Generate(request.Config)
		if losses := services.ConfigLosses(current, candidate); len(losses) > 0 {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "结构化保存可能丢失配置，已阻止", "losses": losses})
			return
		}
	}

	// 创建备份
	backupPath, err := client.CreateBackup(node.ConfigPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建备份失败",
			"error":   err.Error(),
		})
		return
	}

	// 生成新配置内容
	var newContent string
	if request.RawContent != "" {
		newContent = request.RawContent
	} else {
		parser := services.NewConfigParser()
		newContent = parser.Generate(request.Config)
	}

	// 写入配置文件
	if err := client.WriteFile(node.ConfigPath, newContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "写入配置文件失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "配置保存成功",
		"backupPath": backupPath,
		"checksum":   fmt.Sprintf("%x", md5.Sum([]byte(newContent))),
	})
}

// RestartNodeService 重启节点服务
func RestartNodeService(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	client, err := services.NewSSHClient(&node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "连接节点失败",
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	if err := client.RestartService("smartdns"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "重启服务失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "服务重启成功",
	})
}

// GetNodeStatus 获取节点状态
func GetNodeStatus(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	client, err := services.NewSSHClient(&node)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": models.NodeStatus{
				NodeID:      uint(nodeID),
				IsOnline:    false,
				ServiceUp:   false,
				LastChecked: time.Now(),
			},
		})
		return
	}
	defer client.Close()

	status, err := client.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取系统信息失败",
			"error":   err.Error(),
		})
		return
	}

	status.NodeID = uint(nodeID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// GetNodeLogs 获取节点日志
func GetNodeLogs(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}

	lines := 100
	if l := c.Query("lines"); l != "" {
		if parsedLines, err := strconv.Atoi(l); err == nil {
			lines = parsedLines
		}
	}

	var node models.Node
	if err := database.DB.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	client, err := services.NewSSHClient(&node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "连接节点失败",
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	logs, err := client.GetLogs(lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取日志失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
	})
}

// BatchUpdateConfig 批量更新配置
func BatchUpdateConfig(c *gin.Context) {
	var request struct {
		NodeIDs []uint                 `json:"node_ids" binding:"required"`
		Config  *models.SmartDNSConfig `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	results := make(map[uint]map[string]interface{})
	parser := services.NewConfigParser()
	for _, nodeID := range request.NodeIDs {
		result := map[string]interface{}{
			"success": false,
		}

		var node models.Node
		if err := database.DB.First(&node, nodeID).Error; err != nil {
			result["error"] = "节点不存在"
			results[nodeID] = result
			continue
		}

		client, err := services.NewSSHClient(&node)
		if err != nil {
			result["error"] = "连接失败: " + err.Error()
			results[nodeID] = result
			continue
		}

		current, readErr := client.ReadFile(node.ConfigPath)
		if readErr != nil {
			client.Close()
			result["error"] = "读取当前配置失败"
			results[nodeID] = result
			continue
		}
		copy := *request.Config
		copy.SourceContent = current
		newContent := parser.Generate(&copy)
		if losses := services.ConfigLosses(current, newContent); len(losses) > 0 {
			client.Close()
			result["error"] = "批量配置会丢失现有指令，已阻止"
			result["losses"] = losses
			results[nodeID] = result
			continue
		}
		// 创建备份
		backupPath, err := client.CreateBackup(node.ConfigPath)
		if err != nil {
			client.Close()
			result["error"] = "备份失败: " + err.Error()
			results[nodeID] = result
			continue
		}

		// 写入配置
		if err := client.WriteFile(node.ConfigPath, newContent); err != nil {
			client.Close()
			result["error"] = "写入失败: " + err.Error()
			results[nodeID] = result
			continue
		}

		client.Close()
		result["success"] = true
		result["backup_path"] = backupPath
		results[nodeID] = result
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量更新完成",
		"data":    results,
	})
}

// BatchRestart 批量重启服务
func BatchRestart(c *gin.Context) {
	var request struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	results := make(map[uint]map[string]interface{})

	for _, nodeID := range request.NodeIDs {
		result := map[string]interface{}{
			"success": false,
		}

		var node models.Node
		if err := database.DB.First(&node, nodeID).Error; err != nil {
			result["error"] = "节点不存在"
			results[nodeID] = result
			continue
		}

		client, err := services.NewSSHClient(&node)
		if err != nil {
			result["error"] = "连接失败: " + err.Error()
			results[nodeID] = result
			continue
		}

		if err := client.RestartService("smartdns"); err != nil {
			client.Close()
			result["error"] = "重启失败: " + err.Error()
			results[nodeID] = result
			continue
		}

		client.Close()
		result["success"] = true
		results[nodeID] = result
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量重启完成",
		"data":    results,
	})
}
