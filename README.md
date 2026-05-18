# NAS Manager (Go 版本)

基于 Go 语言重写的 NAS 服务管理面板，提供 Web 界面管理 systemd 服务、实时终端等功能。

> 本项目代码由 AI 生成，仅供学习参考。

## 功能特性

- 服务管理：自动发现 systemd 服务，支持启动、停止、重启
- 实时状态：自动刷新服务运行状态
- 日志查看：在线查看服务日志
- Web 终端：提供基于 WebSocket 的终端访问
- 配置管理：支持 YAML 配置文件，热重载
- Demo 模式：在非 systemd 环境下也能运行
- 单文件部署：所有资源打包在一个可执行文件中

## 快速开始

### 编译

```bash
cd nas-manager-go
go build -o nas-manager .
```

或者使用 Makefile：

```bash
make build
```

### 运行

```bash
./nas-manager
```

默认访问地址：http://localhost:5000

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

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/services | 获取所有服务及分类 |
| GET | /api/services/{name} | 获取单个服务详情 |
| POST | /api/services/{name}/start | 启动服务 |
| POST | /api/services/{name}/stop | 停止服务 |
| POST | /api/services/{name}/restart | 重启服务 |
| GET | /api/services/{name}/logs | 获取服务日志 |
| GET | /api/host/info | 获取主机信息 |
| GET | /api/config | 获取配置 |
| PUT | /api/config | 更新配置 |

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

## 跨平台编译

使用 Makefile 进行跨平台编译：

```bash
# Linux
make build-linux

# macOS
make build-darwin

# Windows
make build-windows

# 所有平台
make build-all
```

## 未来计划

- [ ] 完整的 PTY 终端支持
- [ ] 系统服务安装/卸载功能
- [ ] 用户认证
- [ ] 服务分组和标签
- [ ] 性能监控面板

## 许可证

MIT

## 致谢

基于原 Python 版本 NAS Manager 重构，参考了 AdGuard Home 的架构设计。
