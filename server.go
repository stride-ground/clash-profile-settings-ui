package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed index.html
var indexHTML []byte

// FileInfo 扫描到的一份 profile 文件。
type FileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Mtime string `json:"mtime"`
	ts    int64
}

// Server 持有单用户会话态：当前打开的 Doc 与其路径、最近一次扫描结果。
type Server struct {
	doc     *Doc
	path    string
	scanDir string
	files   []FileInfo
}

func defaultClashDir() string {
	return filepath.Join(os.Getenv("APPDATA"), "io.github.clash-verge-rev.clash-verge-rev")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSONStatus(w, code, map[string]any{"error": err.Error()})
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("请求体 JSON 解析失败: %w", err)
	}
	return nil
}

func portBusy(port int) bool {
	c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)), 300*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// scanDir 解析用户给的 Clash 位置：先试 <dir>\profiles，再试 <dir> 本身，
// 收集 *.yaml/*.yml 按修改时间倒序。dir 为空用默认 CVR 目录。
func scanClashDir(dir string) (string, []FileInfo, error) {
	if dir == "" {
		dir = defaultClashDir()
	}
	dir = os.ExpandEnv(strings.TrimSpace(dir))
	var files []FileInfo
	for _, cand := range []string{filepath.Join(dir, "profiles"), dir} {
		ents, err := os.ReadDir(cand)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, FileInfo{
				Name:  e.Name(),
				Path:  filepath.Join(cand, e.Name()),
				Mtime: info.ModTime().Format("2006-01-02 15:04"),
				ts:    info.ModTime().Unix(),
			})
		}
		if len(files) > 0 {
			break // profiles 命中就不再扫目录本身
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ts > files[j].ts })
	return dir, files, nil
}

func (s *Server) loadPath(path string) (*Model, error) {
	d, err := LoadDoc(path)
	if err != nil {
		return nil, err
	}
	s.doc, s.path = d, path
	return d.Model(), nil
}

// looksLikeMihomo：含 proxies/proxy-groups/listeners 任一字段才视为节点 profile
// （排除 CVR 的 Merge.yaml 覆写模板之类文件）
func looksLikeMihomo(d *Doc) bool {
	for _, k := range []string{"proxies", "proxy-groups", "listeners"} {
		if mapGet(d.mapping, k) != nil {
			return true
		}
	}
	return false
}

// autoload 按 files 顺序（已 mtime 倒序）找第一份像节点 profile 的文件加载；
// 全部不合格则返回 nil 不改动当前会话态。
func (s *Server) autoload(files []FileInfo) *Model {
	for _, f := range files {
		d, err := LoadDoc(f.Path)
		if err != nil || !looksLikeMihomo(d) {
			continue
		}
		s.doc, s.path = d, f.Path
		return d.Model()
	}
	return nil
}

func (s *Server) mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/state", s.hState)
	mux.HandleFunc("POST /api/scan", s.hScan)
	mux.HandleFunc("POST /api/load", s.hLoad)
	mux.HandleFunc("POST /api/new", s.hNew)
	mux.HandleFunc("POST /api/validate", s.hValidate)
	mux.HandleFunc("POST /api/preview", s.hPreview)
	mux.HandleFunc("POST /api/save", s.hSave)
	mux.HandleFunc("POST /api/parse-links", s.hParseLinks)
	mux.HandleFunc("POST /api/parse-yaml", s.hParseYAML)
	return mux
}

func (s *Server) hState(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"path": s.path, "scanDir": s.scanDir, "files": s.files}
	if s.doc != nil {
		out["model"] = s.doc.Model()
	}
	writeJSON(w, out)
}

func (s *Server) hScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	dir, files, err := scanClashDir(req.Dir)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	s.scanDir, s.files = dir, files
	out := map[string]any{"dir": dir, "files": files}
	if m := s.autoload(files); m != nil {
		out["model"] = m
		out["path"] = s.path
	}
	writeJSON(w, out)
}

func (s *Server) hLoad(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	if req.Path == "" {
		writeErr(w, 400, fmt.Errorf("path 为空"))
		return
	}
	m, err := s.loadPath(req.Path)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	writeJSON(w, map[string]any{"model": m, "path": s.path})
}

func (s *Server) hNew(w http.ResponseWriter, r *http.Request) {
	s.doc, s.path = NewDoc(), ""
	writeJSON(w, map[string]any{"model": s.doc.Model()})
}

func (s *Server) hValidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model Model `json:"model"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	issues := Validate(&req.Model, portBusy)
	if issues == nil {
		issues = []Issue{}
	}
	writeJSON(w, map[string]any{"issues": issues})
}

func (s *Server) docOrErr() (*Doc, error) {
	if s.doc == nil {
		return nil, fmt.Errorf("尚未加载配置，请先加载文件或新建空白")
	}
	return s.doc, nil
}

func (s *Server) hPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model Model `json:"model"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	d, err := s.docOrErr()
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	dup := d.Clone()
	if err := dup.Apply(&req.Model); err != nil {
		writeErr(w, 400, err)
		return
	}
	y, err := dup.Marshal()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	writeJSON(w, map[string]any{"yaml": string(y)})
}

func (s *Server) hSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string `json:"path"`
		Model Model  `json:"model"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	d, err := s.docOrErr()
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	path := req.Path
	if path == "" {
		path = s.path
	}
	if path == "" {
		writeErr(w, 400, fmt.Errorf("未指定保存路径：请在「配置文件」框填目标路径"))
		return
	}
	if err := d.Apply(&req.Model); err != nil {
		writeErr(w, 400, err)
		return
	}
	d.path = path
	if err := d.Save(); err != nil {
		writeErr(w, 500, fmt.Errorf("写入失败: %w", err))
		return
	}
	s.path = path
	writeJSON(w, map[string]any{
		"ok": true, "path": path, "backup": path + ".bak",
		"model": d.Model(),
	})
}

func (s *Server) hParseLinks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	nodes, errs := ParseLinks(req.Text)
	writeJSON(w, map[string]any{"proxies": nodes, "errors": errs})
}

func (s *Server) hParseYAML(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Raw string `json:"raw"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, err)
		return
	}
	frags, err := parseFrags(req.Raw)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	nodes := []any{}
	for i, f := range frags {
		if getStr(f, "name") == "" {
			writeErr(w, 400, fmt.Errorf("第 %d 个节点缺少 name 字段", i+1))
			return
		}
		var obj map[string]any
		if err := f.Decode(&obj); err != nil {
			writeErr(w, 400, fmt.Errorf("第 %d 个节点解码失败: %w", i+1, err))
			return
		}
		nodes = append(nodes, map[string]any{
			"name": getStr(f, "name"), "type": getStr(f, "type"),
			"server": getStr(f, "server"), "port": getInt(f, "port"),
			"node": obj,
		})
	}
	writeJSON(w, map[string]any{"nodes": nodes})
}
