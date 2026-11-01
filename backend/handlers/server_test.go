package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"smartdns-manager/database"
	"smartdns-manager/models"
)

func TestServerAdvancedEditAndScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:advanced-edit?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	old := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = old })
	if err := db.AutoMigrate(&models.DNSServer{}); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/servers", AddServer)
	r.PUT("/servers/:id", UpdateServer)
	call := func(method, path, body string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code >= 500 {
			t.Fatal(w.Body.String())
		}
		return w.Code
	}
	if status := call("POST", "/servers", `{"address":"https://dns.example/dns-query","type":"https","groups":[],"node_ids":"[3]","host_ip":"192.0.2.1","tls_host_verify":"dns.example","extra_options":"-http-host dns.example"}`); status != http.StatusCreated {
		t.Fatal(status)
	}
	var server models.DNSServer
	db.First(&server)
	if server.NodeIDs != "[3]" || server.HostIP != "192.0.2.1" || !strings.Contains(server.Options, "-http-host dns.example") {
		t.Fatalf("创建丢失高级字段: %+v", server)
	}
	if call("PUT", "/servers/1", `{"groups":["overseas"],"exclude_default":false}`) != 200 {
		t.Fatal("历史表单编辑失败")
	}
	db.First(&server)
	if server.HostIP != "192.0.2.1" || server.TLSHostVerify != "dns.example" || server.NodeIDs != "[3]" || !strings.Contains(server.Options, "-http-host dns.example") {
		t.Fatal("旧表单编辑抹掉高级参数")
	}
	if call("PUT", "/servers/1", `{"host_ip":"","node_ids":"[1,3]","groups":[],"enabled":false}`) != 200 {
		t.Fatal("字段更新失败")
	}
	db.First(&server)
	var groups []string
	_ = json.Unmarshal([]byte(server.GroupsStr), &groups)
	if server.HostIP != "" || strings.Contains(server.Options, "-host-ip") || server.NodeIDs != "[1,3]" || len(groups) != 0 || server.Enabled {
		t.Fatalf("显式清空/作用域更新失败: %+v", server)
	}
	if call("PUT", "/servers/1", `{"node_ids":"[0]"}`) != 400 {
		t.Fatal("接受了错误作用域")
	}
	if call("PUT", "/servers/1", `{"host_ip":"not-an-ip"}`) != 400 {
		t.Fatal("接受了错误 IP")
	}
}
