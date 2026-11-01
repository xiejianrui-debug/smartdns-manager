package services

import (
	"strings"

	"smartdns-manager/models"
)

// renderedEntries 复用生成器，避免维护两份指令序列化规则。
func (p *ConfigParser) renderedEntries(c *models.SmartDNSConfig) map[string][]string {
	copy := *c
	copy.SourceContent = ""
	entries := map[string][]string{}
	for _, line := range strings.Split(p.generateCanonical(&copy), "\n") {
		if kind := sourceKind(line, c); kind != "" {
			entries[kind] = append(entries[kind], line)
		}
	}
	return entries
}

func sourceKind(line string, c *models.SmartDNSConfig) string {
	f := strings.Fields(line)
	if len(f) == 0 || strings.HasPrefix(f[0], "#") {
		return ""
	}
	if isServerDirective(f[0]) && len(f) >= 2 {
		return "server"
	}
	switch f[0] {
	case "address", "cname":
		return "address"
	case "domain-set":
		return "domain-set"
	case "domain-rules":
		return "domain-rules"
	case "nameserver":
		return "nameserver"
	}
	if _, ok := c.BasicSettings[f[0]]; ok {
		return "basic:" + f[0]
	}
	return ""
}

// generatePreserving 保留未建模指令、注释、重复监听项和规则组的原始位置。
func (p *ConfigParser) generatePreserving(c *models.SmartDNSConfig) string {
	base, _ := p.Parse(c.SourceContent)
	old, updated := p.renderedEntries(base), p.renderedEntries(c)
	lines := strings.Split(strings.TrimSuffix(c.SourceContent, "\n"), "\n")
	lastBasic := map[string]int{}
	depth := 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "group-begin ") {
			depth++
			continue
		}
		if t == "group-end" {
			depth--
			continue
		}
		if depth > 0 {
			continue
		}
		kind := sourceKind(t, base)
		if strings.HasPrefix(kind, "basic:") {
			lastBasic[kind] = i
		}
	}
	counts := map[string]int{}
	var output []string
	depth = 0
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "group-begin ") {
			depth++
			output = append(output, raw)
			continue
		}
		if line == "group-end" {
			depth--
			output = append(output, raw)
			continue
		}
		if depth > 0 {
			output = append(output, raw)
			continue
		}
		kind := sourceKind(line, base)
		if kind == "" {
			output = append(output, raw)
			continue
		}
		current := updated[kind]
		if strings.HasPrefix(kind, "basic:") {
			if len(current) == 0 {
				continue
			}
			if current[0] == old[kind][0] || lastBasic[kind] != i {
				output = append(output, raw)
			} else {
				output = append(output, current[0])
			}
			counts[kind] = len(current)
			continue
		}
		index := counts[kind]
		if index < len(current) {
			if kind != "domain-rules" && index < len(old[kind]) && current[index] == old[kind][index] {
				output = append(output, raw)
			} else {
				output = append(output, current[index])
			}
		}
		counts[kind]++
		if counts[kind] == len(old[kind]) && len(current) > counts[kind] {
			output = append(output, current[counts[kind]:]...)
			counts[kind] = len(current)
		}
	}
	sections := []struct{ kind, heading string }{{"server", "DNS Servers"}, {"address", "Address Mappings"}, {"domain-set", "Domain Sets"}, {"domain-rules", "Domain Rules"}, {"nameserver", "Nameserver Rules"}}
	for _, section := range sections {
		if counts[section.kind] >= len(updated[section.kind]) {
			continue
		}
		added := updated[section.kind][counts[section.kind]:]
		at := -1
		for i, line := range output {
			if strings.TrimSpace(line) == "# "+section.heading {
				at = i + 1
				break
			}
		}
		if at < 0 {
			output = append(output, "", "# "+section.heading)
			output = append(output, added...)
		} else {
			tail := append([]string{}, output[at:]...)
			output = append(append(output[:at], added...), tail...)
		}
	}
	// 新增基础项只加入原先不存在的键，稳定按名称输出。
	var basicAdded []string
	for _, line := range p.basicLines(c) {
		kind := sourceKind(line, c)
		if counts[kind] == 0 {
			basicAdded = append(basicAdded, line)
			counts[kind]++
		}
	}
	if len(basicAdded) > 0 {
		at := -1
		for i, line := range output {
			if strings.TrimSpace(line) == "# Basic Settings" {
				at = i + 1
				break
			}
		}
		if at >= 0 {
			tail := append([]string{}, output[at:]...)
			output = append(append(output[:at], basicAdded...), tail...)
		} else {
			output = append(append([]string{"# Basic Settings"}, basicAdded...), output...)
		}
	}
	return strings.Join(output, "\n") + "\n"
}

func (p *ConfigParser) basicLines(c *models.SmartDNSConfig) []string {
	copy := &models.SmartDNSConfig{BasicSettings: c.BasicSettings}
	var out []string
	for _, line := range strings.Split(p.generateCanonical(copy), "\n") {
		if strings.HasPrefix(sourceKind(line, c), "basic:") {
			out = append(out, line)
		}
	}
	return out
}
