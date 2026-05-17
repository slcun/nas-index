let allServices = [];
let categories = {};
let pollingInterval = null;

document.addEventListener('DOMContentLoaded', () => {
    init();
});

function init() {
    document.getElementById('current-year').textContent = new Date().getFullYear();
    fetchHostInfo();
    fetchServices();
    pollingInterval = setInterval(fetchServices, 15000);
    setupSearch();
}

async function fetchHostInfo() {
    try {
        const resp = await fetch('/api/host/info');
        const data = await resp.json();
        document.getElementById('hostname').textContent = data.hostname;
        document.getElementById('host-ip').textContent = data.ip;
        document.getElementById('current-host').textContent = `${window.location.protocol}//${data.ip}:5000`;
    } catch (e) {
        document.getElementById('hostname').textContent = window.location.hostname;
        document.getElementById('host-ip').textContent = window.location.hostname;
        document.getElementById('current-host').textContent = window.location.href;
    }
}

async function fetchServices() {
    try {
        const resp = await fetch('/api/services');
        const data = await resp.json();
        allServices = data.services;
        categories = data.categories;
        renderServices();
    } catch (e) {
        console.error('获取服务列表失败:', e);
    }
}

function renderServices() {
    const container = document.getElementById('service-container');
    const searchTerm = (document.getElementById('search-input').value || '').toLowerCase().trim();

    let filtered = allServices;
    if (searchTerm) {
        filtered = allServices.filter(s =>
            s.display_name.toLowerCase().includes(searchTerm) ||
            (s.description || '').toLowerCase().includes(searchTerm) ||
            s.name.toLowerCase().includes(searchTerm)
        );
    }

    const grouped = {};
    filtered.forEach(s => {
        const cat = s.category || 'other';
        if (!grouped[cat]) grouped[cat] = [];
        grouped[cat].push(s);
    });

    container.innerHTML = '';

    const catKeys = Object.keys(grouped);
    if (catKeys.length === 0) {
        container.innerHTML = '<div class="no-services">没有匹配的服务</div>';
        updateStats();
        return;
    }

    catKeys.forEach(catKey => {
        const section = document.createElement('section');
        const catName = categories[catKey] || catKey;
        const catServices = grouped[catKey];

        let html = `<h2>${escapeHtml(catName)}</h2><nav><ul>`;
        catServices.forEach(s => {
            html += renderCard(s);
        });
        html += '</ul></nav>';
        section.innerHTML = html;
        container.appendChild(section);
    });

    updateStats();
}

function renderCard(service) {
    const isActive = service.active_state === 'active';
    const statusClass = isActive ? 'active' : (service.active_state === 'failed' ? 'failed' : 'inactive');
    const isWeb = service.web && service.port;

    let href = '#';
    let target = '';
    let rel = '';
    let linkClass = 'service-card';

    if (isWeb) {
        const protocol = window.location.protocol;
        const hostname = window.location.hostname;
        const port = service.port;
        const path = service.path || '';
        href = `${protocol}//${hostname}:${port}${path}`;
        target = '_blank';
        rel = 'noopener noreferrer';
        linkClass += ' web-link';
    } else {
        linkClass += ' non-web';
    }

    let actionsHtml = '';
    if (service.managed) {
        actionsHtml = `<div class="actions">
            ${!isActive ? `<button class="btn-start" data-name="${service.name}" title="启动">▶</button>` : ''}
            ${isActive ? `<button class="btn-stop" data-name="${service.name}" title="停止">■</button>` : ''}
            <button class="btn-restart" data-name="${service.name}" title="重启">↻</button>
        </div>`;
    }

    const badge = service.unit_file_state ? `<span class="service-badge">${service.unit_file_state}</span>` : '';

    return `<li>
        <a href="${href}" target="${target}" rel="${rel}" class="${linkClass}" data-name="${service.name}">
            <div class="service-card-header">
                <span class="status-dot ${statusClass}"></span>
                <span class="service-name">${escapeHtml(service.display_name)}</span>
                ${badge}
            </div>
            ${service.description ? `<div class="service-desc">${escapeHtml(service.description)}</div>` : ''}
            <div class="service-meta">
                ${service.port ? `<span>端口 ${service.port}</span>` : ''}
                ${!isWeb ? '<span>非网页服务</span>' : ''}
            </div>
            ${actionsHtml}
        </a>
    </li>`;
}

function updateStats() {
    const total = allServices.length;
    const active = allServices.filter(s => s.active_state === 'active').length;
    const inactive = allServices.filter(s => s.active_state === 'inactive').length;

    document.getElementById('total-count').textContent = total;
    document.getElementById('active-count').textContent = active;
    document.getElementById('inactive-count').textContent = inactive;
}

function setupSearch() {
    const input = document.getElementById('search-input');
    let debounceTimer;
    input.addEventListener('input', () => {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(renderServices, 200);
    });
}

function setupActionButtons() {
    document.querySelectorAll('.btn-start, .btn-stop, .btn-restart').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.preventDefault();
            e.stopPropagation();

            const name = btn.dataset.name;
            const action = btn.classList.contains('btn-start') ? 'start'
                        : btn.classList.contains('btn-stop') ? 'stop'
                        : 'restart';

            btn.disabled = true;
            const origText = btn.textContent;
            btn.textContent = '...';

            try {
                const resp = await fetch(`/api/services/${encodeURIComponent(name)}/${action}`, { method: 'POST' });
                const data = await resp.json();
                showToast(data.success ? 'success' : 'error',
                    `${name}: ${data.message || (data.success ? '操作成功' : '操作失败')}`);
                if (data.success) {
                    setTimeout(fetchServices, 500);
                }
            } catch (e) {
                showToast('error', `${name}: 请求失败`);
            } finally {
                btn.disabled = false;
                btn.textContent = origText;
            }
        });
    });
}

const observer = new MutationObserver(() => {
    setupActionButtons();
});
observer.observe(document.getElementById('service-container'), { childList: true, subtree: true });

function showToast(type, message) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transition = 'opacity 0.3s';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
