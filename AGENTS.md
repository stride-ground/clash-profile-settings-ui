# 项目操作规则

## 项目事实与文档入口

- 目标：编译产出 Windows 单 exe，启动后打开 HTML UI，可视化完成 `proxy-client-research.md` §5 的 mihomo 配置（listeners 一端口一出口 + url-test 策略组），写回 Clash Verge Rev profile。用户=单人本机；非目标与验收见 `docs/PRD.md`。
- 类型/语言：CLI 本地工具，Go 1.27（windows/amd64），标准库 + 唯一依赖 `gopkg.in/yaml.v3`；无数据库（YAML 文件即持久层，保存先写 `.bak`）。
- 前端复杂度：低（embed 单页原生 JS，无构建链）。
- 关键目录：`mihomo.go`=YAML 保真读写与 §5 校验；`server.go`=HTTP API；`main.go`=入口；`index.html`=UI；`docs/`=治理文档。
- 每次开始先读：本文件阶段表 + `docs/PRD.md`、`docs/ARCHITECTURE.md`。
- 发现证据与待确认：`docs/PROJECT_DISCOVERY.md`。

## 当前阶段与审核记录

| 字段 | 当前值 |
|---|---|
| 当前阶段/状态 | A 低保真线框 — **待审核（Gate 2）** |
| 本阶段允许修改 | `design/wireframes/`、`docs/WIREFRAMES.md`、本文件状态记录 |
| 审核产物与版本 | `design/wireframes/01–03` + `docs/WIREFRAMES.md` v2（含扫描+三模式）@ 见 git log |
| 最近明确批准 | 用户，2026-11-22，Gate 1：批准精简门序（合并高保真）、不委派主会话实现、分享链接导入列为 v1 非目标 |
| 未决问题/风险 | Gate 2 反馈已并入（扫描/三模式/环检测）待复批；写回保真性待真实 profile 验证 |
| 下一阶段及进入条件 | Gate 2 批准 → 实现+可运行 Demo（server.go / index.html / main.go） |

## 本项目阶段与交付（提案，待 Gate 1 确认）

| 阶段 | 产物与验证 | 人工门 |
|---|---|---|
| 发现/需求/架构 | 本批文档 + git 初始化 + 模型层草稿 | **Gate 1：范围、技术选型、阶段配置** |
| A 低保真 | `design/wireframes/` 可打开线框 + `docs/WIREFRAMES.md`（区块/交互/状态索引） | Gate 2：功能完整性与布局流程 |
| 实现+可运行 Demo（建议合并 B/C：内部工具无品牌视觉要求，UI 即单页表单） | `go build` 出 exe、启动开页、真实 profile 数据、正常/空/错状态 | Gate 3：产品形态确认 |
| 测试/验收 | 执行 `docs/TEST_PLAN.md` 并记录 | Gate 4：验收 |

未批准前不实现正式功能代码（`mihomo.go` 为发现期先行草稿，Gate 1 后可调整或废弃）。

## 角色与委派

| 阶段职责 | 请求角色 | 来源/版本 | 运行时可用 | 缺失处理 |
|---|---|---|---|---|
| 需求/架构/实现/测试 | agency-agents 角色库 | 未安装 | 不可调用（仅 `subagent_general`/`subagent_explore` 在运行时可调用） | 提案：全程不委派，主会话通用代理直接实现——待 Gate 1 确认 |

## 编码与验证

- 格式化 `gofmt`（tab）；lint=`go vet`；均无配置文件（语言 fallback，本项目首用）。
- 命名/错误处理/目录约定见 `docs/DEVELOPMENT.md`；YAML 操作走 `mihomo.go` 工具函数。
- 变更聚焦当前阶段；不夹带无关重构；不提交 `.bak`/exe/敏感路径。

| 用途 | 命令（仓库根） | 本轮验证状态 |
|---|---|---|
| 格式化检查 | `gofmt -l .` | 已验证（空输出） |
| 静态检查 | `go vet ./...` | 已验证通过 |
| 测试 | `go test ./...` | 未配置（实现阶段补齐） |
| 构建 | `go build -o clash-profile-ui.exe .` | 阻塞：缺 main（实现阶段） |
| 运行 | `.\clash-profile-ui.exe` | 阻塞：同上 |

## Git 与交付约定

- `main` 默认分支；单人可直接提交 main；短期分支 `feat/`、`fix/`、`chore/`；Conventional Commits 1.0.0 英文。
- 提交前：`gofmt -l .` 空 + `go vet` 过 + staged diff 只含本任务文件。
- 用户已授权本阶段 git init 与提交；push 未授权（也无远端）。

## 变更控制

对已批准的页面区块、API 契约、YAML 写回语义、部署形态作实质变更 → `docs/changes/CR-<id>.md`（原因/原新行为/影响/回滚/重审门/状态 proposed→approved/rejected）。创建 CR ≠ 批准。

## Definition of Done 与审核包

- 本阶段 DoD：文档齐全且互相一致、git 初始化完成、分批提交可审阅、命令状态如实记录。
- 每次交付输出一页 Review Packet：产物入口/验证与置信度/≤3 待决项/风险/当前门与下一步。Gate 处暂停，不自行进入下一阶段。
