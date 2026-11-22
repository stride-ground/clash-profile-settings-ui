# 使用手册 —— clash-profile-settings-ui

本文档逐一说明界面各区块、全部字段、命令行参数、文件行为与 API。

## 1. 启动与命令行

```
clash-profile-ui.exe [-addr 127.0.0.1:18630] [-file <yaml路径>] [-no-browser]
```

| 参数 | 默认 | 说明 |
|---|---|---|
| `-addr` | `127.0.0.1:18630` | 监听地址。仅接受 loopback；端口被占用时自动 +1 顺延，最多试 20 个，启动后在控制台打印实际地址 |
| `-file` | （空） | 启动即加载指定 YAML，跳过自动扫描 |
| `-no-browser` | false | 不自动打开浏览器（URL 仍打印在控制台） |

启动预载顺序：`-file` > 自动扫描默认 CVR 目录并加载「最新且含实质配置」的 YAML（按修改时间倒序，跳过无 proxies/groups/listeners 的覆写模板如 `Merge.yaml`）。

## 2. 文件栏（页面顶部）

| 控件 | 说明 |
|---|---|
| Clash 位置 + 扫描 | 填 CVR 数据目录或 profiles 目录 → 扫描其中 `*.yaml`，生成候选列表 |
| 文件下拉框 + 加载 | 从扫描结果手动选文件加载 |
| 新建 | 创建空配置（含空 proxies/groups/listeners 骨架） |
| 配置文件路径 | 显示/修改当前文档的保存目标路径；保存可写到任意路径（用于先存副本试效果） |
| 未保存标记 | 模型与文件不一致时显示 ● |

## 3. 校验区

配置每次变化后约 0.5s 自动校验（也可随时手动触发）。等级：

- **error（红）**：必须修复才能正常工作的配置问题 —— 引用不存在的节点/组、端口重复、端口缺绑定、组循环（A→B→A）、listener 字段缺失
- **warn（黄）**：建议核对 —— 组无成员、vless+Reality 节点缺 `flow` 且未开 `support-x25519mlkem768`、绑定目标疑似重名等

点击校验条目自动滚动定位到对应区块并高亮。

## 4. 模式一：新手任务

面向「有链接但不会配 Clash」的用户，三个任务按序完成。

### 任务① 加节点

- 输入框智能识别：每行含 `://` → 按分享链接解析；否则整段按 YAML 解析
- 支持：`vless://`、`hysteria2://`、`hy2://`；YAML 支持裸映射、`- name:` 单项、多节点序列
- 逐行/逐项解析，失败项单独报错不阻塞成功项
- 待加入表可改名；「加入节点列表」时同名节点询问「覆盖 / 自动改名 `-2`」
- **已有节点表**：名称/类型/服务器/端口 + 「详」（完整 YAML 弹窗+复制）+「✕」删除；顶部过滤框按名称/类型/服务器筛选

### 任务② 组「出口」

| 字段 | 说明 |
|---|---|
| 组名 | 组在配置里的唯一标识，端口绑它 |
| 类型 | url-test（自动测速最快+容灾）/ select（CVR 手动选）/ fallback（顺序用第一个活的）/ load-balance / relay |
| url / interval | 自动类组的测速目标与周期（秒），默认 `generate_204` + 300s |
| 成员 | 复选框勾选节点（可含其他组），顶部过滤框即时筛选；勾选状态与组名输入解耦，先勾后改名不丢 |
| 组列表 | 已有组可「载入」回表单编辑、「删除」；显示各组被哪些端口引用 |

### 任务③ 绑端口

| 列 | 说明 |
|---|---|
| 端口 | 本机代理端口（建议 12001 起；listener 固定监听 127.0.0.1） |
| 出口绑到 | 下拉选节点或组（节点/组分两组列出） |
| 浏览器填这个 | 自动生成 `socks5://127.0.0.1:<端口>`，点「复制」 |

## 5. 模式二：详细模式

完整编辑全部受管区块：

- **listeners**：名称、类型（`mixed`/`socks`/`http`/`redir`/`tproxy`/`tunnel`）、监听地址、端口、绑定目标（datalist 提示全部节点+组名）、自动生成的浏览器地址
- **proxy-groups**：全字段表格 —— 名称、类型、成员（勾选+过滤）、use、url、interval、其他策略字段原样保留
- **proxies**：只读展示名称/类型/服务器/端口（完整字段经「详」查看，原始节点字段不参与编辑以保真）；支持过滤、删除、YAML 片段新增
- **常规**：mixed-port、allow-lan、mode（rule/global/direct）

## 6. 模式三：详细 + 说明

与详细模式相同的编辑能力，每个区块上方多一段大白话说明（字段含义、推荐值、常见坑），适合边学边配。

## 7. YAML 预览与保存

- **YAML 预览**：点「刷新」把当前模型渲染成完整 YAML 预览；可复制、可下载 `.yaml`
- **保存**：校验 error 会阻止保存提醒；写文件前先把原文件备份为 `<文件>.bak`；输出 UTF-8 无 BOM、2 空格缩进
- **写回保真**：未接管的顶层字段（rules、dns、external-controller、订阅 providers、注释等）逐字保留；节点原始字段顺序/引号风格保留；新增节点按字母序排键
- 保存后提示去 CVR 切换 profile 使运行时生效

## 8. HTTP API（本机 127.0.0.1）

| 端点 | 方法 | 说明 |
|---|---|---|
| `/api/state` | GET | 当前 path/scanDir/文件列表/model |
| `/api/scan` | POST `{dir}` | 扫描目录，自动加载首个有效 profile |
| `/api/load` | POST `{path}` | 加载指定文件 |
| `/api/new` | POST | 新建空文档 |
| `/api/validate` | POST `{model}` | 校验，返回 `issues[]`（level/where/message） |
| `/api/preview` | POST `{model}` | 渲染完整 YAML 文本 |
| `/api/save` | POST `{path?, model}` | 校验+应用+备份+写盘 |
| `/api/parse-links` | POST `{text}` | 逐行解析分享链接 → `proxies[]` + `errors[]` |
| `/api/parse-yaml` | POST `{raw}` | 解析 YAML 节点块 → `nodes[]`（含完整 node 字段） |
| `/api/node-detail` | POST `<ProxyEntry>` | 返回节点完整 YAML 文本（详情弹窗用） |

## 9. 字段映射：vless:// 链接 → mihomo 节点

| 链接参数 | 写入字段 |
|---|---|
| `uuid@server:port` | `uuid` / `server` / `port` |
| `#name` | `name` |
| `security=reality` | `tls: true` + `reality-opts` |
| `sni`（或 `peer`） | `servername` |
| `pbk` | `reality-opts.public-key` |
| `sid` | `reality-opts.short-id` |
| `spx` | `reality-opts.spider-x` |
| `support-x25519mlkem768=true` | `reality-opts.support-x25519mlkem768: true` |
| `fp` | `client-fingerprint`（缺省 chrome） |
| `flow` | `flow`（链接没有则不写） |
| `type=ws/grpc` | `network` + `ws-opts`/`grpc-opts` |
| `type=xhttp` | 报「已预留槽位，本版未实现」 |

hysteria2：`password`、`sni`、`insecure→skip-cert-verify`、`obfs`/`obfs-password`、`mport→ports`、`alpn` 均映射。

## 10. 安全与边界

- 仅监听 `127.0.0.1`：局域网无法访问本工具页面与 API；无外部网络请求
- 保存自动 `.bak` 备份；写坏可直接用备份恢复
- 工具不读不存任何凭据到自身文件 —— 节点密钥只存在于你加载的 YAML 中
- listener 监听地址建议保持 `127.0.0.1`；设 `0.0.0.0` 会把无认证代理暴露给局域网
- 订阅型 profile 会被 CVR 刷新覆盖，自定义节点建议放本地 profile 或 CVR Merge 覆写
