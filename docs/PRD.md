# PRD — Clash Profile Settings UI

> 版本：v0.1 草案（待 Gate 1 批准）。需求映射见 §6。

## 1. 用户与场景

单人本机用户，在指纹浏览器（如 Ant Browser）中同时运行多个 Profile，要求"一端口一节点/策略组"多出口在线。工具替代手写 `listeners`/`proxy-groups` YAML 的过程，并保证写回 Clash Verge Rev profile 不破坏其余配置。

## 2. 范围

**做**
- Clash 位置扫描：用户指定目录（CVR 数据目录 / profiles 目录 / 任意含 yaml 目录）→ 列出其中 profile → 自动加载最新一个；启动时自动扫描默认位置 `%APPDATA%\io.github.clash-verge-rev.clash-verge-rev` 并自动加载
- 手填文件路径加载、新建空白配置
- **三种模式**（切换不清空已编辑内容，编辑同一份数据）：
  - **新手任务**（默认入口）：按真实任务流引导——任务①粘 `vless://`/`hy2://` 链接批量加节点（解析预览、逐行报错）＋节点删查；任务②建组（url-test/select/fallback，大白话解释差异＋成员勾选）＋组改删；任务③简化绑端口（端口+目标+复制 socks5 地址）
  - **详细模式**：listeners/proxy-groups/proxies/常规 全字段编辑
  - **详细+说明**：同详细模式界面，每个区块/字段附常驻大白话说明
- 可视化编辑 `listeners`（name/type/port/listen/proxy 绑定）
- 可视化编辑 `proxy-groups`（name/type/成员勾选+过滤/url/interval/tolerance，含 `use` provider 字段）
- 节点列表查看、过滤、删除；新增两条路径：粘贴 `vless://`/`hysteria2://`(`hy2://`) 分享链接（批量）或 mihomo YAML 片段
- 常规项：mixed-port、allow-lan、mode
- 实时校验（§5 规则：回环绑定、端口冲突、引用存在性、url-test 建议、组引用成环检测）
- 保存：写回原文件并生成 `.bak` 备份；保存后提示去 CVR 切换 profile
- 每个 listener 自动生成可复制 `socks5://127.0.0.1:<port>` 供指纹浏览器填写
- YAML 预览/复制/下载

**不做（v1 非目标）**
- 订阅 URL 抓取/更新；`ss://`/`trojan://`/`vmess://`/`tuic://` 链接解析（解析器留扩展位，后续 CR 加）
- rules/dns/tun 等其余 mihomo 配置编辑（原样保留）
- mihomo 内核运行控制、CVR 进程管理、RESTful API 热重载
- 多文件 diff/合并、远程部署

## 3. 页面结构（单页）

顶部：**模式切换器**（新手任务 / 详细模式 / 详细+说明）＋文件栏，三模式共用。

| 区块 | 内容 | 需求源 |
|---|---|---|
| 文件栏 | Clash 位置输入 + 扫描 + profile 下拉 + 路径输入 + 加载/保存/新建/预览 | 扫描功能（用户要求） |
| 校验区 | error/warn/info 列表，实时刷新，点击定位区块 | §5 约束 |
| 任务①加节点 | 链接粘贴框（一行一个）→ 解析预览表 → 加入列表；下方节点清单（删/查） | 用户反馈 CR-001 |
| 任务②建组 | 组名 + 类型单选（自动切换/手动/备用，附大白话）+ 成员勾选过滤 → 创建/更新；下方组清单（改/删） | 用户反馈 CR-001 |
| 任务③绑端口 | 简化 listener 表：端口 + 绑定目标下拉 + socks5 地址复制 | §5.2 |
| 出口端口 | listeners 表格：名称/类型(socks,mixed,http)/监听/端口/绑定目标(下拉候选=节点+组)/浏览器代理地址+复制 | §5.2 listeners |
| 策略组 | 组卡片：名称/类型(select,url-test,fallback,load-balance,relay)/成员勾选+过滤/url/interval/tolerance | §5.2 proxy-groups |
| 节点 | proxies 表（name/type/server/port）+ 过滤 + 删除 + YAML 片段粘贴新增 | §5.1 手动填节点 |
| 常规 | mixed-port / allow-lan / mode | §5.4 只留本地端口 |
| YAML 预览 | 生成结果只读视图 + 复制/下载 | 核对产出 |

新手任务显示：文件栏 + 校验 + 任务①②③；详细/详细+说明显示：文件栏 + 校验 + 全部编辑区块。

## 4. 状态要求

- 空：未加载文件 → 引导区（扫描结果/新建按钮），编辑区禁用
- 加载失败/保存失败：错误横幅显示原因（如 YAML 语法错误行号）
- 脏状态：编辑未保存 → 标题/按钮提示；离开无需拦截（本地工具）
- 校验：任何编辑后 ≤1s 内刷新；error 存在不阻止保存但显著提示
- 无数据：节点为空时策略组成员选择显示空态说明

## 5. 验收标准（验收门核对）

1. `clash-profile-ui.exe` 双击启动，浏览器自动打开 UI，全程无需命令行参数
2. 启动自动扫描默认 CVR 目录 → 自动加载最新 profile 显示 100+ 节点 / 多出口组 / 12001–12004 listeners（以本机实际为准）
3. 手填 Clash 位置（如 `…\clash-verge-rev` 或其 `profiles` 子目录）→ 扫描 → 列表出现 → 自动加载
4. 新手任务：粘贴一条 `vless://` 和一条 `hysteria2://` 链接 → 解析预览正确 → 加入列表 → 任务②建成 url-test 组 → 任务③绑到 12005 → 保存后文件含新节点+新组+新 listener
5. 详细模式新增 listener 绑定某 url-test 组 → 保存 → `.bak` 生成；「详细+说明」每个区块可见说明文字
6. 把某 listener 的 `listen` 改为 `0.0.0.0` → 校验区出现 error 提示
7. 复制按钮给出 `socks5://127.0.0.1:<port>`
8. 新建空白配置 → 加节点+组+listener → 保存为新文件 → `mihomo -t` 或通过 CVR 加载不报错（冒烟级）
9. 人为构造组循环 A→B→A → 校验区报 error

## 6. 需求映射

| Research §5 条目 | 功能 |
|---|---|
| §5.1 导入节点 | 节点列表 + YAML 片段新增（订阅由 CVR 负责） |
| §5.2 listeners 每端口绑定 | 出口端口编辑区 |
| §5.2 `listen: 127.0.0.1` 不要 0.0.0.0 | 校验 error + UI 默认 |
| §5.2 url-test 自动容灾 | 策略组编辑 + 默认类型建议 |
| §5.3 浏览器填 socks5://127.0.0.1:port | 每行代理地址+复制 |
| §5.4 关闭系统代理/TUN | 本工具不提供该开关；仅文档提示 |
| §5.5 proxy 可指节点或组 | 绑定目标下拉混合候选 |
| 附录:改后需切 profile 生效 | 保存成功提示文案 |
| 附录:UTF-8 无 BOM | 读取去 BOM、写无 BOM |
