package main

import (
	"strings"
	"testing"
)

func TestParseLinks(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr string // 非空则期望 errors 含此子串
		check   func(t *testing.T, n ParsedNode)
	}{
		{
			name:  "vless reality 全参数",
			input: `vless://aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@203.0.113.10:443?security=reality&sni=www.microsoft.com&fp=chrome&pbk=PUBKEY123&sid=ab12&flow=xtls-rprx-vision&type=tcp#%E6%88%91%E7%9A%84VPS`,
			check: func(t *testing.T, n ParsedNode) {
				if n.Name != "我的VPS" || n.Type != "vless" || n.Server != "203.0.113.10" || n.Port != 443 {
					t.Fatalf("展示字段不对: %+v", n)
				}
				nd := n.Node
				if nd["uuid"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" ||
					nd["servername"] != "www.microsoft.com" ||
					nd["client-fingerprint"] != "chrome" ||
					nd["flow"] != "xtls-rprx-vision" || nd["tls"] != true {
					t.Fatalf("reality 字段不对: %+v", nd)
				}
				ro, _ := nd["reality-opts"].(map[string]any)
				if ro["public-key"] != "PUBKEY123" || ro["short-id"] != "ab12" {
					t.Fatalf("reality-opts 不对: %+v", ro)
				}
			},
		},
		{
			name:  "vless ws+tls",
			input: `vless://u1@cdn.example.com:443?security=tls&type=ws&path=%2Fray&host=cdn.example.com&sni=cdn.example.com`,
			check: func(t *testing.T, n ParsedNode) {
				if n.Node["network"] != "ws" || n.Node["tls"] != true || n.Node["servername"] != "cdn.example.com" {
					t.Fatalf("ws/tls 字段不对: %+v", n.Node)
				}
				ws, _ := n.Node["ws-opts"].(map[string]any)
				if ws["path"] != "/ray" {
					t.Fatalf("ws-opts 不对: %+v", ws)
				}
			},
		},
		{
			name:  "vless xhttp 预留报错",
			input: `vless://u1@1.2.3.4:443?security=reality&type=xhttp&path=/x#x`,
			// 注意：url.Parse 对 type=xhttp 正常解析，我们显式报未实现
			wantErr: "xhttp",
		},
		{
			name:  "hysteria2 全参数",
			input: `hysteria2://p%40ssw0rd@vps2.example.com:24443?sni=bing.com&insecure=1&obfs=salamander&obfs-password=obfspw&mport=20000-30000#hy2-%E8%8A%82%E7%82%B9`,
			check: func(t *testing.T, n ParsedNode) {
				if n.Name != "hy2-节点" || n.Type != "hysteria2" || n.Port != 24443 {
					t.Fatalf("展示字段不对: %+v", n)
				}
				nd := n.Node
				if nd["password"] != "p@ssw0rd" || nd["sni"] != "bing.com" ||
					nd["skip-cert-verify"] != true || nd["obfs"] != "salamander" ||
					nd["obfs-password"] != "obfspw" || nd["ports"] != "20000-30000" {
					t.Fatalf("hy2 字段不对: %+v", nd)
				}
			},
		},
		{
			name:  "hy2 别名",
			input: `hy2://pw@1.2.3.4:443`,
			check: func(t *testing.T, n ParsedNode) {
				if n.Type != "hysteria2" || n.Name != "1.2.3.4:443" {
					t.Fatalf("hy2 别名/缺省名不对: %+v", n)
				}
			},
		},
		{
			name:    "ss 不支持",
			input:   `ss://YWVzLTEyOC1nY206cHc=@1.2.3.4:8388#x`,
			wantErr: "不支持",
		},
		{
			name:    "非链接",
			input:   `hello world`,
			wantErr: "不是有效链接",
		},
		{
			name:    "缺端口",
			input:   `vless://u1@1.2.3.4?security=tls`,
			wantErr: "端口",
		},
	}
	for _, c := range cases {
		nodes, errs := ParseLinks(c.input)
		if c.wantErr != "" {
			if len(errs) == 0 || !strings.Contains(errs[0].Message, c.wantErr) {
				t.Errorf("%s: 期望错误含 %q，实际 %+v", c.name, c.wantErr, errs)
			}
			continue
		}
		if len(errs) != 0 || len(nodes) != 1 {
			t.Fatalf("%s: 期望 1 节点无错误，实际 nodes=%+v errs=%+v", c.name, nodes, errs)
		}
		c.check(t, nodes[0])
	}
}

func TestParseLinksMultiLine(t *testing.T) {
	text := `vless://u1@1.1.1.1:443?security=tls#a

ss://badline
hysteria2://pw@2.2.2.2:443#b`
	nodes, errs := ParseLinks(text)
	if len(nodes) != 2 || len(errs) != 1 {
		t.Fatalf("混合行应得 2 节点 1 错误: nodes=%d errs=%v", len(nodes), errs)
	}
	if errs[0].Line != 3 {
		t.Errorf("错误行号应为 3，got %d", errs[0].Line)
	}
	if nodes[0].Name != "a" || nodes[1].Name != "b" {
		t.Errorf("名称解析不对: %+v", nodes)
	}
}
