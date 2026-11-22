# 项目发现（PROJECT_DISCOVERY）

> 生成时间：2026-11-22。性质标注：【事实】有证据；【推断】有直接证据但不完整；【建议】待批准决定；【待确认】需要用户裁决。

## 1. 目标与范围

- 【事实】需求来源：用户私有调研文档 `proxy-client-research.md` 第五节"本场景（指纹浏览器多 Profile）推荐配置"（含私人部署信息，不入库）。
- 【事实】目标产物：一个编译后的 Windows 二进制，启动后打开 HTML UI，可视化完成 §5 推荐配置：
  - mihomo `listeners`：每个本地端口绑定固定节点或策略组（一端口一出口）
  - `proxy-groups`：以 `url-test` 为主实现出口内自动容灾
  - 产出/写回 Clash Verge Rev 的 profile YAML
- 【建议】非目标：不实现代理内核本身；不替代 CVR 的订阅/规则管理；不做 TUN/系统代理开关；v1 不做分享 URI（vless:// 等）解析。

## 2. 现状盘点

- 【事实】仓库：`E:\github\clash-profile-settings-ui`，2026-11-22 初始化 git，默认分支 `main`。
- 【事实】已存在：`proxy-client-research.md`（调研+本机 CVR 部署运维记录，本地私有、不入库）；`go.mod`；`mihomo.go`（模型层先行草稿，本次通过 `go vet` 与编译检查，仅缺 `main`）。
- 【事实】环境：Go 1.27.1 windows/amd64；`GOPROXY=goproxy.cn,direct`。
- 【事实】本机部署背景（research 附录）：CVR 安装于 `C:\Program Files\Clash Verge\`，profile 目录 `%APPDATA%\io.github.clash-verge-rev.clash-verge-rev\profiles\`，现用端口 7890(混合)+12001–12004(多出口)。

## 3. 技术判断

- 【事实/建议】语言与形态：Go 1.27，单 exe（`windows/amd64`），`embed` 内嵌单页 UI，`net/http` 本地服务，监听 127.0.0.1，启动后调用默认浏览器。
- 【事实】依赖：`gopkg.in/yaml.v3`（唯一第三方依赖，已入 go.sum）。选择理由：需要保留 CVR profile 中本工具不管理的字段（rules/dns/订阅头）和注释，故以 `yaml.Node` 树保真读写，而非结构体全量重编码。
- 【事实】持久化：**无数据库**。被编辑的 mihomo YAML 即持久层；保存前写 `*.bak` 备份。
- 【建议】前端复杂度：**低**。单页表单/表格编辑，无路由、无实时协作、无多角色、无品牌视觉要求 → 建议保留低保真门、合并"高保真+Demo"为"可运行 Demo"一门（见 AGENTS 阶段表，待批准）。

## 4. 风险与边界

- 【事实】保存生效需用户在 CVR GUI 切换一次 profile（research 附录），UI 保存后必须提示。
- 【事实】写回必须无 BOM UTF-8（带 BOM 会致 CVR 解析失败）——已在读取侧去 BOM、写入侧不产出 BOM。
- 【推断】直接写 `profiles/*.yaml` 属于改 CVR 的本地型 profile，不会触发订阅更新覆盖；远程 profile 会被订阅刷新覆盖，UI 仅提示、不阻止。
- 【风险】listener `proxy` 绑定不存在的节点/组、端口重复、`listen` 非回环 → 已在 `Validate` 中列为 error 级。

## 5. 角色可用性

- 【事实】本会话运行时可调用：`subagent_general`、`subagent_explore`。agency-agents 角色库（Product Manager/UX Architect 等）未安装、不可调用。
- 【建议】本阶段（以及默认全程）不启用委派，由主会话通用代理直接实现——列为 Gate 1 待确认项。

## 6. 待确认项（已裁决）

Gate 1（2026-11-22 用户批准）：
1. ~~阶段配置~~ → 批准"低保真 → 可运行 Demo（合并高保真）→ 验收"精简门序
2. ~~委派模式~~ → 批准不装 agency-agents、全程主会话实现
3. ~~范围~~ → 批准分享链接导入列为 v1 非目标

Gate 2 反馈新增（同日）：
- 用户指定 Clash 位置扫描 + 自动加载 → 已纳入 PRD/线框
- 三模式（傻瓜/详细/引导）→ 已纳入 PRD/线框
- 组环检测：用户委托决定 → 决定实现（error 级）

Gate 2 二次修订（同日，CR-001 approved）：
- 傻瓜模式方向修正为「新手任务」：用户实际卡点在节点导入（vless/hy2 链接→Clash 节点）与组建（代理组/自动切换组）的 CRUD，端口绑定本身可理解 → 分享链接解析由 v1 非目标转核心功能（vless://、hysteria2://；其余类型留扩展）
