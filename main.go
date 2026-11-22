package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18630", "监听地址（端口占用时自动顺延最多 20 个）")
	file := flag.String("file", "", "启动即加载的 YAML 文件（优先于自动扫描）")
	noBrowser := flag.Bool("no-browser", false, "不自动打开浏览器")
	flag.Parse()

	srv := &Server{}

	// 启动预载：-file 优先；否则自动扫描默认 CVR 目录并加载最新 profile
	if *file != "" {
		if _, err := srv.loadPath(*file); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 加载 %s 失败: %v\n", *file, err)
		}
	} else {
		dir, files, err := scanClashDir("")
		if err == nil {
			srv.scanDir, srv.files = dir, files
			srv.autoload(files)
		}
	}

	ln, url := listenOrNext(*addr)
	if ln == nil {
		fmt.Fprintln(os.Stderr, "找不到可用端口（"+*addr+" 起 20 个均被占用）")
		os.Exit(1)
	}
	fmt.Println("Clash 出站配置器已启动:", url)
	if srv.path != "" {
		fmt.Println("已自动加载:", srv.path)
	}
	if !*noBrowser {
		openBrowser(url)
	}
	if err := http.Serve(ln, srv.mux()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listenOrNext(addr string) (net.Listener, string) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, ""
	}
	port, _ := strconv.Atoi(portStr)
	for i := 0; i < 20; i++ {
		p := port + i
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(p)))
		if err == nil {
			return ln, "http://" + net.JoinHostPort(host, strconv.Itoa(p))
		}
	}
	return nil, ""
}

func openBrowser(url string) {
	// start 需要空标题参数；失败不致命，URL 已打印
	_ = exec.Command("cmd", "/c", "start", "", url).Start()
}
