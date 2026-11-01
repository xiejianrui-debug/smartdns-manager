package services

import (
	"strings"
	"testing"

	"smartdns-manager/models"
)

func TestNewServerInsertedBesideMatchingGroup(t *testing.T) {
	input := "# DNS Servers\nserver 192.0.2.1 -group local\nserver 192.0.2.2 -group local\nserver 192.0.2.3 -group ad\nserver-https https://dns.example/dns-query -host-ip 192.0.2.4\n\n# Address Mappings\naddress /example/192.0.2.5\n"
	p := NewConfigParser()
	c, err := p.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	added := models.DNSServer{Address: "192.0.2.6", Type: "udp", GroupsStr: `["local"]`}
	c = NewConfigSyncService().mergeConfigs(c, []models.DNSServer{added}, nil)
	out := p.Generate(c)
	if !strings.Contains(out, "server 192.0.2.2 -group local\nserver 192.0.2.6 -group local\nserver 192.0.2.3 -group ad") {
		t.Fatalf("同组上游没有相邻：\n%s", out)
	}
	if losses := ConfigLosses(input, out); len(losses) != 0 {
		t.Fatalf("配置损失：%v", losses)
	}
	next, _ := p.Parse(out)
	next = NewConfigSyncService().mergeConfigs(next, []models.DNSServer{added}, nil)
	if p.Generate(next) != out {
		t.Fatal("重复同步改变配置")
	}
}
