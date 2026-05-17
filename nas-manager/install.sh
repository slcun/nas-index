#!/bin/bash
# NAS 服务管理面板 — 一键安装脚本
# 适用于 Debian 13 (Trixie)
# 用法: sudo bash install.sh
# 卸载: sudo bash install.sh --uninstall

set -euo pipefail

APP_NAME="nasmanager"
APP_WS_NAME="nasmanager-ws"
APP_DIR="/opt/${APP_NAME}"
APP_USER="${APP_NAME}"
VENV_DIR="${APP_DIR}/venv"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
WS_SERVICE_FILE="/etc/systemd/system/${APP_WS_NAME}.service"
SUDOERS_FILE="/etc/sudoers.d/${APP_NAME}"

uninstall() {
    echo "============================================"
    echo " NAS 服务管理面板 — 卸载"
    echo "============================================"
    echo ""

    echo "[1/4] 停止服务..."
    systemctl stop "${APP_NAME}" 2>/dev/null || true
    systemctl stop "${APP_WS_NAME}" 2>/dev/null || true
    systemctl disable "${APP_NAME}" 2>/dev/null || true
    systemctl disable "${APP_WS_NAME}" 2>/dev/null || true

    echo "[2/4] 删除 systemd 服务文件..."
    rm -f "${SERVICE_FILE}" "${WS_SERVICE_FILE}"
    systemctl daemon-reload

    echo "[3/4] 删除应用目录..."
    rm -rf "${APP_DIR}"

    echo "[4/4] 清理系统用户和 sudoers..."
    rm -f "${SUDOERS_FILE}"
    if id -u "${APP_USER}" &>/dev/null; then
        userdel "${APP_USER}" 2>/dev/null || true
    fi

    echo ""
    echo "卸载完成。"
    exit 0
}

if [ "${1:-}" = "--uninstall" ]; then
    uninstall
fi

echo "============================================"
echo " NAS 服务管理面板 — 安装脚本"
echo "============================================"
echo ""

if [ "$EUID" -ne 0 ]; then
    echo "错误: 请使用 sudo 运行此脚本: sudo bash install.sh"
    exit 1
fi

if [ ! -f /etc/debian_version ]; then
    echo "错误: 此脚本仅适用于 Debian 系统"
    exit 1
fi

echo "[1/8] 安装系统依赖..."
apt-get update -qq
apt-get install -y -qq python3 python3-venv
python3 --version

echo "[2/8] 创建应用目录..."
mkdir -p "${APP_DIR}"
mkdir -p "${APP_DIR}/static/css"
mkdir -p "${APP_DIR}/static/js"
mkdir -p "${APP_DIR}/static/libs/wterm"
mkdir -p "${APP_DIR}/templates"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cp "${SCRIPT_DIR}/app.py"            "${APP_DIR}/"
cp "${SCRIPT_DIR}/ws_server.py"      "${APP_DIR}/"
cp "${SCRIPT_DIR}/config_manager.py" "${APP_DIR}/"
cp "${SCRIPT_DIR}/service_manager.py" "${APP_DIR}/"
cp "${SCRIPT_DIR}/requirements.txt"  "${APP_DIR}/"

if [ -f "${APP_DIR}/config.yaml" ]; then
    BACKUP="${APP_DIR}/config.yaml.bak.$(date +%Y%m%d%H%M%S)"
    echo "  检测到已有配置文件，备份至 ${BACKUP}"
    cp "${APP_DIR}/config.yaml" "${BACKUP}"
else
    cp "${SCRIPT_DIR}/config.yaml" "${APP_DIR}/"
fi

cp "${SCRIPT_DIR}/static/css/"*      "${APP_DIR}/static/css/"
cp "${SCRIPT_DIR}/static/js/"*       "${APP_DIR}/static/js/"
cp "${SCRIPT_DIR}/static/libs/wterm/"* "${APP_DIR}/static/libs/wterm/"
cp "${SCRIPT_DIR}/templates/"*       "${APP_DIR}/templates/"

echo "[3/8] 创建 Python 虚拟环境..."
if [ ! -d "${VENV_DIR}" ]; then
    python3 -m venv "${VENV_DIR}"
fi
"${VENV_DIR}/bin/pip" install --upgrade pip -q
"${VENV_DIR}/bin/pip" install -r "${APP_DIR}/requirements.txt" -q

echo "[4/8] 创建系统用户..."
if ! id -u "${APP_USER}" &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "${APP_USER}"
fi
chown -R "${APP_USER}:${APP_USER}" "${APP_DIR}"
chmod -R 755 "${APP_DIR}"

echo "[5/8] 配置 sudo 权限..."
cat > "${SUDOERS_FILE}" << EOF
# ${APP_NAME}: 允许管理 systemd 服务
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl start *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl status *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl is-active *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl show *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/systemctl list-unit-files *
${APP_USER} ALL=(ALL) NOPASSWD: /usr/bin/journalctl *
EOF
chmod 440 "${SUDOERS_FILE}"

echo "[6/8] 创建 systemd 服务..."
cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=NAS Service Management Panel
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_USER}
Group=${APP_USER}
WorkingDirectory=${APP_DIR}
ExecStart=${VENV_DIR}/bin/python3 ${APP_DIR}/app.py
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

cat > "${WS_SERVICE_FILE}" << EOF
[Unit]
Description=NAS Manager WebSocket Terminal
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_USER}
Group=${APP_USER}
WorkingDirectory=${APP_DIR}
ExecStart=${VENV_DIR}/bin/python3 ${APP_DIR}/ws_server.py
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${APP_NAME}"
systemctl enable "${APP_WS_NAME}"

echo "[7/8] 启动服务..."
systemctl restart "${APP_NAME}" || {
    echo "错误: Web 面板启动失败，查看日志: journalctl -u ${APP_NAME} -n 50 --no-pager"
    exit 1
}
systemctl restart "${APP_WS_NAME}" || {
    echo "错误: WebSocket 服务启动失败，查看日志: journalctl -u ${APP_WS_NAME} -n 50 --no-pager"
    exit 1
}

echo "[8/8] 验证服务状态..."
sleep 1
if systemctl is-active --quiet "${APP_NAME}" && systemctl is-active --quiet "${APP_WS_NAME}"; then
    echo "  ✓ 所有服务运行正常"
else
    echo "  ✗ 部分服务异常，请检查日志"
fi

echo ""
echo "============================================"
echo " 安装完成!"
echo "============================================"
echo ""
echo "访问地址: http://$(hostname -I | awk '{print $1}'):5000"
echo ""
echo "管理命令:"
echo "  sudo systemctl status ${APP_NAME}       # Web 面板状态"
echo "  sudo systemctl restart ${APP_NAME}      # 重启面板"
echo "  sudo systemctl status ${APP_WS_NAME}    # WebSocket 终端状态"
echo "  sudo systemctl restart ${APP_WS_NAME}   # 重启终端"
echo "  sudo journalctl -u ${APP_NAME} -f       # 面板日志"
echo "  sudo journalctl -u ${APP_WS_NAME} -f    # 终端日志"
echo ""
echo "配置文件: ${APP_DIR}/config.yaml"
echo "  - 编辑后无需重启，系统会自动检测变更"
echo ""
echo "卸载: sudo bash install.sh --uninstall"
echo ""
