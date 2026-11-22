# 架构（ARCHITECTURE）

> 版本：v0.1 草案。与 `mihomo.go` 已实现部分一致；`server.go`/`main.go`/`index.html` 为设计目标。

## 1. 模块边界

```
clash-profile-settings-ui/
├── main.go        入口：flags、net.Listen、打开浏览器、服务启动
├── server.go      HTTP 路由 + handler + 会话态（当前 *Doc）
├── mihomo.go      Doc/Model：YAML 保真读写、Apply、Validate（已实现）
├── shareparse.go  vless://、hysteria2://(hy2://) 分享链接 → mihomo proxy 映射（CR-001）
├── index.html     embed 单页 UI（原生 JS，无构建步骤）
└── go.mod         唯一依赖 gopkg.in/yaml.v3
```

无前端构建链、无数据库、无后台任务。单进程、单用户、串行请求，内存态只保留当前打开的一份 `*Doc`。

## 2. 数据流

```
YAML 文件 ──LoadDoc──> yaml.Node 树（完整保留）
                          │ Model()          ┌──────────────┐
                          ▼                  │  index.html  │
                 JSON Model（带 origIndex）──>│  用户编辑     │
                          ▲                  └──────────────┘
                          │ POST /api/*
                  Apply()：origIndex 复用原节点；
                  组/监听器 clone 后 patch 已知 key，
                  未知 key 原样保留 → Marshal → .bak + 写回
```

- `origIndex` 是前端条目指向原文件序列位置的引用；新增条目不带它，节点用 `raw`（YAML 片段）表达。
- proxies 整体重建序列但复用原节点 → 删除/重排无损；groups/listeners 用 clone+patch → 未知字段（`filter`、`rule`、`icon`…）与注释保留。
- 顶层未管理 key（`rules`、`dns`、`tun`、订阅头等）不动。

## 3. HTTP API（全部 JSON，仅绑 127.0.0.1）

| 方法/路径 | 入参 | 返回 | 说明 |
|---|---|---|---|
| GET `/` | — | index.html | embed 静态页 |
| POST `/api/scan` | `{dir}` | `{dir', files:[{name,path,mtime}], model?}` | 目录扫描：先试 `<dir>\profiles\*.yaml`，再试 `<dir>\*.yaml/.yml`；`dir` 为空用默认 CVR 目录；autoload=找到时返回最新一个的 model |
| POST `/api/load` | `{path}` | `{model, exists}` | 读文件→Model；解析失败 400+原因 |
| POST `/api/new` | — | `{model}` | 空白配置模板（不绑定路径） |
| POST `/api/parse-links` | `{text}` | `{proxies:[{name,type,server,port,raw}], errors:[{line,message}]}` | 分享链接逐行解析；`raw`=该节点 mihomo YAML，保存时按 `ProxyEntry.Raw` 入列 |
| POST `/api/validate` | `{model}` | `{issues:[{level,where,message}]}` | §5 规则校验 + 组环检测 + 端口占用探测 |
| POST `/api/preview` | `{model}` | `{yaml}` | Apply→Marshal 不落盘 |
| POST `/api/save` | `{path, model}` | `{ok, backup}` | Apply→备份→写回；失败 400 |
| GET `/api/state` | — | `{path, model}` | 恢复当前会话（刷新页面用） |

启动流程：`main` 先对默认 CVR 目录跑一次 `/api/scan` 同逻辑 → 有结果则预载最新 profile 到会话态 → UI 打开即已加载（空目录则空态）。

错误一律 `{error: "可读中文原因"}` + 4xx/5xx。

## 4. 关键设计决定

- **保真编辑**：选 yaml.v3 Node 树而非 struct 往返——CVR profile 含大量本工具不管理的字段，struct 重编码会丢字段、注释与顺序。代价：代码以 mapGet/mapSet 手动操作 Node。
- **写回策略**：先 `path+".bak"` 复制原文件再覆盖写；无 BOM UTF-8；`yaml.Encoder` 缩进 2。
- **校验在服务端**：`Validate` 是唯一规则源，UI 只展示——保证保存/预览/前端提示三者一致。含组引用成环检测（DFS，error 级）。
- **端口占用探测**：`net.DialTimeout(127.0.0.1:port, 300ms)`，仅 info 级（本机跑着 mihomo 属正常）。
- **新手任务在客户端组装**：任务①调 `/api/parse-links` 把链接转 YAML 片段挂进 Model；任务②③直接改 Model 的 groups/listeners——三模式编辑同一份数据，统一走 validate→save。
- **链接解析在服务端**：`shareparse.go` 用 `net/url` 解析；vless 取 uuid/host/port + query（security=reality→reality-opts{sni→servername, pbk→public-key, sid→short-id}、fp→client-fingerprint、flow、type=ws/grpc→传输层字段）；hy2 取 password + query（sni、insecure→skip-cert-verify、obfs/obfs-password）。解析产物先编码为 `raw` YAML 入 Model，写回走统一 Apply。
- **flags**：`-addr`（默认 `127.0.0.1:18630`，占用时顺延最多 20 端口）、`-file`（启动即加载指定文件，优先于自动扫描）、`-no-browser`。

## 5. 安全

- 只监听回环；无鉴权（本地单用户前提）；UI 侧默认且校验强制 listener `listen=127.0.0.1`。
- 不读取/输出文件内容以外的信息；不写日志到文件；路径由用户显式给出或来自固定 CVR 目录扫描，不做任意目录遍历接口。
- 依赖最小化：仅 yaml.v3（供应链考虑，不引前端框架/网络库）。

## 6. 部署与恢复

- 构建：`go build -o clash-profile-ui.exe .`，单文件分发，无运行时依赖。
- 恢复：`.bak` 即回滚副本；CVR 中切换 profile 重新生成运行时配置。
- 失败边界：YAML 解析失败不修改内存态；写盘失败时 `.bak` 已存在可直接还原。
