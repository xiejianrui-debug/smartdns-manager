package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"smartdns-manager/database"
	"smartdns-manager/models"
	"smartdns-manager/services"
)

// AddServer 添加DNS服务器
func AddServer(c *gin.Context) {
	var server models.DNSServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 验证地址
	if server.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "服务器地址不能为空",
		})
		return
	}

	services.HydrateServerOptions(&server)
	if server.Type == "" {
		server.Type = services.ServerType(server.Directive, server.Address)
	}
	server.OptionsVersion = 1
	server.Options = services.BuildServerOptions(server)
	if err := services.ValidateServer(server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	groupsJSON, _ := json.Marshal(server.Groups)
	server.GroupsStr = string(groupsJSON)

	// 检查是否已存在
	var existing models.DNSServer
	if err := database.DB.Where("address = ?", server.Address).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "该DNS服务器已存在",
		})
		return
	}

	if err := database.DB.Create(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "添加DNS服务器失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "DNS服务器添加成功",
		"data":    server,
	})
}

// UpdateServer 更新DNS服务器
func UpdateServer(c *gin.Context) {
	id := c.Param("id")
	serverID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的服务器ID",
		})
		return
	}

	var server models.DNSServer
	if err := database.DB.First(&server, serverID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "DNS服务器不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "查询DNS服务器失败",
		})
		return
	}

	if server.GroupsStr != "" {
		_ = json.Unmarshal([]byte(server.GroupsStr), &server.Groups)
	}
	services.HydrateServerOptions(&server)
	var fields map[string]json.RawMessage
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}
	// 兼容历史 Options 输入；随后显式字段具有更高优先级。
	if raw, ok := fields["options"]; ok {
		var options string
		if err := json.Unmarshal(raw, &options); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "附加参数格式错误"})
			return
		}
		parsed := models.DNSServer{Options: options}
		services.HydrateServerOptions(&parsed)
		server.HostIP, server.TLSHostVerify, server.ExtraOptions = parsed.HostIP, parsed.TLSHostVerify, parsed.ExtraOptions
		if _, exists := fields["groups"]; !exists {
			server.Groups = parsed.Groups
		}
		if _, exists := fields["exclude_default"]; !exists {
			server.ExcludeDefault = parsed.ExcludeDefault
		}
	}
	allowed := map[string]json.RawMessage{}
	for _, key := range []string{"address", "type", "directive", "groups", "exclude_default", "host_ip", "tls_host_verify", "extra_options", "node_ids", "enabled"} {
		if value, ok := fields[key]; ok {
			allowed[key] = value
		}
	}
	encoded, _ := json.Marshal(allowed)
	oldType := server.Type
	if err := json.Unmarshal(encoded, &server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "上游字段格式错误"})
		return
	}
	if server.Type != oldType {
		if _, explicit := fields["directive"]; !explicit {
			server.Directive = ""
		}
	}
	server.OptionsVersion = 1
	server.Options = services.BuildServerOptions(server)
	if err := services.ValidateServer(server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	groupsJSON, _ := json.Marshal(server.Groups)
	server.GroupsStr = string(groupsJSON)

	if err := database.DB.Save(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新DNS服务器失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "DNS服务器更新成功",
		"data":    server,
	})
}

// DeleteServer 删除DNS服务器
func DeleteServer(c *gin.Context) {
	id := c.Param("id")
	serverID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的服务器ID",
		})
		return
	}

	var server models.DNSServer
	if err := database.DB.First(&server, serverID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "DNS服务器不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "查询DNS服务器失败",
		})
		return
	}

	if err := database.DB.Delete(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除DNS服务器失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "DNS服务器删除成功",
	})
}

// GetServers 获取DNS服务器列表
func GetServers(c *gin.Context) {
	var servers []models.DNSServer

	query := database.DB

	// 支持类型筛选
	if serverType := c.Query("type"); serverType != "" {
		query = query.Where("type = ?", serverType)
	}

	// 支持分组筛选
	if group := c.Query("group"); group != "" {
		query = query.Where("groups LIKE ?", "%"+group+"%")
	}

	if err := query.Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取DNS服务器列表失败",
			"error":   err.Error(),
		})
		return
	}

	// 反序列化分组并读取历史高级参数。
	for i := range servers {
		if servers[i].GroupsStr != "" {
			json.Unmarshal([]byte(servers[i].GroupsStr), &servers[i].Groups)
		}
		services.HydrateServerOptions(&servers[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    servers,
		"total":   len(servers),
	})
}

// 辅助函数：生成服务器选项字符串
func generateServerOptions(server *models.DNSServer) string {
	options := []string{}

	for _, group := range server.Groups {
		options = append(options, "-group "+group)
	}

	if server.ExcludeDefault {
		options = append(options, "-exclude-default-group")
	}

	return strings.Join(options, " ")
}
