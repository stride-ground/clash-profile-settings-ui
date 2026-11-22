# PRD — Clash Profile Settings UI

> 版本：v0.1 草案（待 Gate 1 批准）。需求映射见 §6。

## 1. 用户与场景

单人本机用户，在指纹浏览器（如 Ant Browser）中同时运行多个 Profile，要求"一端口一节点/策略组"多出口在线。工具替代手写 `listeners`/`proxy-groups` YAML 的过程，并保证写回 Clash Verge Rev profile 不破坏其余配置。

## 2. 范围

**做**
- 打开/自动扫描 CVR profile（`%APPDATA%\io.github.clash-verge-rev.clash-verge-rev\profiles\*.yaml`），也支持手填路径与新建空白配置
- 可视化编辑 `listeners`（name/type/port/listen/proxy 绑定）
- 可视化编辑 `proxy-groups`（name/type/成员/url/interval/tolerance，含 `use` provider 字段）
- 节点列表查看、删除、粘贴 mihomo YAML 片段新增
- 常规项：mixed-port、allow-lan、mode
- 实时校验（§5 规则：回环绑定、端口冲突、引用存在性、url-test 建议）
- 保存：写回原文件并生成 `.bak` 备份；保存后提示去 CVR 切换 profile
- 每个 listener 自动生成可复制 `socks5://127.0.0.1:<port>` 供指纹浏览器填写
- YAML 预览/复制/下载

**不做（v1 非目标）**
- 分享链接（vless://、hy2:// 等）解析导入；订阅抓取/更新
- rules/dns/tun 等其余 mihomo 配置编辑（原样保留）
- mihomo 内核运行控制、CVR 进程管理、RESTful API 热重载
- 多文件 diff/合并、远程部署

## 3. 页面结构（单页）

| 区块 | 内容 | 需求源 |
|---|---|---|
| 文件栏 | profile 下拉（扫描结果）+ 路径输入 + 加载/保存/新建 | §5.1 导入后改配置 |
| 校验区 | error/warn/info 列表，实时刷新 | §5 约束 |
| 出口端口 | listeners 表格：名称/类型(socks,mixed,http)/监听/端口/绑定目标(下拉候选=节点+组)/浏览器代理地址+复制 | §5.2 listeners |
| 策略组 | 组卡片：名称/类型(select,url-test,fallback,load-balance,relay)/成员勾选/url/interval/tolerance | §5.2 proxy-groups |
| 节点 | proxies 表（name/type/server/port）+ 删除 + YAML 片段粘贴新增 | §5.1 手动填节点 |
| 常规 | mixed-port / allow-lan / mode | §5.4 只留本地端口 |
| YAML 预览 | 生成结果只读视图 + 复制/下载 | 核对产出 |

## 4. 状态要求

- 空：未加载文件 → 引导区（扫描结果/新建按钮），编辑区禁用
- 加载失败/保存失败：错误横幅显示原因（如 YAML 语法错误行号）
- 脏状态：编辑未保存 → 标题/按钮提示；离开无需拦截（本地工具）
- 校验：任何编辑后 ≤1s 内刷新；error 存在不阻止保存但显著提示
- 无数据：节点为空时策略组成员选择显示空态说明

## 5. 验收标准（验收门核对）

1. `clash-profile-ui.exe` 双击启动，浏览器自动打开 UI，全程无需命令行参数
2. 扫描到本机 CVR profile 并成功加载显示 100+ 节点 / 多出口组 / 12001–12004 listeners（以本机实际为准）
3. 新增一个 listener 绑定某 url-test 组 → 保存 → 原文件更新且生成 `.bak`，其余字段（rules、节点完整字段、注释）不丢失
4. 把某 listener 的 `listen` 改为 `0.0.0.0` → 校验区出现 error 提示
5. 复制按钮给出 `socks5://127.0.0.1:<port>`
6. 新建空白配置 → 加节点+组+listener → 保存为新文件 → `mihomo -t` 或通过 CVR 加载不报错（冒烟级）

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
