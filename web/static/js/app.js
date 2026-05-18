let allServices = [];
let categories = {};
let activeCategory = null;
let pollingInterval = null;

const SVG = {
    start: '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>',
    stop: '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="12" height="16" rx="2"/></svg>',
    restart: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>',
    port: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="12" x2="2" y2="12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg>',
    cube: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="21 16 8 12 3 16"/><line x1="3" y1="16" x2="3" y2="22"/><polyline points="21 16 21 22"/><polyline points="12 12 12 18"/><rect x="3" y="6" width="18" height="6" rx="2"/></svg>',
    empty: '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>',
};

document.addEventListener('DOMContentLoaded', init);

function init() {
    initTheme();
    checkAuth();
    fetchHostInfo();
    fetchServices();
    pollingInterval = setInterval(fetchServices, 15000);
    setupSearch();
}

function initTheme() {
    const saved = localStorage.getItem('nas-theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const theme = saved || (prefersDark ? 'dark' : 'light');
    setTheme(theme);

    document.getElementById('theme-toggle').addEventListener('click', () => {
        const next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
        setTheme(next);
        localStorage.setItem('nas-theme', next);
    });
}

function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    const showMoon = theme === 'dark';
    document.getElementById('theme-icon-sun').style.display = showMoon ? 'none' : '';
    document.getElementById('theme-icon-moon').style.display = showMoon ? '' : 'none';
}

async function checkAuth() {
    try {
        const resp = await fetch('/api/auth/check');
        const data = await resp.json();
        if (data.authenticated && data.username) {
            const el = document.getElementById('user-badge');
            el.style.display = 'inline-flex';
            document.getElementById('username-display').textContent = data.username;
        }
    } catch {}
}

async function fetchHostInfo() {
    try {
        const resp = await fetch('/api/host/info');
        const data = await resp.json();
        document.getElementById('hostname').textContent = data.hostname;
        document.getElementById('host-ip').textContent = data.ip;
    } catch {
        document.getElementById('hostname').textContent = window.location.hostname;
        document.getElementById('host-ip').textContent = window.location.hostname;
    }
}

async function fetchServices() {
    try {
        const resp = await fetch('/api/services');
        if (resp.status === 401) {
            window.location.href = '/login';
            return;
        }
        const data = await resp.json();
        allServices = data.services || [];
        categories = data.categories || {};
        renderCategoryBar();
        renderServices();
    } catch (e) {
        console.error('获取服务列表失败:', e);
    }
}

function renderCategoryBar() {
    const container = document.getElementById('category-bar');
    if (!container) return;

    const catKeys = [...new Set(allServices.map(s => s.category || 'other'))];

    let html = `<button class="filter-btn${activeCategory === null ? ' active' : ''}" data-cat="">全部</button>`;
    catKeys.forEach(key => {
        const name = categories[key] || key;
        html += `<button class="filter-btn${activeCategory === key ? ' active' : ''}" data-cat="${escapeHtml(key)}">${escapeHtml(name)}</button>`;
    });
    container.innerHTML = html;

    container.querySelectorAll('.filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const cat = btn.dataset.cat;
            activeCategory = activeCategory === cat ? null : cat;
            renderCategoryBar();
            renderServices();
        });
    });
}

function renderServices() {
    const container = document.getElementById('service-container');
    const searchTerm = (document.getElementById('search-input').value || '').toLowerCase().trim();

    let filtered = allServices;

    if (searchTerm) {
        filtered = filtered.filter(s =>
            (s.display_name || '').toLowerCase().includes(searchTerm) ||
            (s.description || '').toLowerCase().includes(searchTerm) ||
            s.name.toLowerCase().includes(searchTerm)
        );
    }

    if (activeCategory) {
        filtered = filtered.filter(s => (s.category || 'other') === activeCategory);
    }

    if (filtered.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                ${SVG.empty}
                <p>${searchTerm || activeCategory ? '没有匹配的服务' : '正在加载服务列表...'}</p>
            </div>`;
        updateStats();
        return;
    }

    const grouped = {};
    filtered.forEach(s => {
        const cat = s.category || 'other';
        if (!grouped[cat]) grouped[cat] = [];
        grouped[cat].push(s);
    });

    container.innerHTML = '';

    Object.keys(grouped).forEach(catKey => {
        const catName = categories[catKey] || catKey;
        const services = grouped[catKey];

        const section = document.createElement('div');
        section.className = 'category-section';

        section.innerHTML = `
            <div class="category-header">
                <h2>${escapeHtml(catName)}</h2>
                <span class="category-count">${services.length}</span>
            </div>
            <div class="service-grid">
                ${services.map(s => renderCard(s)).join('')}
            </div>`;

        container.appendChild(section);
    });

    updateStats();
    setupActions();
}

function renderCard(svc) {
    const isActive = svc.active_state === 'active';
    const statusClass = isActive ? 'active' : (svc.active_state === 'failed' ? 'failed' : (svc.active_state === 'inactive' ? 'inactive' : 'unknown'));
    const isWeb = svc.web && svc.port;

    let href = '#';
    let target = '';
    let rel = '';
    let linkClass = 'service-card status-' + statusClass;

    if (isWeb) {
        const port = svc.port;
        const path = svc.path || '';
        href = `//${window.location.hostname}:${port}${path}`;
        target = '_blank';
        rel = 'noopener noreferrer';
        linkClass += ' web-link';
    }

    let actionsHtml = '';
    if (svc.managed) {
        actionsHtml = `<div class="actions">
            ${!isActive ? `<button class="action-btn start" data-name="${svc.name}" data-action="start" title="启动">${SVG.start}</button>` : ''}
            ${isActive ? `<button class="action-btn stop" data-name="${svc.name}" data-action="stop" title="停止">${SVG.stop}</button>` : ''}
            <button class="action-btn restart" data-name="${svc.name}" data-action="restart" title="重启">${SVG.restart}</button>
        </div>`;
    }

    const badge = svc.unit_file_state && svc.unit_file_state !== 'unknown'
        ? `<span class="service-badge">${svc.unit_file_state}</span>` : '';

    const metaHtml = [];
    if (svc.port) metaHtml.push(`<span class="meta-tag">${SVG.port} ${svc.port}</span>`);
    if (!svc.web) metaHtml.push(`<span class="meta-tag">${SVG.cube} 非网页</span>`);

    return `<a href="${href}" target="${target}" rel="${rel}" class="${linkClass}" data-name="${svc.name}">
        <div class="service-card-inner">
            <div class="service-card-row1">
                <span class="status-dot ${statusClass}"></span>
                <span class="service-name">${escapeHtml(svc.display_name)}</span>
                ${badge}
            </div>
            ${svc.description ? `<div class="service-desc">${escapeHtml(svc.description)}</div>` : ''}
            ${metaHtml.length ? `<div class="service-meta">${metaHtml.join('')}</div>` : ''}
        </div>
        ${actionsHtml}
    </a>`;
}

function updateStats() {
    document.getElementById('total-count').textContent = allServices.length;
    document.getElementById('active-count').textContent = allServices.filter(s => s.active_state === 'active').length;
    document.getElementById('inactive-count').textContent = allServices.filter(s => s.active_state === 'inactive').length;
}

function setupSearch() {
    const input = document.getElementById('search-input');
    let timer;
    input.addEventListener('input', () => {
        clearTimeout(timer);
        timer = setTimeout(renderServices, 200);
    });
}

function setupActions() {
    document.querySelectorAll('.action-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.preventDefault();
            e.stopPropagation();

            const name = btn.dataset.name;
            const action = btn.dataset.action;
            btn.disabled = true;

            try {
                const resp = await fetch(`/api/services/${encodeURIComponent(name)}/${action}`, { method: 'POST' });
                const data = await resp.json();
                showToast(data.success ? 'success' : 'error',
                    `${name}: ${data.message || (data.success ? '操作成功' : '操作失败')}`);
                if (data.success) setTimeout(fetchServices, 500);
            } catch {
                showToast('error', `${name}: 请求失败`);
            } finally {
                btn.disabled = false;
            }
        });
    });
}

function showToast(type, message) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transition = 'opacity 0.25s';
        setTimeout(() => toast.remove(), 250);
    }, 3500);
}

function escapeHtml(text) {
    if (!text) return '';
    const d = document.createElement('div');
    d.textContent = text;
    return d.innerHTML;
}
