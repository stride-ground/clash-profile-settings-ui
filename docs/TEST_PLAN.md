# 测试计划（TEST_PLAN）

## 1. 单元测试（`go test ./...`，随实现补齐）

| 目标 | 用例 |
|---|---|
| `LoadDoc`/`Model` | 完整 profile 解析；缺 listeners/proxy-groups 键；空文件；带 BOM；非法 YAML 报错 |
| `Apply`+`Marshal` 往返 | 改 listener 后 rules/dns/注释保留；origIndex 复用同节点；新组含全部必填 key；use/url/interval 零值删 key |
| `parseFrag` | 行内 `{name:x,...}` 与多行块均可；非 mapping 报错；缺 name 由 Apply 报错 |
| `Validate` | 表驱动：0.0.0.0→error；重复端口→error；proxy 引用不存在→error；绑单节点→warn；url-test 单成员→info；空 listeners→warn；allow-lan→warn；mixed-port 冲突→error；组循环 A→B→A→error |
| `Doc.Save` | 生成 `.bak`；新文件直接写；输出无 BOM |
| `scan`（handler 层） | profiles 子目录命中；目录直含 yaml；递归/空目录返回空列表；mtime 排序取最新 |

## 2. 端到端（手动清单，验收门执行）

1. `clash-profile-ui.exe` 启动 → 自动扫描默认 CVR 目录 → 默认浏览器打开已加载最新 profile（傻瓜模式）
2. 文件栏改 Clash 位置为 profiles 子目录/任意 yaml 目录 → 扫描 → 列表出现 → 自动加载最新
3. 傻瓜模式：4 环境、环境2 选备用 → 生成并保存 → 文件含 4 listener + `环境2-自动` url-test 组 → 指纹浏览器可填 socks5://127.0.0.1:1200x
4. 切详细模式：新增 listener 名 prof-x、socks、13001、绑 `出口A` → 该行显示 socks5 地址可复制
5. 切引导模式：每个区块出现说明文字，内容与大白话一致
6. 编辑产生 error（listen 改 0.0.0.0）→ 校验区实时出现 error；改回消失
7. 保存 → 提示成功+备份路径+「去 CVR 切换 profile 生效」→ 文件更新、`.bak` 存在、注释保留
8. 新建空白配置 → 粘贴节点 YAML → 建 url-test 组 → 加 listener → 预览 YAML → 保存为 `test-out.yaml` → `mihomo -t -f test-out.yaml` 通过
9. 端口冲突：`12001` 已有 listener 再加一个 → error；组循环 A→B→A → error
10. 坏文件：手改坏 YAML → 加载报错含原因，内存态不变
11. 三模式互切：傻瓜生成后切详细可见新条目，详细改完切引导数据不丢

## 3. 安全/边界

- UI/API 仅 127.0.0.1 可达（`netstat` 抽查）
- 路径穿越：`/api/load` 传 `..\..\x.yaml` 行为符合「按用户显式路径读文件」预期，无目录枚举接口
- 大文件：本机 100+ 节点 profile（约数百 KB）加载/保存 <1s

## 4. 冒烟（每次交付前）

`gofmt -l .` 空 → `go vet ./...` 过 → `go test ./...` 过 → `go build` 出 exe → 启动开页 → 加载真实 profile → 保存还原
