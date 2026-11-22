# 开发（DEVELOPMENT）

## 环境

- Go 1.27.1+（windows/amd64 验证）；`GOPROXY` 国内建议 `goproxy.cn,direct`
- 无 Node/前端工具链；`index.html` 为纯静态原生 JS
- 可选验证工具：CVR 自带 `verge-mihomo.exe -t -f <文件>` 或任意 mihomo 二进制

## 命令（工作目录=仓库根）

| 用途 | 命令 | 状态 |
|---|---|---|
| 拉依赖 | `go mod tidy` | 已验证 |
| 格式化 | `gofmt -w .` / 检查 `gofmt -l .` | 已验证（输出为空） |
| 静态检查 | `go vet ./...` | 已验证通过 |
| 编译检查 | `go build ./...` | 进行中：缺 main（后续文件补齐后应通过） |
| 运行 | `go run .` 或 `.\clash-profile-ui.exe` | 待实现 |
| 构建 | `go build -o clash-profile-ui.exe .` | 待实现 |
| 测试 | `go test ./...` | 待实现（见 TEST_PLAN） |

## 目录约定

- 根目录仅放 Go 源、`index.html`、`go.mod/go.sum`、调研与 AGENTS；文档在 `docs/`，设计产物在 `design/`，变更在 `docs/changes/`
- 二进制、`*.bak` 不入库（见 .gitignore）

## 代码约定

- gofmt（tab 缩进）；导出标识符写 Go doc；错误用 `fmt.Errorf("中文上下文: %w", err)` 包装返回，不 panic（`enc()` 的内部不变式除外）
- YAML 操作统一走 `mihomo.go` 的 mapGet/mapSet/enc 工具；前端字段名 camelCase
- 日志只到 stderr，不落文件，不含敏感内容

## Git 约定

- 分支：`main` 为默认；短期分支 `feat/<kebab>`、`fix/<kebab>`、`chore/<kebab>`；单人可直接在 main 提交，但保持"一个可审阅目标一个提交"
- 提交格式：Conventional Commits 1.0.0，英文（`feat:`/`fix:`/`docs:`/`chore:`/`test:`）
- 提交前：`gofmt -l .` 为空、`go vet ./...` 通过、staged diff 只含本任务文件
- 不 push（未配置远端）；不提交 `.bak`/二进制/个人路径配置

## 变更控制

已批准的页面区块、API 契约、YAML 写回语义、部署形态作实质变更 → 先写 `docs/changes/CR-<id>.md`（原因/原新行为/影响/回滚/重审门/状态），批准后再改。
