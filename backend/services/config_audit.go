package services

import (
	"sort"
	"strings"
)

// ConfigLosses 比较有效指令，忽略空白、注释和重复的同值规则参数。
// 这是保守检查：值被替换也会报告，调用方不能把未审查的覆盖当作安全同步。
func ConfigLosses(before, after string) []string {
	p := NewConfigParser()
	canonical := func(text string) map[string]int {
		c, _ := p.Parse(text)
		text = p.Generate(c)
		counts := map[string]int{}
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if fields := strings.Fields(line); len(fields) >= 2 && isServerDirective(fields[0]) {
				server := p.parseServer(line)
				groups := server.Groups
				if !server.ExcludeDefault {
					groups = append(append([]string{}, groups...), "__default__")
				}
				server.Groups = nil
				server.ExcludeDefault = false
				base := server.Type + " " + server.Address + " " + BuildServerOptions(*server)
				for _, group := range groups {
					counts[base+" group="+group]++
				}
				continue
			}
			counts[strings.Join(strings.Fields(line), " ")]++
		}
		return counts
	}
	a, b := canonical(before), canonical(after)
	var losses []string
	for line, count := range a {
		if b[line] < count {
			losses = append(losses, line)
		}
	}
	sort.Strings(losses)
	return losses
}
