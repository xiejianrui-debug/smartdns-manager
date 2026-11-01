package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"smartdns-manager/models"
	"strconv"
	"strings"
	"testing"
)

// 可选的现场快照验证；生产文件始终放在仓库外，不提交测试数据。
func TestExternalConfigSnapshots(t *testing.T) {
	dir := os.Getenv("SMARTDNS_CONFIG_AUDIT_DIR")
	if dir == "" {
		t.Skip("未指定现场快照目录")
	}
	var servers []models.DNSServer
	var addresses []models.AddressMap
	b, err := os.ReadFile(filepath.Join(dir, "planned-servers.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &servers); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(dir, "addresses.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &addresses); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "node-*.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("没有找到节点快照")
	}
	for _, file := range files {
		id, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(file), "node-"), ".conf"))
		if err != nil {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		p := NewConfigParser()
		cfg, err := p.Parse(string(content))
		if err != nil {
			t.Fatal(err)
		}
		s := &ConfigSyncService{}
		cfg = s.mergeConfigs(cfg, s.filterServersForNode(servers, uint(id)), s.filterConfigForNode(addresses, uint(id)))
		after := p.Generate(cfg)
		losses := ConfigLosses(string(content), after)
		if len(losses) > 0 {
			t.Errorf("节点 %d 可能丢失: %v", id, losses)
		}
		if err = os.WriteFile(filepath.Join(dir, strconv.Itoa(id)+".plan.conf"), []byte(after), 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("节点 %d: 丢失 %d 项，生成 %d 字节", id, len(losses), len(after))
	}
}

func TestSyncProtectsUnknownAndAdvancedOptions(t *testing.T) {
	p := NewConfigParser()
	before := "bind :53\ncache-persist yes\nserver-https https://dns.example/dns-query -host-ip 192.0.2.1 -tls-host-verify dns.example -http-host dns.example\nconf-file /etc/smartdns/extra.conf\n"
	cfg, _ := p.Parse(before)
	record := models.DNSServer{Address: "https://dns.example/dns-query", Type: "https", Options: "", GroupsStr: "[]"}
	cfg.Servers[0] = MergeServerForSync(cfg.Servers[0], record)
	cfg.Servers = append(cfg.Servers, models.DNSServer{Address: "192.0.2.2", Type: "udp", Groups: []string{"local"}, ExcludeDefault: true})
	after := p.Generate(cfg)
	if losses := ConfigLosses(before, after); len(losses) != 0 {
		t.Fatal(losses)
	}
	if !strings.Contains(after, "-host-ip 192.0.2.1") || !strings.Contains(after, "-http-host dns.example") {
		t.Fatal(after)
	}
	removed := strings.Replace(after, "conf-file /etc/smartdns/extra.conf\n", "", 1)
	if len(ConfigLosses(before, removed)) != 1 {
		t.Fatal("未检测出未知配置丢失")
	}
	changed := strings.Replace(after, "-host-ip 192.0.2.1", "-host-ip 192.0.2.99", 1)
	if len(ConfigLosses(before, changed)) != 1 {
		t.Fatal("未检测出高级字段变化")
	}
}

func TestParserPreservesMultipleListenersAndAppendsInSection(t *testing.T) {
	p := NewConfigParser()
	before := "# Basic Settings\nbind :53\nbind 127.0.0.1:5353\n# DNS Servers\nserver 192.0.2.1 -group local\n# Domain Rules\nclient-rules 192.0.2.0/24 -group office\n"
	cfg, _ := p.Parse(before)
	cfg.Servers = append(cfg.Servers, models.DNSServer{Address: "192.0.2.2", Type: "udp", Groups: []string{"local"}})
	after := p.Generate(cfg)
	if len(ConfigLosses(before, after)) != 0 || strings.Count(after, "bind ") != 2 {
		t.Fatal(after)
	}
	if strings.Index(after, "server 192.0.2.2") > strings.Index(after, "# Domain Rules") {
		t.Fatal("新增上游未归入原分类")
	}
}

func TestLegacySyncRetainsNodeSpecificBootstrapGroup(t *testing.T) {
	p := NewConfigParser()
	before := "server 192.0.2.1 -group cn -group bootstrap\n"
	cfg, _ := p.Parse(before)
	cfg.Servers[0] = MergeServerForSync(cfg.Servers[0], models.DNSServer{Address: "192.0.2.1", Type: "udp", GroupsStr: `["cn"]`, Options: "-group cn"})
	after := p.Generate(cfg)
	if !strings.Contains(after, "-group bootstrap") || len(ConfigLosses(before, after)) != 0 {
		t.Fatal(after)
	}
	if len(ConfigLosses(before, "server 192.0.2.1 -group cn\n")) != 1 {
		t.Fatal("未检测到分组丢失")
	}
}
