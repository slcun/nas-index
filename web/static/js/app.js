let allServices = [];
let availableTags = [];
let availableGroups = [];
let activeTag = null;
let activeGroup = null;
let pollingInterval = null;
let systemServicesCache = null;

const SVG = {
    start: '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>',
    stop: '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="12" height="16" rx="2"/></svg>',
    restart: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>',
    port: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="12" x2="2" y2="12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg>',
    cube: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="21 16 8 12 3 16"/><line x1="3" y1="16" x2="3" y2="22"/><polyline points="21 16 21 22"/><polyline points="12 12 12 18"/><rect x="3" y="6" width="18" height="6" rx="2"/></svg>',
    empty: '<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>',
    edit: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
    trash: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>',
};

document.addEventListener('DOMContentLoaded', init);

function init() {
    initTheme();
    checkAuth();
    fetchHostInfo();
    fetchServices();
    pollingInterval = setInterval(fetchServices, 15000);
    setupSearch();
    setupAddBtn();
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
            const menu = document.getElementById('user-menu');
            menu.style.display = '';
            document.getElementById('username-display').textContent = data.username;
            setupUserMenu();
        }
    } catch {}
}

function setupUserMenu() {
    const badge = document.getElementById('user-badge');
    const dropdown = document.getElementById('user-dropdown');

    badge.addEventListener('click', (e) => {
        e.stopPropagation();
        dropdown.classList.toggle('open');
    });

    document.addEventListener('click', () => {
        dropdown.classList.remove('open');
    }, false);

    dropdown.addEventListener('click', (e) => {
        e.stopPropagation();
    });

    document.getElementById('logout-btn').addEventListener('click', async () => {
        try {
            await fetch('/logout');
        } catch {}
        window.location.href = '/login';
    });
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
        availableTags = data.tags || [];
        availableGroups = data.groups || [];
        renderGroupBar();
        renderTagBar();
        renderServices();
    } catch (e) {
        console.error('获取服务列表失败:', e);
    }
}

function setupAddBtn() {
    const btn = document.getElementById('add-service-btn');
    if (btn) {
        btn.addEventListener('click', () => openServiceModal());
    }
}

function renderGroupBar() {
    const container = document.getElementById('group-bar');
    if (!container) return;

    const groups = [...new Set(allServices.map(s => s.group || '未分组'))];

    let html = `<button class="group-btn${activeGroup === null ? ' active' : ''}" data-group="">全部</button>`;
    groups.forEach(group => {
        html += `<button class="group-btn${activeGroup === group ? ' active' : ''}" data-group="${escapeHtml(group)}">${escapeHtml(group)}</button>`;
    });
    container.innerHTML = html;

    container.querySelectorAll('.group-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const group = btn.dataset.group;
            activeGroup = activeGroup === group ? null : group;
            renderGroupBar();
            renderServices();
        });
    });
}

function renderTagBar() {
    const container = document.getElementById('tag-bar');
    if (!container || availableTags.length === 0) {
        if (container) container.innerHTML = '';
        return;
    }

    let html = `<button class="tag-btn${activeTag === null ? ' active' : ''}" data-tag="">全部</button>`;
    availableTags.forEach(tag => {
        html += `<button class="tag-btn${activeTag === tag ? ' active' : ''}" data-tag="${escapeHtml(tag)}">${escapeHtml(tag)}</button>`;
    });
    container.innerHTML = html;

    container.querySelectorAll('.tag-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const tag = btn.dataset.tag;
            activeTag = activeTag === tag ? null : tag;
            renderTagBar();
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
            s.name.toLowerCase().includes(searchTerm) ||
            (s.tags || []).some(t => t.toLowerCase().includes(searchTerm)) ||
            (s.group || '').toLowerCase().includes(searchTerm)
        );
    }

    if (activeTag) {
        filtered = filtered.filter(s => (s.tags || []).includes(activeTag));
    }

    if (activeGroup) {
        filtered = filtered.filter(s => (s.group || '未分组') === activeGroup);
    }

    if (filtered.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                ${SVG.empty}
                <p>${searchTerm || activeTag || activeGroup ? '没有匹配的服务' : '暂无服务，点击上方「添加」按钮添加'}</p>
            </div>`;
        updateStats();
        return;
    }

    const grouped = {};
    filtered.forEach(s => {
        const groupKey = s.group || '未分组';
        if (!grouped[groupKey]) grouped[groupKey] = [];
        grouped[groupKey].push(s);
    });

    const sortedKeys = Object.keys(grouped).sort((a, b) => {
        if (a === '未分组') return 1;
        if (b === '未分组') return -1;
        return a.localeCompare(b, 'zh-CN');
    });

    container.innerHTML = '';

    sortedKeys.forEach(key => {
        const services = grouped[key];

        const section = document.createElement('div');
        section.className = 'category-section';

        section.innerHTML = `
            <div class="category-header">
                <h2>${escapeHtml(key)}</h2>
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

    const metaHtml = [];
    if (svc.port) metaHtml.push(`<span class="meta-tag">${SVG.port} ${svc.port}</span>`);
    if (!svc.web) metaHtml.push(`<span class="meta-tag">${SVG.cube} 非网页</span>`);

    const tagsHtml = (svc.tags || []).map(t =>
        `<span class="service-tag" data-tag="${escapeHtml(t)}">${escapeHtml(t)}</span>`
    ).join('');

    return `<a href="${href}" target="${target}" rel="${rel}" class="${linkClass}" data-name="${svc.name}">
        <div class="service-card-inner">
            <div class="service-card-row1">
                <span class="status-dot ${statusClass}"></span>
                <span class="service-name">${escapeHtml(svc.display_name)}</span>
                ${svc.unit_file_state && svc.unit_file_state !== 'unknown' ? `<span class="service-badge">${svc.unit_file_state}</span>` : ''}
            </div>
            ${svc.description ? `<div class="service-desc">${escapeHtml(svc.description)}</div>` : ''}
            ${metaHtml.length ? `<div class="service-meta">${metaHtml.join('')}</div>` : ''}
            ${tagsHtml ? `<div class="service-tags">${tagsHtml}</div>` : ''}
        </div>
        <div class="actions">
            ${!isActive ? `<button class="action-btn start" data-name="${svc.name}" data-action="start" title="启动">${SVG.start}</button>` : ''}
            ${isActive ? `<button class="action-btn stop" data-name="${svc.name}" data-action="stop" title="停止">${SVG.stop}</button>` : ''}
            <button class="action-btn restart" data-name="${svc.name}" data-action="restart" title="重启">${SVG.restart}</button>
            <button class="action-btn edit" data-name="${svc.name}" data-action="edit" title="编辑">${SVG.edit}</button>
            <button class="action-btn delete" data-name="${svc.name}" data-action="delete" title="删除">${SVG.trash}</button>
        </div>
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

            if (action === 'edit') {
                const svc = allServices.find(s => s.name === name);
                if (svc) openServiceModal(svc);
                return;
            }

            if (action === 'delete') {
                if (!confirm(`确定要删除服务「${name}」吗？`)) return;
                try {
                    const resp = await fetch(`/api/services/${encodeURIComponent(name)}`, { method: 'DELETE' });
                    const data = await resp.json();
                    showToast(data.success ? 'success' : 'error', data.message || (data.success ? '删除成功' : '删除失败'));
                    if (data.success) fetchServices();
                } catch {
                    showToast('error', '删除请求失败');
                }
                return;
            }

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

    document.querySelectorAll('.service-tag').forEach(tag => {
        tag.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            activeTag = tag.dataset.tag;
            renderTagBar();
            renderServices();
        });
    });
}

// ========== 服务编辑弹窗 ==========

async function openServiceModal(existingSvc) {
    const isEdit = !!existingSvc;
    const modal = document.getElementById('service-modal');
    const title = document.getElementById('modal-title');

    title.textContent = isEdit ? '编辑服务' : '添加服务';

    const nameInput = document.getElementById('svc-name');
    const nameGroup = document.getElementById('svc-name-group');
    const selectGroup = document.getElementById('svc-system-select-group');

    if (isEdit) {
        nameGroup.style.display = '';
        selectGroup.style.display = 'none';
        nameInput.value = existingSvc.name;
        nameInput.readOnly = true;
    } else {
        nameGroup.style.display = 'none';
        selectGroup.style.display = '';
        nameInput.readOnly = false;
        nameInput.value = '';
    }

    document.getElementById('svc-display-name').value = isEdit ? (existingSvc.display_name || '') : '';
    document.getElementById('svc-description').value = isEdit ? (existingSvc.description || '') : '';
    document.getElementById('svc-port').value = isEdit ? (existingSvc.port || '') : '';
    document.getElementById('svc-path').value = isEdit ? (existingSvc.path || '') : '';
    document.getElementById('svc-web').checked = isEdit ? existingSvc.web : true;
    document.getElementById('svc-tags').value = isEdit ? (existingSvc.tags || []).join(', ') : '';
    document.getElementById('svc-group').value = isEdit ? (existingSvc.group || '') : '';

    const groupDatalist = document.getElementById('group-list');
    groupDatalist.innerHTML = availableGroups.map(g => `<option value="${escapeHtml(g)}">`).join('');

    if (!isEdit) {
        await loadSystemServices();
    }

    modal.classList.add('open');

    const closeBtn = document.getElementById('modal-close');
    const cancelBtn = document.getElementById('modal-cancel');
    const saveBtn = document.getElementById('modal-save');

    const closeHandler = () => modal.classList.remove('open');
    const saveHandler = () => saveService(isEdit, existingSvc);

    closeBtn.onclick = closeHandler;
    cancelBtn.onclick = closeHandler;
    saveBtn.onclick = saveHandler;

    modal.querySelector('.modal-overlay').onclick = closeHandler;

    const systemSelect = document.getElementById('svc-system-select');
    if (!isEdit) {
        systemSelect.onchange = () => {
            const opt = systemSelect.selectedOptions[0];
            if (opt && opt.value) {
                nameInput.value = opt.value;
                document.getElementById('svc-display-name').value = opt.dataset.display || '';
                document.getElementById('svc-description').value = opt.dataset.desc || '';
            }
        };
    }
}

async function loadSystemServices() {
    const select = document.getElementById('svc-system-select');
    if (systemServicesCache) {
        renderSystemSelect(select, systemServicesCache);
        return;
    }

    select.innerHTML = '<option value="">加载中...</option>';
    try {
        const resp = await fetch('/api/system/services');
        const data = await resp.json();
        systemServicesCache = data.services || [];
        renderSystemSelect(select, systemServicesCache);
    } catch {
        select.innerHTML = '<option value="">加载失败</option>';
    }
}

function renderSystemSelect(select, services) {
    let html = '<option value="">-- 从系统服务选择 --</option>';
    html += '<option value="__custom">自定义服务...</option>';

    const unconfigured = services.filter(s => !s.configured);
    const configured = services.filter(s => s.configured);

    if (unconfigured.length > 0) {
        html += '<optgroup label="未配置">';
        unconfigured.forEach(s => {
            const displayName = s.name.replace('.service', '');
            html += `<option value="${escapeHtml(s.name)}" data-display="${escapeHtml(displayName)}" data-desc="${escapeHtml(s.description)}">${escapeHtml(displayName)} — ${escapeHtml(s.description)}</option>`;
        });
        html += '</optgroup>';
    }

    if (configured.length > 0) {
        html += '<optgroup label="已配置">';
        configured.forEach(s => {
            const displayName = s.name.replace('.service', '');
            html += `<option value="${escapeHtml(s.name)}" data-display="${escapeHtml(displayName)}" data-desc="${escapeHtml(s.description)}">${escapeHtml(displayName)} — ${escapeHtml(s.description)}</option>`;
        });
        html += '</optgroup>';
    }

    select.innerHTML = html;

    select.onchange = () => {
        const nameInput = document.getElementById('svc-name');
        if (select.value === '__custom') {
            document.getElementById('svc-name-group').style.display = '';
            nameInput.value = '';
            nameInput.readOnly = false;
            nameInput.focus();
        } else if (select.value) {
            document.getElementById('svc-name-group').style.display = '';
            nameInput.value = select.value;
            nameInput.readOnly = true;
            const opt = select.selectedOptions[0];
            if (opt) {
                document.getElementById('svc-display-name').value = opt.dataset.display || '';
                document.getElementById('svc-description').value = opt.dataset.desc || '';
            }
        } else {
            document.getElementById('svc-name-group').style.display = 'none';
        }
    };
}

async function saveService(isEdit, existingSvc) {
    const nameInput = document.getElementById('svc-name');
    const name = nameInput.value.trim();

    if (!name) {
        showToast('error', '请输入服务名称');
        return;
    }

    const tagsStr = document.getElementById('svc-tags').value.trim();
    const tags = tagsStr ? tagsStr.split(/[,，]/).map(t => t.trim()).filter(t => t) : [];

    const svc = {
        name: name,
        display_name: document.getElementById('svc-display-name').value.trim(),
        description: document.getElementById('svc-description').value.trim(),
        port: parseInt(document.getElementById('svc-port').value) || 0,
        path: document.getElementById('svc-path').value.trim(),
        web: document.getElementById('svc-web').checked,
        tags: tags,
        group: document.getElementById('svc-group').value.trim(),
    };

    try {
        let resp;
        if (isEdit) {
            resp = await fetch(`/api/services/${encodeURIComponent(existingSvc.name)}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(svc),
            });
        } else {
            resp = await fetch('/api/services', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(svc),
            });
        }

        const data = await resp.json();
        if (data.success) {
            document.getElementById('service-modal').classList.remove('open');
            systemServicesCache = null;
            fetchServices();
        }
        showToast(data.success ? 'success' : 'error', data.message || (data.success ? '保存成功' : '保存失败'));
    } catch {
        showToast('error', '保存请求失败');
    }
}

// ========== 工具函数 ==========

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
