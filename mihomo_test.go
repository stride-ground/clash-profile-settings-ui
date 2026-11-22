package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func issueLevels(m *Model) map[string][]string {
	out := map[string][]string{}
	for _, i := range Validate(m, nil) {
		out[i.Level] = append(out[i.Level], i.Where+": "+i.Message)
	}
	return out
}

func hasIssue(issues map[string][]string, level, substr string) bool {
	for _, s := range issues[level] {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func baseModel() *Model {
	return &Model{
		Proxies: []ProxyEntry{
			{Name: "n1", Type: "vless", Server: "1.1.1.1", Port: 443},
			{Name: "n2", Type: "hysteria2", Server: "2.2.2.2", Port: 443},
		},
		Groups: []GroupEntry{
			{Name: "auto", Type: "url-test", Proxies: []string{"n1", "n2"},
				URL: "https://www.gstatic.com/generate_204", Interval: 300},
		},
		Listeners: []ListenerEntry{
			{Name: "p1", Type: "socks", Port: 12001, Listen: "127.0.0.1", Proxy: "auto"},
		},
		Mode: "rule", MixedPort: 7890,
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Model)
		level string
		sub   string
	}{
		{"listen 0.0.0.0 报 error", func(m *Model) { m.Listeners[0].Listen = "0.0.0.0" }, "error", "0.0.0.0"},
		{"端口重复报 error", func(m *Model) {
			m.Listeners = append(m.Listeners, ListenerEntry{Name: "p2", Type: "socks", Port: 12001, Listen: "127.0.0.1", Proxy: "auto"})
		}, "error", "重复"},
		{"绑定不存在目标报 error", func(m *Model) { m.Listeners[0].Proxy = "ghost" }, "error", "不存在"},
		{"绑单节点 warn", func(m *Model) { m.Listeners[0].Proxy = "n1" }, "warn", "url-test 组"},
		{"url-test 单成员 info", func(m *Model) { m.Groups[0].Proxies = []string{"n1"} }, "info", "1 个成员"},
		{"无 listeners warn", func(m *Model) { m.Listeners = nil }, "warn", "没有配置任何出口端口"},
		{"allow-lan warn", func(m *Model) { m.AllowLan = true }, "warn", "allow-lan"},
		{"mixed-port 冲突 error", func(m *Model) { m.Listeners[0].Port = 7890 }, "error", "mixed-port 冲突"},
		{"组成员不存在 error", func(m *Model) { m.Groups[0].Proxies = []string{"n1", "ghost"} }, "error", "ghost"},
		{"非法类型 error", func(m *Model) { m.Listeners[0].Type = "weird" }, "error", "未知类型"},
		{"组循环 A→B→A error", func(m *Model) {
			m.Groups = append(m.Groups,
				GroupEntry{Name: "gB", Type: "select", Proxies: []string{"auto"}},
			)
			m.Groups[0].Proxies = append(m.Groups[0].Proxies, "gB")
		}, "error", "循环"},
	}
	for _, c := range cases {
		m := baseModel()
		c.mut(m)
		if !hasIssue(issueLevels(m), c.level, c.sub) {
			t.Errorf("%s: 期望 %s 含 %q，实际 %+v", c.name, c.level, c.sub, issueLevels(m))
		}
	}
	// 基准模型不应有 error
	if errs := issueLevels(baseModel())["error"]; len(errs) > 0 {
		t.Errorf("基准模型不应有 error: %v", errs)
	}
}

const sampleYAML = `# 注释行
mixed-port: 7890
allow-lan: false
mode: rule
proxies:
  - {name: n1, type: vless, server: 1.1.1.1, port: 443, uuid: "u-1", flow: xtls-rprx-vision}
  - {name: n2, type: hysteria2, server: "h.example.com", port: 8443, password: pw}
proxy-groups:
  - name: auto
    type: url-test
    proxies: [n1, n2]
    url: https://www.gstatic.com/generate_204
    interval: 300
listeners:
  - {name: p1, type: socks, port: 12001, listen: 127.0.0.1, proxy: auto}
rules:
  - MATCH,DIRECT
dns:
  enable: true
`

func TestLoadApplyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(path, []byte(sampleYAML), 0644); err != nil {
		t.Fatal(err)
	}
	d, err := LoadDoc(path)
	if err != nil {
		t.Fatal(err)
	}
	m := d.Model()
	if len(m.Proxies) != 2 || len(m.Groups) != 1 || len(m.Listeners) != 1 {
		t.Fatalf("解析数量不对: %+v", m)
	}
	if m.Proxies[0].Name != "n1" || m.Proxies[0].Type != "vless" {
		t.Fatalf("节点字段不对: %+v", m.Proxies[0])
	}

	// 修改 listener 端口 + 新增组 + 新增 raw 节点
	m.Listeners[0].Port = 13001
	m.Groups = append(m.Groups, GroupEntry{Name: "sel", Type: "select", Proxies: []string{"n1"}})
	m.Proxies = append(m.Proxies, ProxyEntry{Raw: `{name: n3, type: vless, server: 3.3.3.3, port: 443, uuid: "u-3"}`})
	if err := d.Apply(m); err != nil {
		t.Fatal(err)
	}
	out, err := d.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"port: 13001", "name: sel", "name: n3",
		"# 注释行",                  // 注释保留
		"flow: xtls-rprx-vision", // 节点未管理字段保留
		"dns:", "enable: true",   // 未管理顶层字段保留
		"MATCH,DIRECT",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("输出缺少 %q:\n%s", want, s)
		}
	}
	// 无 BOM
	if strings.HasPrefix(s, "\ufeff") {
		t.Error("输出不应带 BOM")
	}
	// Marshal 输出可再解析且新值生效（往返无损）
	path2 := filepath.Join(dir, "out.yaml")
	if err := os.WriteFile(path2, out, 0644); err != nil {
		t.Fatal(err)
	}
	d2, err := LoadDoc(path2)
	if err != nil {
		t.Fatalf("输出应可再解析: %v", err)
	}
	m2 := d2.Model()
	if m2.Listeners[0].Port != 13001 || len(m2.Groups) != 2 || len(m2.Proxies) != 3 {
		t.Errorf("往返后值不对: %+v", m2)
	}
}

func TestSaveBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(path, []byte(sampleYAML), 0644); err != nil {
		t.Fatal(err)
	}
	d, _ := LoadDoc(path)
	m := d.Model()
	m.Listeners[0].Port = 14001
	if err := d.Apply(m); err != nil {
		t.Fatal(err)
	}
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("应生成 .bak: %v", err)
	}
	if string(bak) != sampleYAML {
		t.Error(".bak 应为原始内容")
	}
	cur, _ := os.ReadFile(path)
	if !strings.Contains(string(cur), "port: 14001") {
		t.Error("新文件应含修改")
	}
}

func TestBOMStrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bom.yaml")
	if err := os.WriteFile(path, append([]byte("\xef\xbb\xbf"), []byte(sampleYAML)...), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDoc(path); err != nil {
		t.Fatalf("带 BOM 文件应可解析: %v", err)
	}
}

func TestParseFrag(t *testing.T) {
	if _, err := parseFrag(`{name: x, type: vless, server: s, port: 1}`); err != nil {
		t.Errorf("行内映射应可解析: %v", err)
	}
	if _, err := parseFrag("name: x\ntype: vless\nserver: s\nport: 1"); err != nil {
		t.Errorf("块映射应可解析: %v", err)
	}
	if _, err := parseFrag(`- a
- b`); err == nil {
		t.Error("非标量序列元素应报错")
	}
	if _, err := parseFrag(``); err == nil {
		t.Error("空应报错")
	}
}

func TestParseFragsSeq(t *testing.T) {
	// 配置文件里整段复制的 - name: 列表项
	single, err := parseFrags(`- name: demo-sg
  type: vless
  server: vless1.demo-example.com
  port: 31085
  uuid: aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
  tls: true
  servername: th.bing.com
  reality-opts:
    public-key: DEMO_PUBLIC_KEY_wwwwwwwwwwwwwwwwwwwww
    short-id: ab01cd
  network: tcp
  client-fingerprint: chrome`)
	if err != nil || len(single) != 1 {
		t.Fatalf("单列表项应解析为 1 节点: %v len=%d", err, len(single))
	}
	if getStr(single[0], "name") != "demo-sg" || getStr(single[0], "servername") != "th.bing.com" {
		t.Error("列表项字段读取不对")
	}
	// 多节点序列
	multi, err := parseFrags(`- {name: a, type: vless, server: s1, port: 1}
- {name: b, type: hysteria2, server: s2, port: 2}`)
	if err != nil || len(multi) != 2 {
		t.Fatalf("双节点序列应解析为 2: %v len=%d", err, len(multi))
	}
}

func TestValidateGroupCycleMessages(t *testing.T) {
	m := baseModel()
	m.Groups = append(m.Groups, GroupEntry{Name: "gB", Type: "select", Proxies: []string{"auto"}})
	m.Groups[0].Proxies = append(m.Groups[0].Proxies, "gB")
	found := false
	for _, i := range Validate(m, nil) {
		if i.Level == "error" && strings.Contains(i.Message, "循环") && strings.Contains(i.Message, "→") {
			found = true
		}
	}
	if !found {
		t.Error("应报含路径的循环 error")
	}
}

func TestProxyEntryNodePriority(t *testing.T) {
	d := NewDoc()
	m := d.Model()
	m.Proxies = []ProxyEntry{{
		Node: map[string]any{"name": "fromNode", "type": "vless", "server": "9.9.9.9", "port": 443, "uuid": "x"},
	}}
	if err := d.Apply(m); err != nil {
		t.Fatal(err)
	}
	if got := getStr(d.proxies[0], "name"); got != "fromNode" {
		t.Errorf("Node 字段应优先生效，got %q", got)
	}
}
