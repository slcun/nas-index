# NAS Manager (Go 版本)

基于 Go 语言的 NAS 服务管理面板，提供 Web 界面管理 systemd 服务、实时终端等功能。

> 本项目代码由 AI 生成，仅供学习参考。
> **仅部署在 Linux 平台**（依赖 systemd），开发机为 Windows。

## 功能特性

- 服务管理：自动发现 systemd 服务，支持启动、停止、重启
- 实时状态：自动刷新服务运行状态
- 日志查看：在线查看服务日志
- Web 终端：提供基于 WebSocket 的终端访问
- 配置管理：支持 YAML 配置文件，热重载
- Demo 模式：在非 Linux 环境下自动启用（11 个模拟服务），方便开发调试
- 单文件部署：所有资源打包在一个可执行文件中

## 快速开始

### 编译（Linux 目标）

在开发机（Windows）上编译 Linux 二进制：

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -ldflags="-s -w" -o nas-manager .
```

在 Linux 目标机上直接编译：

```bash
go build -ldflags="-s -w" -o nas-manager .
```

> **注意**：`Makefile` 中所有 `build` 目标的路径 `./cmd/nas-manager` 已废弃，请直接使用上方命令。

### 运行

```bash
sudo ./nas-manager
```

默认访问地址：<http://localhost:5000>

### 命令行参数

```
./nas-manager [选项]

选项:
  -config string
        配置文件路径
  -port int
        HTTP 服务端口 (default 5000)
  -ws-port int
        WebSocket 服务端口 (default 5001)
  -install
        安装为系统服务（待实现）
  -uninstall
        卸载系统服务（待实现）
  -version
        显示版本信息
```

## 项目结构

```
nas-manager-go/
├── main.go                 # 主程序入口
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖锁定文件
├── Makefile                # 编译脚本
├── config.yaml             # 配置文件
├── internal/               # 内部包
│   ├── api/                # API 处理
│   ├── config/             # 配置管理
│   ├── host/               # 主机信息
│   ├── service/            # 服务管理
│   └── terminal/           # 终端处理
└── web/                    # Web 资源
    ├── static/             # 静态文件
    └── templates/          # HTML 模板
```

## 技术栈

- 后端：Go 1.21+
- Web框架：标准库 net/http
- 配置：YAML (gopkg.in/yaml.v3)
- WebSocket：github.com/gorilla/websocket
- 前端：原生 HTML/CSS/JavaScript + wterm

## API 接口

| 方法   | 路径                           | 说明        |
| ---- | ---------------------------- | --------- |
| GET  | /api/services                | 获取所有服务及分类 |
| GET  | /api/services/{name}         | 获取单个服务详情  |
| POST | /api/services/{name}/start   | 启动服务      |
| POST | /api/services/{name}/stop    | 停止服务      |
| POST | /api/services/{name}/restart | 重启服务      |
| GET  | /api/services/{name}/logs    | 获取服务日志    |
| GET  | /api/host/info               | 获取主机信息    |
| GET  | /api/config                  | 获取配置      |
| PUT  | /api/config                  | 更新配置      |

## 配置文件

编辑 `config.yaml` 来自定义服务列表：

```yaml
services:
  - name: jellyfin.service
    display_name: Jellyfin
    description: 媒体服务器
    port: 8096
    category: media
    web: true

exclude_services:
  - accounts-daemon.service

categories:
  media: 媒体中心
  download: 下载工具
```

## 部署到 Linux

```bash
# 在目标机上
sudo ./nas-manager
```

建议创建 systemd 服务实现自启动（需手动编写 service 单元文件）。`-install` / `-uninstall` 参数暂未实现。

## 未来计划

- [ ] 系统服务安装/卸载功能
- [ ] 性能监控面板

## 许可证

MIT
