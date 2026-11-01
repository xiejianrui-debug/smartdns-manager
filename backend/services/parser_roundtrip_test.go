package services

import (
	"strings"
	"testing"
)

func TestParserPreservesAdvancedConfiguration(t *testing.T) {
	input := `# 自定义配置必须保留
bind :53
bind-tcp :53
cache-persist yes
conf-file /etc/smartdns/extra.conf
server-https https://dns.example/dns-query -host-ip 192.0.2.1 -tls-host-verify dns.example -http-host dns.example
server-tls 192.0.2.2:853 -tls-host-verify tls.example
cname /alias.example/target.example
domain-rules /example/ -address 192.0.2.3 -nameserver local -speed-check-mode none -cname target.example
group-begin private
server 192.0.2.4 -group nested
group-end
`
	p := NewConfigParser()
	cfg, err := p.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Servers) != 2 {
		t.Fatalf("协议上游解析遗漏: %+v", cfg.Servers)
	}
	output := p.Generate(cfg)
	for _, line := range strings.Split(strings.TrimSpace(input), "\n") {
		if !strings.Contains(output, line) {
			t.Errorf("配置丢失: %s\n%s", line, output)
		}
	}
	for i := 0; i < 3; i++ {
		cfg, err = p.Parse(output)
		if err != nil {
			t.Fatal(err)
		}
		next := p.Generate(cfg)
		if next != output {
			t.Fatalf("重复同步改变配置\n%s\n%s", output, next)
		}
		output = next
	}
}

func TestParserDoesNotAccumulateRuleOptions(t *testing.T) {
	p := NewConfigParser()
	input := "domain-rules /example/ -address 0.0.0.0 -speed-check-mode none -nameserver local\n"
	for i := 0; i < 4; i++ {
		cfg, err := p.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		input = p.Generate(cfg)
		for _, option := range []string{"-address", "-speed-check-mode", "-nameserver"} {
			if strings.Count(input, option) != 1 {
				t.Fatalf("参数重复: %s", input)
			}
		}
	}
}
