package services

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"smartdns-manager/models"
)

var optionTokens = regexp.MustCompile(`"(?:\\.|[^"\\])*"|'[^']*'|\S+`)

func optionValue(s string) string {
	if strings.HasPrefix(s, `"`) {
		if v, err := strconv.Unquote(s); err == nil {
			return v
		}
	}
	if len(s) > 1 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func removeManagedOptions(options string, managed map[string]bool) string {
	fields := optionTokens.FindAllString(options, -1)
	var rest []string
	for i := 0; i < len(fields); i++ {
		if fields[i] == "#" {
			rest = append(rest, fields[i:]...)
			break
		}
		if managed[fields[i]] && i+1 < len(fields) {
			i++
			continue
		}
		rest = append(rest, fields[i])
	}
	return strings.Join(rest, " ")
}

func isServerDirective(s string) bool {
	switch s {
	case "server", "server-tcp", "server-tls", "server-https", "server-quic", "server-h3", "server-http3":
		return true
	}
	return false
}

func ServerType(directive, address string) string {
	if directive != "" && directive != "server" {
		return strings.TrimPrefix(strings.Replace(directive, "server-http3", "server-h3", 1), "server-")
	}
	for _, proto := range []string{"https", "tls", "tcp", "quic", "h3"} {
		if strings.HasPrefix(address, proto+"://") {
			return proto
		}
	}
	return "udp"
}

// HydrateServerOptions 兼容数据库中只保存在 Options 内的历史参数。
func HydrateServerOptions(s *models.DNSServer) {
	fields := optionTokens.FindAllString(s.Options, -1)
	var groups []string
	var rest []string
	for i := 0; i < len(fields); i++ {
		key := fields[i]
		if key == "#" {
			rest = append(rest, fields[i:]...)
			break
		}
		if key == "-exclude-default-group" || key == "-e" {
			s.ExcludeDefault = true
			continue
		}
		if i+1 < len(fields) {
			value := optionValue(fields[i+1])
			switch key {
			case "-group", "-g":
				groups = append(groups, value)
				i++
				continue
			case "-host-ip":
				if s.HostIP == "" {
					s.HostIP = value
				}
				i++
				continue
			case "-tls-host-verify":
				if s.TLSHostVerify == "" {
					s.TLSHostVerify = value
				}
				i++
				continue
			}
		}
		rest = append(rest, fields[i])
	}
	if s.Groups == nil {
		s.Groups = groups
	}
	if s.ExtraOptions == "" {
		s.ExtraOptions = strings.Join(rest, " ")
	}
}

func BuildServerOptions(s models.DNSServer) string {
	var opts []string
	for _, g := range s.Groups {
		opts = append(opts, "-group", quoteOption(g))
	}
	if s.ExcludeDefault {
		opts = append(opts, "-exclude-default-group")
	}
	if s.HostIP != "" {
		opts = append(opts, "-host-ip", quoteOption(s.HostIP))
	}
	if s.TLSHostVerify != "" {
		opts = append(opts, "-tls-host-verify", quoteOption(s.TLSHostVerify))
	}
	if strings.TrimSpace(s.ExtraOptions) != "" {
		opts = append(opts, strings.TrimSpace(s.ExtraOptions))
	}
	return strings.Join(opts, " ")
}

// MergeServerForSync 旧记录缺少高级字段时保留节点的现有参数；新版记录按显式编辑结果下发。
func MergeServerForSync(existing, record models.DNSServer) models.DNSServer {
	HydrateServerOptions(&existing)
	if record.GroupsStr != "" {
		_ = json.Unmarshal([]byte(record.GroupsStr), &record.Groups)
	}
	HydrateServerOptions(&record)
	if record.OptionsVersion == 0 {
		groups := append([]string{}, existing.Groups...)
		for _, group := range record.Groups {
			found := false
			for _, old := range groups {
				if old == group {
					found = true
					break
				}
			}
			if !found {
				groups = append(groups, group)
			}
		}
		record.Groups = groups
		record.ExcludeDefault = existing.ExcludeDefault
		if record.HostIP == "" {
			record.HostIP = existing.HostIP
		}
		if record.TLSHostVerify == "" {
			record.TLSHostVerify = existing.TLSHostVerify
		}
		if record.ExtraOptions == "" {
			record.ExtraOptions = existing.ExtraOptions
		}
	}
	if record.Directive == "" && record.Type == existing.Type {
		record.Directive = existing.Directive
	}
	record.Options = BuildServerOptions(record)
	return record
}

func quoteOption(s string) string {
	if strings.ContainsAny(s, " \t\"'") {
		return strconv.Quote(s)
	}
	return s
}

func RenderServer(s models.DNSServer) string {
	HydrateServerOptions(&s)
	directive := s.Directive
	if !isServerDirective(directive) {
		directive = "server"
		if s.Type != "" && s.Type != "udp" {
			directive += "-" + s.Type
		}
	}
	line := directive + " " + s.Address
	if options := BuildServerOptions(s); options != "" {
		line += " " + options
	}
	return line
}

func ValidateServer(s models.DNSServer) error {
	if s.ExcludeDefault && len(s.Groups) == 0 {
		return fmt.Errorf("排除默认组时必须指定至少一个分组")
	}
	if s.Directive != "" && (!isServerDirective(s.Directive) || (s.Directive != "server" && ServerType(s.Directive, s.Address) != s.Type)) {
		return fmt.Errorf("协议指令与服务器类型不一致")
	}
	if strings.TrimSpace(s.Address) == "" || strings.ContainsAny(s.Address, "\r\n\t ") {
		return fmt.Errorf("服务器地址为空或包含空白字符")
	}
	if strings.Contains(s.Address, "://") && ServerType("server", s.Address) != s.Type {
		return fmt.Errorf("地址协议与服务器类型不一致")
	}
	if !isServerDirective("server-"+s.Type) && s.Type != "udp" {
		return fmt.Errorf("不支持的服务器类型")
	}
	if s.HostIP != "" && net.ParseIP(s.HostIP) == nil {
		return fmt.Errorf("固定连接 IP 格式无效")
	}
	for _, v := range append(append([]string{s.TLSHostVerify, s.ExtraOptions}, s.Groups...), s.NodeIDs) {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("上游参数不能包含换行")
		}
	}
	if strings.ContainsAny(s.TLSHostVerify, " \t") {
		return fmt.Errorf("证书校验域名不能包含空白")
	}
	if s.NodeIDs != "" {
		if strings.TrimSpace(s.NodeIDs) == "null" {
			return fmt.Errorf("节点作用域必须为 JSON 数组")
		}
		var ids []uint
		if err := json.Unmarshal([]byte(s.NodeIDs), &ids); err != nil {
			return fmt.Errorf("节点作用域必须为 JSON 数组")
		}
		for _, id := range ids {
			if id == 0 {
				return fmt.Errorf("节点 ID 必须大于零")
			}
		}
	}
	return nil
}
