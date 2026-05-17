# NAS Index — NAS 服务导航与管理面板

> 🤖 **本项目完全由 AI 编写**，从架构设计到代码实现、从样式到部署脚本，全部由 AI 生成，未经过人工修改。

## 项目简介

NAS Index 是一套为家用 NAS 服务器设计的 Web 工具集，包含两个独立模块：

| 模块 | 说明 |
|------|------|
| **NAS 导航页** (`index.html`) | 静态服务导航页面，一键跳转 NAS 上运行的各种 Web 服务 |
| **NAS 管理面板** (`nas-manager/`) | Flask Web 应用，可视化管理 NAS 上的 systemd 服务（启停、状态监控、日志查看） |

## 功能特性

### NAS 导航页

- 按分类展示所有 NAS 服务入口（媒体中心、文件管理、下载工具等）
- 自动根据当前访问 IP 拼接各服务 URL
- 响应式布局，适配手机与桌面
- 纯静态页面，无需后端

### NAS 管理面板

- **服务发现**：自动扫描 `systemctl list-unit-files` 及 `/etc/systemd/system/` 目录
- **服务管理**：一键启动 / 停止 / 重启 systemd 服务
- **实时状态**：15 秒自动刷新服务运行状态
- **日志查看**：在线查看 journalctl 服务日志
- **配置热重载**：修改 `config.yaml` 后无需重启，自动检测变更
- **Demo 模式**：在非 systemd 环境（Windows、容器）下自动切换为模拟数据，方便开发调试
- **主机信息**：显示当前主机名与 IP 地址

## 技术栈

- **后端**：Python 3 + Flask
- **前端**：原生 HTML / CSS / JavaScript（无构建工具）
- **配置**：YAML
- **部署**：systemd + 一键安装脚本

## 项目结构

```
nas-index/
├── index.html                  # NAS 导航页（静态）
├── nas-manager/                # NAS 管理面板
│   ├── app.py                  # Flask 入口，定义所有 API 路由
│   ├── service_manager.py      # 服务发现与管理逻辑
│   ├── config_manager.py       # YAML 配置读取与热重载
│   ├── config.yaml             # 服务配置文件
│   ├── requirements.txt        # Python 依赖
│   ├── install.sh              # 一键安装脚本（Debian 13）
│   ├── templates/
│   │   └── index.html          # 管理面板页面模板
│   └── static/
│       ├── css/style.css       # 样式
│       └── js/app.js           # 前端交互逻辑
└── AGENTS.md                   # AI 开发指引
```

## 快速开始

### 开发环境

```bash
cd nas-manager
pip install -r requirements.txt
python app.py
```

启动后访问 http://localhost:5000

### 生产部署（Debian 13）

```bash
sudo bash nas-manager/install.sh
```

安装脚本会自动完成：
1. 安装 Python 依赖
2. 将应用部署到 `/opt/nasmanager/`
3. 创建 `nasmanager` 系统用户
4. 配置 sudoers 权限（仅允许 systemctl 操作）
5. 注册并启动 systemd 服务

部署后管理命令：

```bash
sudo systemctl status nasmanager     # 查看状态
sudo systemctl restart nasmanager    # 重启
sudo systemctl stop nasmanager       # 停止
sudo journalctl -u nasmanager -f     # 查看日志
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/services` | 获取所有服务及分类 |
| GET | `/api/services/<name>` | 获取单个服务详情 |
| POST | `/api/services/<name>/start` | 启动服务 |
| POST | `/api/services/<name>/stop` | 停止服务 |
| POST | `/api/services/<name>/restart` | 重启服务 |
| GET | `/api/services/<name>/logs?lines=50` | 获取服务日志 |
| GET | `/api/host/info` | 获取主机名与 IP |

## 配置说明

编辑 `config.yaml` 自定义服务列表：

```yaml
services:
  - name: jellyfin.service
    display_name: Jellyfin
    description: 媒体服务器
    port: 8096
    category: media
    web: true

exclude_services:
  - systemd-*.service

categories:
  media: 媒体中心
  download: 下载工具
```

- `services`：手动定义的服务（优先级高于自动发现）
- `exclude_services`：从面板中隐藏的服务名
- `categories`：分类显示名称

> ⚠️ `exclude_services` 中的条目是**精确字符串匹配**，不支持通配符。如需模糊匹配，需修改代码使用 `fnmatch`。

## 许可证

MIT
