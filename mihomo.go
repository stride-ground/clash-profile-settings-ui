package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------- 与前端交互的模型 ----------

type ProxyEntry struct {
	OrigIndex *int           `json:"origIndex,omitempty"` // 指向原文件 proxies[i]，保留完整字段
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Server    string         `json:"server"`
	Port      int            `json:"port"`
	Reality   bool           `json:"reality,omitempty"` // 含 reality-opts（用于缺 flow 预警）
	Flow      string         `json:"flow,omitempty"`
	Raw       string         `json:"raw,omitempty"`  // 新节点：粘贴的 mihomo YAML 片段
	Node      map[string]any `json:"node,omitempty"` // 新节点：链接解析的结构化产物（优先于 Raw）
}

type GroupEntry struct {
	OrigIndex *int     `json:"origIndex,omitempty"`
	Name      string   `json:"name"`
	Type      string   `json:"type"` // select / url-test / fallback / load-balance / relay
	Proxies   []string `json:"proxies"`
	Use       []string `json:"use,omitempty"`
	URL       string   `json:"url,omitempty"`
	Interval  int      `json:"interval,omitempty"`
	Tolerance int      `json:"tolerance,omitempty"`
	Lazy      bool     `json:"lazy,omitempty"`
}

type ListenerEntry struct {
	OrigIndex *int   `json:"origIndex,omitempty"`
	Name      string `json:"name"`
	Type      string `json:"type"` // socks / mixed / http
	Port      int    `json:"port"`
	Listen    string `json:"listen"`
	Proxy     string `json:"proxy"` // 节点名或策略组名；空 = 按规则分流
}

type Model struct {
	Proxies   []ProxyEntry    `json:"proxies"`
	Groups    []GroupEntry    `json:"groups"`
	Listeners []ListenerEntry `json:"listeners"`
	MixedPort int             `json:"mixedPort"`
	AllowLan  bool            `json:"allowLan"`
	Mode      string          `json:"mode"`
}

type Issue struct {
	Level   string `json:"level"` // error / warn / info
	Where   string `json:"where"`
	Message string `json:"message"`
}

// ---------- yaml.Node 工具 ----------

func mapGet(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func mapSet(m *yaml.Node, key string, val *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = val
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
}

func mapDel(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

func enc(v interface{}) *yaml.Node {
	n := &yaml.Node{}
	if err := n.Encode(v); err != nil {
		panic(err)
	}
	return n
}

func getStr(m *yaml.Node, key string) string {
	if n := mapGet(m, key); n != nil {
		return n.Value
	}
	return ""
}

func getInt(m *yaml.Node, key string) int {
	if n := mapGet(m, key); n != nil {
		v, _ := strconv.Atoi(n.Value)
		return v
	}
	return 0
}

func getBool(m *yaml.Node, key string) bool {
	if n := mapGet(m, key); n != nil {
		return n.Value == "true"
	}
	return false
}

func getStrList(m *yaml.Node, key string) []string {
	n := mapGet(m, key)
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		out = append(out, c.Value)
	}
	return out
}

func cloneNode(n *yaml.Node) *yaml.Node {
	c := *n
	if n.Content != nil {
		c.Content = make([]*yaml.Node, len(n.Content))
		for i := range n.Content {
			c.Content[i] = cloneNode(n.Content[i])
		}
	}
	return &c
}

func seqItems(m *yaml.Node, key string) []*yaml.Node {
	n := mapGet(m, key)
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}
	return n.Content
}

// ---------- 文档 ----------

type Doc struct {
	path    string
	exists  bool
	doc     *yaml.Node // DocumentNode
	mapping *yaml.Node
	proxies []*yaml.Node
	groups  []*yaml.Node
	listens []*yaml.Node
}

func newMappingDoc() *Doc {
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping}}
	return &Doc{doc: doc, mapping: mapping}
}

func LoadDoc(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // 去 BOM
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %w", err)
	}
	d := &Doc{path: path, exists: true, doc: &root}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		d.mapping = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = []*yaml.Node{d.mapping}
	} else {
		d.mapping = root.Content[0]
	}
	d.proxies = seqItems(d.mapping, "proxies")
	d.groups = seqItems(d.mapping, "proxy-groups")
	d.listens = seqItems(d.mapping, "listeners")
	return d, nil
}

// NewDoc 新建空白配置（§5 场景的合理默认）
func NewDoc() *Doc {
	d := newMappingDoc()
	mapSet(d.mapping, "mixed-port", enc(7897))
	mapSet(d.mapping, "allow-lan", enc(false))
	mapSet(d.mapping, "mode", enc("rule"))
	mapSet(d.mapping, "log-level", enc("info"))
	mapSet(d.mapping, "proxies", enc([]string{}))
	mapSet(d.mapping, "proxy-groups", enc([]string{}))
	mapSet(d.mapping, "listeners", enc([]string{}))
	mapSet(d.mapping, "rules", enc([]string{"MATCH,DIRECT"}))
	return d
}

func (d *Doc) Model() *Model {
	m := &Model{
		MixedPort: getInt(d.mapping, "mixed-port"),
		AllowLan:  getBool(d.mapping, "allow-lan"),
		Mode:      getStr(d.mapping, "mode"),
	}
	if m.Mode == "" {
		m.Mode = "rule"
	}
	for i, p := range d.proxies {
		idx := i
		m.Proxies = append(m.Proxies, ProxyEntry{
			OrigIndex: &idx,
			Name:      getStr(p, "name"),
			Type:      getStr(p, "type"),
			Server:    getStr(p, "server"),
			Port:      getInt(p, "port"),
			Reality:   mapGet(p, "reality-opts") != nil,
			Flow:      getStr(p, "flow"),
		})
	}
	for i, g := range d.groups {
		idx := i
		m.Groups = append(m.Groups, GroupEntry{
			OrigIndex: &idx,
			Name:      getStr(g, "name"),
			Type:      getStr(g, "type"),
			Proxies:   getStrList(g, "proxies"),
			Use:       getStrList(g, "use"),
			URL:       getStr(g, "url"),
			Interval:  getInt(g, "interval"),
			Tolerance: getInt(g, "tolerance"),
			Lazy:      getBool(g, "lazy"),
		})
	}
	for i, l := range d.listens {
		idx := i
		m.Listeners = append(m.Listeners, ListenerEntry{
			OrigIndex: &idx,
			Name:      getStr(l, "name"),
			Type:      getStr(l, "type"),
			Port:      getInt(l, "port"),
			Listen:    getStr(l, "listen"),
			Proxy:     getStr(l, "proxy"),
		})
	}
	return m
}

func (d *Doc) proxyNode(i int) *yaml.Node {
	if i >= 0 && i < len(d.proxies) {
		return d.proxies[i]
	}
	return nil
}
func (d *Doc) groupNode(i int) *yaml.Node {
	if i >= 0 && i < len(d.groups) {
		return d.groups[i]
	}
	return nil
}
func (d *Doc) listenNode(i int) *yaml.Node {
	if i >= 0 && i < len(d.listens) {
		return d.listens[i]
	}
	return nil
}

func newMapNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

// parseFrags 解析节点 YAML，接受三种形态：
// 裸映射（{name:…} 或多行块）、单/多列表项（- name: …）、整个节点序列。
func parseFrags(raw string) ([]*yaml.Node, error) {
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &n); err != nil {
		return nil, err
	}
	if len(n.Content) == 0 {
		return nil, fmt.Errorf("空内容")
	}
	f := n.Content[0]
	switch f.Kind {
	case yaml.MappingNode:
		return []*yaml.Node{f}, nil
	case yaml.SequenceNode:
		if len(f.Content) == 0 {
			return nil, fmt.Errorf("空序列")
		}
		for _, item := range f.Content {
			if item.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("序列元素必须是节点映射（{name:…, type:…}）")
			}
		}
		return f.Content, nil
	default:
		return nil, fmt.Errorf("需要节点映射或列表（如 {name: x, type: vless, …} 或 - name: x …）")
	}
}

func parseFrag(raw string) (*yaml.Node, error) {
	ns, err := parseFrags(raw)
	if err != nil {
		return nil, err
	}
	if len(ns) != 1 {
		return nil, fmt.Errorf("需要恰好一个节点映射")
	}
	return ns[0], nil
}

// setOrDel：val 为零值时删除该 key
func setOrDel(m *yaml.Node, key string, val interface{}, zero interface{}) {
	if val == zero {
		mapDel(m, key)
		return
	}
	mapSet(m, key, enc(val))
}

// Apply 把前端模型写回 yaml 树，保留未管理的字段（rules/dns/订阅头等）
func (d *Doc) Apply(m *Model) error {
	// proxies
	pseq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, p := range m.Proxies {
		if p.OrigIndex != nil {
			if n := d.proxyNode(*p.OrigIndex); n != nil {
				pseq.Content = append(pseq.Content, n)
				continue
			}
			return fmt.Errorf("节点 origIndex=%d 越界", *p.OrigIndex)
		}
		var ns []*yaml.Node
		if p.Node != nil {
			ns = []*yaml.Node{enc(p.Node)}
		} else {
			var err error
			ns, err = parseFrags(p.Raw)
			if err != nil {
				return fmt.Errorf("新节点 YAML 解析失败: %w", err)
			}
		}
		for _, n := range ns {
			if getStr(n, "name") == "" {
				return fmt.Errorf("新节点缺少 name 字段")
			}
			pseq.Content = append(pseq.Content, n)
		}
	}
	mapSet(d.mapping, "proxies", pseq)

	// proxy-groups
	gseq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, g := range m.Groups {
		var n *yaml.Node
		if g.OrigIndex != nil {
			if o := d.groupNode(*g.OrigIndex); o != nil {
				n = cloneNode(o)
			} else {
				return fmt.Errorf("策略组 origIndex=%d 越界", *g.OrigIndex)
			}
		} else {
			n = newMapNode()
		}
		mapSet(n, "name", enc(g.Name))
		mapSet(n, "type", enc(g.Type))
		mapSet(n, "proxies", enc(g.Proxies))
		setOrDel(n, "use", g.Use, nil)
		setOrDel(n, "url", g.URL, "")
		setOrDel(n, "interval", g.Interval, 0)
		setOrDel(n, "tolerance", g.Tolerance, 0)
		setOrDel(n, "lazy", g.Lazy, false)
		gseq.Content = append(gseq.Content, n)
	}
	mapSet(d.mapping, "proxy-groups", gseq)

	// listeners
	lseq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, l := range m.Listeners {
		var n *yaml.Node
		if l.OrigIndex != nil {
			if o := d.listenNode(*l.OrigIndex); o != nil {
				n = cloneNode(o)
			} else {
				return fmt.Errorf("监听器 origIndex=%d 越界", *l.OrigIndex)
			}
		} else {
			n = newMapNode()
		}
		mapSet(n, "name", enc(l.Name))
		mapSet(n, "type", enc(l.Type))
		mapSet(n, "port", enc(l.Port))
		mapSet(n, "listen", enc(l.Listen))
		setOrDel(n, "proxy", l.Proxy, "")
		lseq.Content = append(lseq.Content, n)
	}
	mapSet(d.mapping, "listeners", lseq)

	// 顶层常规项
	setOrDel(d.mapping, "mixed-port", m.MixedPort, 0)
	mapSet(d.mapping, "allow-lan", enc(m.AllowLan))
	if m.Mode != "" {
		mapSet(d.mapping, "mode", enc(m.Mode))
	}
	// 新建配置时保证 rules 存在
	if mapGet(d.mapping, "rules") == nil {
		target := "DIRECT"
		if len(m.Groups) > 0 {
			target = m.Groups[0].Name
		}
		mapSet(d.mapping, "rules", enc([]string{"MATCH," + target}))
	}
	// 索引随重建的序列刷新，保证后续 origIndex 一致
	d.proxies = seqItems(d.mapping, "proxies")
	d.groups = seqItems(d.mapping, "proxy-groups")
	d.listens = seqItems(d.mapping, "listeners")
	return nil
}

// Clone 深拷贝文档（preview 不落盘、不改当前会话态）
func (d *Doc) Clone() *Doc {
	root := cloneNode(d.doc)
	nd := &Doc{path: d.path, exists: d.exists, doc: root}
	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		nd.mapping = root.Content[0]
	} else {
		nd.mapping = newMapNode()
		root.Content = []*yaml.Node{nd.mapping}
	}
	nd.proxies = seqItems(nd.mapping, "proxies")
	nd.groups = seqItems(nd.mapping, "proxy-groups")
	nd.listens = seqItems(nd.mapping, "listeners")
	return nd
}

func (d *Doc) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	e := yaml.NewEncoder(&buf)
	e.SetIndent(2)
	if err := e.Encode(d.doc); err != nil {
		return nil, err
	}
	if err := e.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Save 备份原文件后写回（无 BOM UTF-8）
func (d *Doc) Save() error {
	out, err := d.Marshal()
	if err != nil {
		return err
	}
	if d.exists {
		if orig, err := os.ReadFile(d.path); err == nil {
			_ = os.WriteFile(d.path+".bak", orig, 0644)
		}
	}
	return os.WriteFile(d.path, out, 0644)
}

// ---------- 校验（§5 场景规则） ----------

func Validate(m *Model, portBusy func(int) bool) []Issue {
	var issues []Issue
	add := func(level, where, msg string) {
		issues = append(issues, Issue{level, where, msg})
	}

	// mihomo 内建出站，可作为组成员/绑定目标
	builtins := map[string]bool{
		"DIRECT": true, "REJECT": true, "REJECT-DROP": true,
		"PASS": true, "GLOBAL": true, "COMPATIBLE": true,
	}
	names := map[string]bool{}
	for _, p := range m.Proxies {
		if p.Name != "" {
			names[p.Name] = true
		}
		// vless+Reality 缺 flow：多数服务端强制 xtls-rprx-vision，缺失握手失败
		reality := p.Reality
		flow := p.Flow
		if p.Node != nil {
			if _, ok := p.Node["reality-opts"]; ok {
				reality = true
			}
			if flow == "" {
				flow, _ = p.Node["flow"].(string)
			}
		}
		if p.Type == "vless" && reality && flow == "" {
			add("warn", "节点:"+p.Name, "vless+Reality 节点缺少 flow 字段；多数服务端要求 flow: xtls-rprx-vision，缺失会握手失败（延迟测试 error/timeout）")
		}
	}
	groupNames := map[string]bool{}
	for _, g := range m.Groups {
		if g.Name == "" {
			add("error", "策略组", "存在未命名的策略组")
			continue
		}
		if groupNames[g.Name] {
			add("error", "策略组:"+g.Name, "策略组名重复")
		}
		groupNames[g.Name] = true
		names[g.Name] = true
	}

	validGroupType := map[string]bool{
		"select": true, "url-test": true, "fallback": true,
		"load-balance": true, "relay": true,
	}
	for _, g := range m.Groups {
		w := "策略组:" + g.Name
		if !validGroupType[g.Type] {
			add("error", w, fmt.Sprintf("未知类型 %q（可选 select/url-test/fallback/load-balance/relay）", g.Type))
		}
		if len(g.Proxies) == 0 && len(g.Use) == 0 {
			add("warn", w, "组内无成员节点")
		}
		for _, p := range g.Proxies {
			if !names[p] && !groupNames[p] && !builtins[p] {
				add("error", w, fmt.Sprintf("成员 %q 不存在于节点或其他组", p))
			}
		}
		if g.Type == "url-test" || g.Type == "fallback" || g.Type == "load-balance" {
			if g.URL == "" {
				add("warn", w, "未设测速 url，mihomo 将用默认 http://www.gstatic.com/generate_204")
			}
			if g.Interval == 0 {
				add("info", w, "未设 interval，默认 600 秒")
			}
		}
		if g.Type == "url-test" && len(g.Proxies) == 1 {
			add("info", w, "url-test 组只有 1 个成员，自动容灾不会生效")
		}
	}

	// 组引用成环检测：A→B→A 会让 mihomo 解析/选路异常
	adj := map[string][]string{}
	for _, g := range m.Groups {
		for _, p := range g.Proxies {
			if groupNames[p] {
				adj[g.Name] = append(adj[g.Name], p)
			}
		}
	}
	state := map[string]int{} // 1=在栈中 2=已完成
	var stack []string
	var walk func(n string)
	walk = func(n string) {
		state[n] = 1
		stack = append(stack, n)
		for _, next := range adj[n] {
			if state[next] == 1 {
				i := len(stack) - 1
				for i >= 0 && stack[i] != next {
					i--
				}
				cycle := append(append([]string{}, stack[i:]...), next)
				add("error", "策略组:"+next, "组引用形成循环 "+strings.Join(cycle, " → "))
			} else if state[next] == 0 {
				walk(next)
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = 2
	}
	for _, g := range m.Groups {
		if state[g.Name] == 0 {
			walk(g.Name)
		}
	}

	seenPorts := map[int]string{}
	listenNames := map[string]bool{}
	validLType := map[string]bool{"socks": true, "mixed": true, "http": true}
	for _, l := range m.Listeners {
		w := "端口:" + orName(l.Name, l.Port)
		listenNames[l.Name] = true
		if !validLType[l.Type] {
			add("error", w, fmt.Sprintf("未知类型 %q（可选 socks/mixed/http）", l.Type))
		}
		if l.Port < 1 || l.Port > 65535 {
			add("error", w, fmt.Sprintf("端口 %d 非法", l.Port))
		} else {
			if prev, dup := seenPorts[l.Port]; dup {
				add("error", w, fmt.Sprintf("端口 %d 与 %s 重复", l.Port, prev))
			}
			seenPorts[l.Port] = w
		}
		if l.Listen != "127.0.0.1" && l.Listen != "localhost" && l.Listen != "::1" {
			add("error", w, fmt.Sprintf("listen=%q 会把无认证代理口暴露到局域网，§5 要求 127.0.0.1", l.Listen))
		}
		if l.Proxy == "" {
			add("info", w, "未绑定 proxy，流量将走 rules 全局分流（指纹浏览器场景建议绑定固定节点/组）")
		} else if !names[l.Proxy] && !builtins[l.Proxy] {
			add("error", w, fmt.Sprintf("绑定的 %q 不存在于节点或策略组", l.Proxy))
		} else if !groupNames[l.Proxy] && !builtins[l.Proxy] {
			add("warn", w, "绑定的是单节点；建议指向 url-test 组实现单节点故障自动切换")
		}
		if l.Port == m.MixedPort && m.MixedPort != 0 {
			add("error", w, fmt.Sprintf("端口 %d 与 mixed-port 冲突", l.Port))
		}
		for _, p := range []int{10808, 7890, 7897, 9090, 9097} {
			if l.Port == p && portBusy != nil && portBusy(p) {
				add("info", w, fmt.Sprintf("本机已有进程监听 %d（可能是正在运行的代理软件），注意占用冲突", p))
			}
		}
	}
	if len(m.Listeners) == 0 {
		add("warn", "listeners", "没有配置任何出口端口；§5 场景需要每个指纹浏览器 Profile 一个端口")
	}
	if m.AllowLan {
		add("warn", "allow-lan", "allow-lan=true 会把代理暴露到局域网，指纹浏览器单机场景建议关闭")
	}
	return issues
}

func orName(name string, port int) string {
	if name != "" {
		return name
	}
	return strconv.Itoa(port)
}
