package handlers

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"smartdns-manager/database"
	"smartdns-manager/models"
	"smartdns-manager/services"
)

// PreviewNodeSync 按当前文件和数据库生成只读同步计划。
func PreviewNodeSync(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "节点 ID 无效"})
		return
	}
	var node models.Node
	if err := database.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "节点不存在"})
		return
	}
	client, err := services.NewSSHClient(&node)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "连接节点失败"})
		return
	}
	defer client.Close()
	before, err := client.ReadFile(node.ConfigPath)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "读取节点失败"})
		return
	}
	after, err := services.NewConfigSyncService().PlanFullSync(before, uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	losses := services.ConfigLosses(before, after)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"before": before, "after": after, "losses": losses, "safe": len(losses) == 0, "changed": before != after, "checksum": fmt.Sprintf("%x", sha256.Sum256([]byte(before)))}})
}
