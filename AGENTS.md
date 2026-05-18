# AGENTS.md

## Stack: Go 1.21+ 单二进制

**入口点:** `main.go` — 启动 HTTP (:5000) + WebSocket (:5001) 两个服务

**构建:**
```bash
go build -ldflags="-s -w" -o nas-manager .
```
`Makefile` 中 build 目标路径 `./cmd/nas-manager` 已损坏，请直接使用上方命令。

**运行:** `./nas-manager [-port 5000] [-ws-port 5001] [-config path]`

**依赖:** `github.com/gorilla/websocket v1.5.3`, `gopkg.in/yaml.v3 v3.0.1`

**无测试 / 无 linter / 无 CI / 无 typecheck / 无热重载**

## 架构

```
main.go -> config.Manager (YAML 热重载) -> service.Manager (systemd / demo) -> api.Handlers -> http.ServeMux
```

- 前端通过 `//go:embed web/static web/templates` 嵌入二进制
- **Demo 模式**自动激活（非 systemd 平台如 Windows），返回 11 个硬编码模拟服务（`internal/service/manager.go:133-256`）
- **WebSocket 终端为 demo/stub**：仅回显 + 打印 `"Command executed (demo mode)"`；`pty.go` 中 `Read()`/`Write()` 返回错误，无真实 PTY 连接
- **`-install` / `-uninstall` 未实现**（`main.go:129-134` 打印占位信息）

## API 路由 (`internal/api/router.go`)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/services` | 所有服务 + 分类 |
| GET | `/api/services/{name}` | 单个服务详情 |
| POST | `/api/services/{name}/start\|stop\|restart` | 管理服务 |
| GET | `/api/services/{name}/logs?lines=50` | `journalctl` 输出 |
| GET | `/api/host/info` | 主机信息 |
| GET/PUT | `/api/config` | 读/写配置 |

路由使用 Go 1.22+ `http.ServeMux` 模式匹配（`{name}` 语法）。

## Config (`config.yaml`)

- `services`: 列表 — 手动定义覆盖自动发现（`internal/service/model.go` 中 `mergeServiceConfig`）
- `exclude_services`: 列表 — **字面字符串匹配，非 glob**。`systemd-*.service` 不会展开通配符
- `categories`: 映射 — 分类显示名称

配置热重载：每次 `ListServices()` 调用检查文件 mtime，无需重启。

## 前端

- `internal/service/model.go` 中定义 `ServiceInfo` 结构体，包含 `Name`、`DisplayName`、`Description`、`Port`、`Category`、`ActiveState`、`UnitFileState`、`Web`、`Managed`
- `static/js/app.js` 每 15s 轮询 `/api/services`，搜索防抖 200ms
- WebSocket 终端页面使用本地 wterm 库（`web/static/libs/wterm/`），无 CDN 依赖
- 无用户认证 / 无 CSRF 保护（假设内网环境）

## 约定

- Go 源代码文件**无注释无文档字符串**；配置/脚本注释用中文
- `exclude_services` 条目使用字面字符串比较（参见 `internal/service/systemd.go`）
