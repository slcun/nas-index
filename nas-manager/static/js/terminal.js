import { WTerm } from '@wterm/dom';

let term = null;
let ws = null;
let reconnectTimer = null;
let isConnecting = false;

function getWsUrl() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = location.hostname;
    const port = 5001;
    return `${protocol}//${host}:${port}`;
}

async function init() {
    document.getElementById('term-host').textContent = window.location.hostname;

    const el = document.getElementById('terminal');
    term = new WTerm(el);
    await term.init();

    connect();
}

function connect() {
    if (isConnecting) return;
    isConnecting = true;
    setStatus('connecting');

    ws = new WebSocket(getWsUrl());
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
        isConnecting = false;
        setStatus('connected');
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }

        const { cols, rows } = term.getSize();
        ws.send(`\x1b[RESIZE:${cols};${rows}]`);
    };

    ws.onmessage = (e) => {
        const data = e.data;
        if (data instanceof ArrayBuffer) {
            term.writeBytes(new Uint8Array(data));
        } else if (typeof data === 'string') {
            term.writeString(data);
        }
    };

    ws.onclose = () => {
        isConnecting = false;
        setStatus('disconnected');
        scheduleReconnect();
    };

    ws.onerror = () => {
        isConnecting = false;
        setStatus('error');
    };

    term.onData((data) => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(data);
        }
    });

    new ResizeObserver(() => {
        if (!term || !ws || ws.readyState !== WebSocket.OPEN) return;
        const { cols, rows } = term.getSize();
        ws.send(`\x1b[RESIZE:${cols};${rows}]`);
    }).observe(el);

    el.addEventListener('focus', () => term.focus());
    el.focus();
}

function scheduleReconnect() {
    if (reconnectTimer) return;
    reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
    }, 3000);
}

function setStatus(state) {
    const el = document.getElementById('term-status');
    const map = {
        connecting: ['● 连接中', 'status-connecting'],
        connected: ['● 已连接', 'status-connected'],
        disconnected: ['● 已断开', 'status-disconnected'],
        error: ['● 连接失败', 'status-error'],
    };
    const [text, cls] = map[state] || ['● 未知', ''];
    el.textContent = text;
    el.className = 'term-status ' + cls;
}

document.addEventListener('DOMContentLoaded', init);
