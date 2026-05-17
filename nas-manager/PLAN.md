# wterm Web 终端集成计划

## 架构概览

```
Browser                          Debian 13
┌─────────────────────┐         ┌──────────────────────────────┐
│  Flask Page :5000   │         │  Flask (app.py)              │
│  /terminal          │  HTTP   │  ├─ /                        │
│  WTerm + importmap  │◄────────│  ├─ /terminal  (新页面)       │
│                     │         │  ├─ /api/services            │
│  WebSocket connect  │         │  ├─ /api/host/info           │
│  ────────────────►  │────────►│  └─ 端口 5000                │
│                     │  WS     │                              │
│                     │◄────────│  WS Server (ws_server.py)    │
│                     │         │  ├─ websockets 库             │
│                     │         │  ├─ ptyprocess (新依赖)       │
│                     │         │  ├─ 每连接 spawn /bin/bash    │
│                     │         │  ├─ 处理 resize 信令          │
│                     │         │  └─ 端口 5001                 │
└─────────────────────┘         └──────────────────────────────┘
```

两个独立服务：Flask (HTTP, :5000) + WebSocket Server (WS, :5001)

---

## 文件变更清单

### 新增文件

| 文件 | 用途 |
|------|------|
| `ws_server.py` | WebSocket → PTY 中继服务（asyncio） |
| `templates/terminal.html` | 终端页面模板 |
| `static/js/terminal.js` | WTerm 初始化 + WebSocket 连接逻辑 |
| `static/css/terminal.css` | 终端全屏/布局样式 |

### 修改文件

| 文件 | 改动 |
|------|------|
| `requirements.txt` | 追加 `websockets`、`ptyprocess` |
| `install.sh` | 新增 ws_server systemd service 注册；install.sh 中下载 wterm WASM |
| `templates/index.html` | 在"系统工具"分类中添加"Web 终端"卡片 |
| `app.py` | 启动时可选启动 ws_server 子进程，或在 install.sh 中作为独立服务 |
| `config.yaml` | 可在系统工具分类中增加 terminal 服务定义 |
| `AGENTS.md` | 更新架构信息 |

---

## 详细设计

### 1. ws_server.py — WebSocket PTY Server

**依赖：** `websockets` + `ptyprocess`

**核心逻辑：**
```
websocket 连接到达
  │
  └─ ptyprocess.spawn(['/bin/bash', '-l'])
       │
       ├─ asyncio loop: os.read(fd, 4096) → websocket.send(data)
       │
       └─ asyncio loop: websocket.recv() → os.write(fd, data)
            │
            └─ 特殊处理: \x1b[RESIZE:<cols>;<rows>]
                 → TIOCSWINSZ ioctl 调整 PTY 大小
```

**关键点：**
- 每个 WebSocket 连接对应一个独立的 bash 进程
- 使用 `fctl` 设置 PTY fd 为非阻塞，配合 `asyncio` 事件循环
- 连接断开时 kill bash 进程
- 监听 `0.0.0.0:5001`
- 日志记录连接/断开事件

**关于 ptyprocess 阻塞读的处理：**
- 使用 `loop.run_in_executor(ThreadPoolExecutor)` 将阻塞的 `process.read()` 放到线程池
- 或直接使用 `os.read()` + `os.set_blocking()` 的低级 fd 操作配合 `add_reader`

### 2. 前端 — WTerm 终端页面

**wterm 加载方式：** 通过 importmap 从 CDN 加载（避免引入 npm/bundler）

```html
<script type="importmap">
{
  "imports": {
    "@wterm/dom": "https://cdn.jsdelivr.net/npm/@wterm/dom@0.2.1/+esm"
  }
}
</script>
```

**CSS 处理：**
- wterm CSS 从 CDN 获取后注入页面（`fetch()` + `document.adoptedStyleSheets` 或加载到 `<style>`）
- 或在 install.sh 中下载 CSS 文件到 `static/lib/wterm/` 本地引用

**终端页面布局：**
```
┌─────────────────────────────────────────┐
│  ◀ 返回面板    Web 终端                  │
├─────────────────────────────────────────┤
│ ┌─────────────────────────────────────┐ │
│ │  user@debian:~$ _                   │ │
│ │                                     │ │
│ │      wterm 全屏终端                   │ │
│ │                                     │ │
│ │                                     │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

**terminal.js 核心逻辑：**
```javascript
import { WTerm } from "@wterm/dom";
import "@wterm/dom/css";

const term = new WTerm(document.getElementById('terminal'));
await term.init();

// WebSocket 连接
const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
const ws = new WebSocket(`${protocol}//${location.hostname}:5001`);
ws.binaryType = 'arraybuffer';

term.onData(data => ws.send(data));

ws.onmessage = (e) => {
  if (typeof e.data === 'string') {
    term.writeString(e.data);
  }
};

// 窗口 resize 时发送信令
new ResizeObserver(() => {
  const { cols, rows } = term.getSize();
  ws.send(`\x1b[RESIZE:${cols};${rows}]`);
}).observe(term.element);
```

### 3. install.sh 变更

新增：
```bash
# 安装 WebSocket 服务依赖
pip3 install websockets ptyprocess

# 注册 WebSocket 服务
cat > /etc/systemd/system/nasmanager-ws.service << 'EOF'
[Unit]
Description=NAS Manager WebSocket Terminal
After=network.target

[Service]
Type=simple
User=nasmanager
Group=nasmanager
WorkingDirectory=/opt/nasmanager
ExecStart=/usr/bin/python3 /opt/nasmanager/ws_server.py
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

systemctl enable nasmanager-ws.service
systemctl start nasmanager-ws.service
```

### 4. 导航页面集成

在 `templates/index.html` 的"系统工具"分类中增加：
```html
<section id="system">
  <h2>系统工具</h2>
  <nav>
    <ul>
      ...
      <li><a href="/terminal" class="terminal-link">Web 终端</a></li>
    </ul>
  </nav>
</section>
```

终端链接不跳转到外部服务，而是跳转到 Flask 自己的 `/terminal` 页面。

---

## 安全注意事项

1. **终端用户身份：** bash 以 `nasmanager` 用户运行，该用户已有 sudo 权限（用于 systemctl）
2. **端口隔离：** WS 服务绑 5001，不对外暴露（可配置 iptables/nftables）
3. **连接限制：** ws_server.py 中可限制最大并发连接数（如 `MAX_CONNECTIONS = 5`）
4. **会话超时：** 连接闲置超过一定时间（如 15 分钟）自动断开
5. **历史记录：** 终端内容不会被记录到日志中（无持久化）
6. **认证：** 如果需要，可在 WebSocket 连接时验证 `Sec-WebSocket-Protocol` 头传递的 token

---

## 实施步骤

| # | 步骤 | 文件 |
|---|------|------|
| 1 | 更新 requirements.txt | `requirements.txt` |
| 2 | 创建 ws_server.py | `ws_server.py` |
| 3 | 创建 terminal.html | `templates/terminal.html` |
| 4 | 创建 terminal.js | `static/js/terminal.js` |
| 5 | 创建 terminal.css | `static/css/terminal.css` |
| 6 | 更新 index.html（加终端链接） | `templates/index.html` |
| 7 | 更新 install.sh | `install.sh` |
| 8 | 测试验证 | — |
| 9 | 更新 AGENTS.md | `AGENTS.md` |

---

## 备选方案讨论

### 为什么不直接集成到 Flask 进程内？
Flask (Werkzeug) 不支持 WebSocket。选项：
- Flask-SocketIO：需要 eventlet/gevent 切换 WSGI server，复杂度高且 dev 调试不便
- **独立服务**（选择方案）：干净分离，各自独立部署和重启

### 为什么不使用 node-pty + Node.js？
虽然 wterm 官方示例使用 node-pty，但本项目技术栈是 Python。`ptyprocess` 是等价的 Python 库。
