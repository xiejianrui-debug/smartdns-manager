package services

import (
	"fmt"
	"regexp"
	"smartdns-manager/models"
	"sort"
	"strings"
	"time"
)

type ConfigParser struct{}

func NewConfigParser() *ConfigParser {
	return &ConfigParser{}
}

func (p *ConfigParser) Parse(content string) (*models.SmartDNSConfig, error) {
	config := &models.SmartDNSConfig{
		Servers:       []models.DNSServer{},
		Addresses:     []models.AddressMap{},
		DomainSets:    []models.DomainSet{},
		DomainRules:   []models.DomainRule{},
		Nameservers:   []models.Nameserver{},
		BasicSettings: make(map[string]string),
		SourceContent: content,
	}

	lines := strings.Split(content, "\n")

	depth := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "group-begin ") {
			depth++
			continue
		}
		if line == "group-end" {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth > 0 {
			continue
		}

		// 解析显式及 URL 协议的上游。
		if isServerDirective(strings.Fields(line)[0]) {
			server := p.parseServer(line)
			if server != nil {
				config.Servers = append(config.Servers, *server)
			}
		}

		// 解析地址和 CNAME 映射。
		if strings.HasPrefix(line, "cname /") {
			mapped := p.parseAddress(strings.Replace(line, "cname ", "address ", 1))
			if mapped != nil {
				mapped.Type = "cname"
				mapped.CNAME = mapped.IP
				mapped.IP = ""
				config.Addresses = append(config.Addresses, *mapped)
			}
		}
		if strings.HasPrefix(line, "address /") {
			address := p.parseAddress(line)
			if address != nil {
				config.Addresses = append(config.Addresses, *address)
			}
		}

		// 解析 domain-set
		if strings.HasPrefix(line, "domain-set ") {
			domainSet := p.parseDomainSet(line)
			if domainSet != nil {
				config.DomainSets = append(config.DomainSets, *domainSet)
			}
		}

		// 解析 domain-rules
		if strings.HasPrefix(line, "domain-rules /") {
			rule := p.parseDomainRule(line)
			if rule != nil {
				config.DomainRules = append(config.DomainRules, *rule)
			}
		}

		// 解析 nameserver
		if strings.HasPrefix(line, "nameserver /") {
			ns := p.parseNameserver(line)
			if ns != nil {
				config.Nameservers = append(config.Nameservers, *ns)
			}
		}

		// 解析基础设置
		p.parseBasicSetting(line, config.BasicSettings)
	}

	return config, nil
}

func (p *ConfigParser) parseServer(line string) *models.DNSServer {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	server := &models.DNSServer{Directive: fields[0], Address: fields[1]}
	server.Type = ServerType(server.Directive, server.Address)
	server.Options = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, fields[0])), fields[1]))
	HydrateServerOptions(server)
	return server
}

func (p *ConfigParser) parseAddress(line string) *models.AddressMap {
	re := regexp.MustCompile(`address\s+/(.*?)/(.*)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil
	}

	return &models.AddressMap{
		Domain: matches[1],
		IP:     matches[2],
	}
}

func (p *ConfigParser) parseDomainSet(line string) *models.DomainSet {
	nameRe := regexp.MustCompile(`-name\s+(\S+)`)
	fileRe := regexp.MustCompile(`-file\s+(\S+)`)

	nameMatch := nameRe.FindStringSubmatch(line)
	fileMatch := fileRe.FindStringSubmatch(line)

	if len(nameMatch) < 2 {
		return nil
	}

	ds := &models.DomainSet{
		Name: nameMatch[1],
	}

	if len(fileMatch) > 1 {
		ds.FilePath = fileMatch[1]
	}

	return ds
}

func (p *ConfigParser) parseDomainRule(line string) *models.DomainRule {
	// 匹配 domain-rules /domain/ options 或 domain-rules /domain-set:name/ options
	re := regexp.MustCompile(`domain-rules\s+/(.*?)/\s*(.*)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 3 {
		return nil
	}

	domain := strings.TrimSpace(matches[1])
	options := strings.TrimSpace(matches[2])

	rule := &models.DomainRule{
		Enabled: true,
	}

	// 检查是否是域名集引用
	if strings.HasPrefix(domain, "domain-set:") {
		rule.IsDomainSet = true
		rule.DomainSetName = strings.TrimPrefix(domain, "domain-set:")
		rule.Domain = domain // 保存完整的引用
	} else {
		rule.IsDomainSet = false
		rule.Domain = domain
	}

	// 仅提取已建模参数，其余参数和注释保持；重复赋值沿用最后一个值。
	fields := optionTokens.FindAllString(options, -1)
	for i := 0; i < len(fields); i++ {
		if fields[i] == "#" {
			break
		}
		if i+1 >= len(fields) {
			continue
		}
		switch fields[i] {
		case "-address":
			rule.Address = optionValue(fields[i+1])
			i++
		case "-nameserver":
			rule.Nameserver = optionValue(fields[i+1])
			i++
		case "-speed-check-mode":
			rule.SpeedCheckMode = optionValue(fields[i+1])
			i++
		}
	}
	rule.OtherOptions = removeManagedOptions(options, map[string]bool{"-address": true, "-nameserver": true, "-speed-check-mode": true})

	return rule
}

func (p *ConfigParser) parseNameserver(line string) *models.Nameserver {
	// 匹配 nameserver /domain/group 或 nameserver /domain-set:name/group
	re := regexp.MustCompile(`nameserver\s+/(.*?)/(.*)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil
	}

	domain := strings.TrimSpace(matches[1])
	group := strings.TrimSpace(matches[2])

	ns := &models.Nameserver{
		Group:   group,
		Enabled: true,
	}

	// 检查是否是域名集引用
	if strings.HasPrefix(domain, "domain-set:") {
		ns.IsDomainSet = true
		ns.DomainSetName = strings.TrimPrefix(domain, "domain-set:")
		ns.Domain = domain // 保存完整的引用
	} else {
		ns.IsDomainSet = false
		ns.Domain = domain
	}

	return ns
}

func (p *ConfigParser) parseBasicSetting(line string, settings map[string]string) {
	basicKeys := []string{
		"bind", "cache-size", "prefetch-domain", "serve-expired",
		"force-AAAA-SOA", "dualstack-ip-selection", "rr-ttl-min",
		"rr-ttl-max", "log-level", "log-file", "log-size",
		"audit-enable", "audit-num", "audit-size", "audit-file",
		"speed-check-mode", "expand-ptr-from-address",
	}

	for _, key := range basicKeys {
		if strings.HasPrefix(line, key+" ") || strings.HasPrefix(line, key+":") {
			value := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, key), ":"))
			settings[key] = strings.TrimSpace(value)
			break
		}
	}
}

func (p *ConfigParser) Generate(config *models.SmartDNSConfig) string {
	if config.SourceContent != "" {
		return p.generatePreserving(config)
	}
	return p.generateCanonical(config)
}

func (p *ConfigParser) generateCanonical(config *models.SmartDNSConfig) string {
	var builder strings.Builder
	builder.WriteString("# SmartDNS Configuration\n")
	builder.WriteString("# Auto-generated by SmartDNS Manager\n")
	builder.WriteString(fmt.Sprintf("# Generated at: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 基础设置
	if len(config.BasicSettings) > 0 {
		builder.WriteString("# Basic Settings\n")
		// 按顺序输出重要设置
		orderedKeys := []string{
			"bind", "cache-size", "prefetch-domain", "serve-expired",
			"rr-ttl-min", "rr-ttl-max", "log-level", "log-file", "log-size",
			"audit-enable", "audit-num", "audit-size", "audit-file",
			"force-AAAA-SOA", "dualstack-ip-selection", "speed-check-mode",
			"expand-ptr-from-address",
		}
		for _, key := range orderedKeys {
			if value, ok := config.BasicSettings[key]; ok {
				builder.WriteString(fmt.Sprintf("%s %s\n", key, value))
			}
		}
		// 输出其他设置
		keys := make([]string, 0, len(config.BasicSettings))
		for key := range config.BasicSettings {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := config.BasicSettings[key]
			found := false
			for _, orderedKey := range orderedKeys {
				if key == orderedKey {
					found = true
					break
				}
			}
			if !found {
				builder.WriteString(fmt.Sprintf("%s %s\n", key, value))
			}
		}
		builder.WriteString("\n")
	}

	// DNS 服务器
	if len(config.Servers) > 0 {
		builder.WriteString("# DNS Servers\n")
		for _, server := range config.Servers {
			builder.WriteString(RenderServer(server) + "\n")
		}
		builder.WriteString("\n")
	}

	// 地址映射和CNAME
	if len(config.Addresses) > 0 {
		builder.WriteString("# Address Mappings\n")
		for _, addr := range config.Addresses {
			if addr.Type == "cname" {
				// CNAME 格式
				builder.WriteString(fmt.Sprintf("cname /%s/%s", addr.Domain, addr.CNAME))
			} else {
				// Address 格式
				builder.WriteString(fmt.Sprintf("address /%s/%s", addr.Domain, addr.IP))
			}
			// 添加注释
			if addr.Comment != "" {
				builder.WriteString(fmt.Sprintf(" # %s", addr.Comment))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// Domain Sets
	if len(config.DomainSets) > 0 {
		builder.WriteString("# Domain Sets\n")
		for _, ds := range config.DomainSets {
			builder.WriteString(fmt.Sprintf("domain-set -name %s -file %s\n", ds.Name, ds.FilePath))
		}
		builder.WriteString("\n")
	}

	// Domain Rules
	if len(config.DomainRules) > 0 {
		builder.WriteString("# Domain Rules\n")
		for _, rule := range config.DomainRules {
			var domain string
			if rule.IsDomainSet {
				domain = fmt.Sprintf("domain-set:%s", rule.DomainSetName)
			} else {
				domain = rule.Domain
			}
			line := fmt.Sprintf("domain-rules /%s/", domain)

			// 构建选项
			var opts []string
			addOptIfNotExists := func(newOpt string) {
				for _, existing := range opts {
					if existing == newOpt {
						return
					}
				}
				opts = append(opts, newOpt)
			}

			if rule.Address != "" {
				addOptIfNotExists(fmt.Sprintf("-address %s", rule.Address))
			}
			if rule.Nameserver != "" {
				addOptIfNotExists(fmt.Sprintf("-nameserver %s", rule.Nameserver))
			}
			if rule.SpeedCheckMode != "" {
				opts = append(opts, fmt.Sprintf("-speed-check-mode %s", rule.SpeedCheckMode))
			}
			if rule.OtherOptions != "" {
				opts = append(opts, rule.OtherOptions)
			}

			if len(opts) > 0 {
				line += " " + strings.Join(opts, " ")
			}
			builder.WriteString(line + "\n")
		}
		builder.WriteString("\n")
	}

	// Nameserver Rules
	if len(config.Nameservers) > 0 {
		builder.WriteString("# Nameserver Rules\n")
		for _, ns := range config.Nameservers {
			var domain string
			if ns.IsDomainSet {
				domain = fmt.Sprintf("domain-set:%s", ns.DomainSetName)
			} else {
				domain = ns.Domain
			}
			builder.WriteString(fmt.Sprintf("nameserver /%s/%s\n", domain, ns.Group))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
