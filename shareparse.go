package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParsedNode 是分享链接解析出的节点：Node 为完整 mihomo proxy 字段，
// 其余为展示字段。
type ParsedNode struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Server string         `json:"server"`
	Port   int            `json:"port"`
	Node   map[string]any `json:"node"`
}

// LineError 逐行报错（line 为原文 1 起始行号）。
type LineError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// ParseLinks 逐行解析分享链接文本。v1 支持 vless://、hysteria2://、hy2://；
// XHTTP+TLS 已预留槽位（见 parseVLESS 的 transport 分支），本版不实现。
func ParseLinks(text string) ([]ParsedNode, []LineError) {
	var out []ParsedNode
	var errs []LineError
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		n, err := parseLink(line)
		if err != nil {
			errs = append(errs, LineError{Line: i + 1, Message: err.Error()})
			continue
		}
		out = append(out, *n)
	}
	return out, errs
}

func parseLink(s string) (*ParsedNode, error) {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		return nil, fmt.Errorf("不是有效链接")
	}
	switch u.Scheme {
	case "vless":
		return parseVLESS(u)
	case "hysteria2", "hy2":
		return parseHy2(u)
	default:
		return nil, fmt.Errorf("不支持的链接类型 %s://（本版支持 vless://、hysteria2://、hy2://；其他请改用 YAML 片段）", u.Scheme)
	}
}

func linkName(u *url.URL, def string) string {
	if name, err := url.PathUnescape(u.Fragment); err == nil {
		if name = strings.TrimSpace(name); name != "" {
			return name
		}
	}
	return def
}

func hostPort(u *url.URL) (string, int, error) {
	host := u.Hostname()
	if host == "" {
		return "", 0, fmt.Errorf("缺少服务器地址")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("端口 %q 非法", u.Port())
	}
	return host, port, nil
}

func parseVLESS(u *url.URL) (*ParsedNode, error) {
	uuid := u.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("vless 缺少 uuid")
	}
	host, port, err := hostPort(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	node := map[string]any{
		"name":   linkName(u, host+":"+strconv.Itoa(port)),
		"type":   "vless",
		"server": host,
		"port":   port,
		"uuid":   uuid,
		"udp":    true,
	}
	if f := q.Get("flow"); f != "" {
		node["flow"] = f
	}

	switch q.Get("security") {
	case "reality":
		node["tls"] = true
		node["servername"] = firstOf(q.Get("sni"), q.Get("peer"))
		ro := map[string]any{}
		if v := q.Get("pbk"); v != "" {
			ro["public-key"] = v
		}
		if v := q.Get("sid"); v != "" {
			ro["short-id"] = v
		}
		node["reality-opts"] = ro
		node["client-fingerprint"] = firstOf(q.Get("fp"), "chrome")
	case "tls":
		node["tls"] = true
		if v := firstOf(q.Get("sni"), q.Get("peer")); v != "" {
			node["servername"] = v
		}
		if v := q.Get("alpn"); v != "" {
			node["alpn"] = strings.Split(v, ",")
		}
		if v := q.Get("fp"); v != "" {
			node["client-fingerprint"] = v
		}
		if v := q.Get("insecure"); v == "1" || v == "true" {
			node["skip-cert-verify"] = true
		}
	}

	// 传输层。xhttp 为 XHTTP+TLS（CDN 回源）预留槽位，本版显式报未实现。
	switch q.Get("type") {
	case "", "tcp":
		// 默认 tcp，不写 network
	case "ws":
		node["network"] = "ws"
		ws := map[string]any{}
		if v := q.Get("path"); v != "" {
			ws["path"] = v
		}
		if v := q.Get("host"); v != "" {
			ws["headers"] = map[string]any{"Host": v}
		}
		node["ws-opts"] = ws
	case "grpc":
		node["network"] = "grpc"
		node["grpc-opts"] = map[string]any{"grpc-service-name": q.Get("serviceName")}
	case "xhttp", "splithttp":
		return nil, fmt.Errorf("vless+type=xhttp（XHTTP+TLS 方案）已预留槽位，本版未实现")
	default:
		return nil, fmt.Errorf("vless 未知传输类型 %q", q.Get("type"))
	}

	return &ParsedNode{
		Name: node["name"].(string), Type: "vless",
		Server: host, Port: port, Node: node,
	}, nil
}

func parseHy2(u *url.URL) (*ParsedNode, error) {
	password, err := url.QueryUnescape(u.User.String())
	if err != nil || password == "" {
		return nil, fmt.Errorf("hysteria2 缺少密码")
	}
	host, port, err := hostPort(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	node := map[string]any{
		"name":     linkName(u, host+":"+strconv.Itoa(port)),
		"type":     "hysteria2",
		"server":   host,
		"port":     port,
		"password": password,
	}
	if v := q.Get("sni"); v != "" {
		node["sni"] = v
	}
	if v := q.Get("insecure"); v == "1" || v == "true" {
		node["skip-cert-verify"] = true
	}
	if v := q.Get("obfs"); v != "" {
		node["obfs"] = v
		if v := q.Get("obfs-password"); v != "" {
			node["obfs-password"] = v
		}
	}
	if v := q.Get("mport"); v != "" {
		node["ports"] = v
	}
	if v := q.Get("alpn"); v != "" {
		node["alpn"] = strings.Split(v, ",")
	}

	return &ParsedNode{
		Name: node["name"].(string), Type: "hysteria2",
		Server: host, Port: port, Node: node,
	}, nil
}

func firstOf(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
