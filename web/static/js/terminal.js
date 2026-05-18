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
    return `${protocol}//${host}:${port}?cols=${currentCols}&rows=${currentRows}`;
}

async function init() {
    document.getElementById('term-host').textContent = window.location.hostname;

    const el = document.getElementById('terminal');
    el.classList.add('theme-monokai');
    try {
        term = new WTerm(el, {
            cols: currentCols,
            rows: currentRows,
            onData: (data) => {
                if (ws && ws.readyState === WebSocket.OPEN) {
                    if (typeof data === 'string') {
                        ws.send(data);
                    } else {
                        ws.send(data);
                    }
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
        await term.init();
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
    ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
        isConnecting = false;
        reconnectCount = 0;
        setStatus('connected');
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
        scheduleReconnect();
    };

    ws.onerror = () => {
        isConnecting = false;
        setStatus('error');
    };

    term.focus();
}

function scheduleReconnect() {
    if (reconnectTimer) return;

    if (reconnectCount >= maxReconnectAttempts) {
        setStatus('error');
        return;
    }

    reconnectCount++;
    const delay = Math.min(reconnectDelay * Math.pow(2, reconnectCount - 1), 16000);

    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
    }, delay);
}

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
