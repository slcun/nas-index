import { WTerm } from '@wterm/dom';

let term = null;
let ws = null;
let reconnectTimer = null;
let isConnecting = false;
let currentCols = 80;
let currentRows = 24;
let reconnectCount = 0;
let maxReconnectAttempts = 10;
let reconnectDelay = 1000;

function getWsUrl() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = location.hostname;
    const port = 5001;
    return `${protocol}//${host}:${port}`;
}

async function init() {
    document.getElementById('term-host').textContent = window.location.hostname;

    const el = document.getElementById('terminal');
    el.classList.add('theme-monokai');
    console.log('[wterm] 开始初始化, cols=%d, rows=%d', currentCols, currentRows);
    try {
        term = new WTerm(el, {
            cols: currentCols,
            rows: currentRows,
            onData: (data) => {
                if (ws && ws.readyState === WebSocket.OPEN) {
                    ws.send(data);
                }
            },
            onResize: (cols, rows) => {
                currentCols = cols;
                currentRows = rows;
                if (ws && ws.readyState === WebSocket.OPEN) {
                    ws.send(`\x1b[RESIZE:${cols};${rows}]`);
                }
            },
        });
        console.log('[wterm] WTerm 实例已创建, 开始调用 init()');
        await term.init();
        console.log('[wterm] WTerm 初始化成功');
    } catch (err) {
        console.error('[wterm] WTerm 初始化失败:', err);
        setStatus('error');
        return;
    }

    connect();
}

function connect() {
    if (isConnecting) return;
    isConnecting = true;
    setStatus('connecting');

    const url = getWsUrl();
    console.log('[wterm] 开始连接 WebSocket:', url);
    ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
        isConnecting = false;
        reconnectCount = 0; // 重置重连计数
        setStatus('connected');
        console.log('[wterm] WebSocket 已连接');
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }

        ws.send(`\x1b[RESIZE:${currentCols};${currentRows}]`);
    };

    ws.onmessage = (e) => {
        const data = e.data;
        if (data instanceof ArrayBuffer) {
            term.write(new Uint8Array(data));
        } else if (typeof data === 'string') {
            term.write(data);
        }
    };

    ws.onclose = (e) => {
        isConnecting = false;
        setStatus('disconnected');
        console.warn('[wterm] WebSocket 已断开, code=%d, reason=%s', e.code, e.reason);
        scheduleReconnect();
    };

    ws.onerror = (e) => {
        isConnecting = false;
        setStatus('error');
        console.error('[wterm] WebSocket 连接错误', e);
    };

    term.focus();
}

function scheduleReconnect() {
    if (reconnectTimer) return;
    
    if (reconnectCount >= maxReconnectAttempts) {
        console.error('[wterm] 已达到最大重连次数 (%d/%d)，停止重连', reconnectCount, maxReconnectAttempts);
        setStatus('error');
        return;
    }
    
    reconnectCount++;
    
    // 指数退避策略：1s, 2s, 4s, 8s, 16s，最大16s
    const delay = Math.min(reconnectDelay * Math.pow(2, reconnectCount - 1), 16000);
    console.log('[wterm] %d秒后进行第%d次重连 (最大%d次)', delay/1000, reconnectCount, maxReconnectAttempts);
    
    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
    }, delay);
}

// 页面卸载时清理资源
window.addEventListener('beforeunload', () => {
    if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }
    if (ws) {
        ws.close();
        ws = null;
    }
});

function setStatus(state) {
    const el = document.getElementById('term-status');
    const map = {
        connecting: [`● 重连中(${reconnectCount}/${maxReconnectAttempts})`, 'status-connecting'],
        connected: ['● 已连接', 'status-connected'],
        disconnected: ['● 已断开', 'status-disconnected'],
        error: [`● 连接失败(${reconnectCount}/${maxReconnectAttempts})`, 'status-error'],
    };
    if (reconnectCount === 0) {
        map.connecting = ['● 连接中', 'status-connecting'];
        map.error = ['● 连接失败', 'status-error'];
    }
    const [text, cls] = map[state] || ['● 未知', ''];
    el.textContent = text;
    el.className = 'term-status ' + cls;
}

document.addEventListener('DOMContentLoaded', init);
