#!/bin/bash
# NAS 服务管理面板 — 一键安装脚本
# 适用于 Debian 13 (Trixie)
# 用法: sudo bash install.sh

set -e

APP_NAME="nasmanager"
APP_DIR="/opt/${APP_NAME}"
APP_USER="${APP_NAME}"
CONFIG_DIR="${APP_DIR}"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
SUDOERS_FILE="/etc/sudoers.d/${APP_NAME}"

echo "============================================"
echo " NAS 服务管理面板 — 安装脚本"
echo "============================================"
echo ""

if [ "$EUID" -ne 0 ]; then
    echo "请使用 sudo 运行此脚本: sudo bash install.sh"
    exit 1
fi

echo "[1/7] 安装 Python 依赖..."
apt-get update -qq
apt-get install -y -qq python3 python3-pip python3-venv
python3 --version

echo "[2/7] 创建应用目录..."
mkdir -p "${APP_DIR}"
mkdir -p "${APP_DIR}/static/css"
mkdir -p "${APP_DIR}/static/js"
mkdir -p "${APP_DIR}/templates"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp -r "${SCRIPT_DIR}/app.py"          "${APP_DIR}/"
cp -r "${SCRIPT_DIR}/config_manager.py" "${APP_DIR}/"
cp -r "${SCRIPT_DIR}/service_manager.py" "${APP_DIR}/"
cp -r "${SCRIPT_DIR}/config.yaml"     "${APP_DIR}/"
cp -r "${SCRIPT_DIR}/requirements.txt" "${APP_DIR}/"
cp -r "${SCRIPT_DIR}/static/css/"*    "${APP_DIR}/static/css/"
cp -r "${SCRIPT_DIR}/static/js/"*     "${APP_DIR}/static/js/"
cp -r "${SCRIPT_DIR}/templates/"*     "${APP_DIR}/templates/"

echo "[3/7] 安装 Python 包..."
pip3 install -r "${APP_DIR}/requirements.txt" -q

echo "[4/7] 创建系统用户..."
if ! id -u "${APP_USER}" &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "${APP_USER}"
fi
chown -R "${APP_USER}:${APP_USER}" "${APP_DIR}"
chmod -R 755 "${APP_DIR}"

echo "[5/7] 配置 sudo 权限..."
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

echo "[6/7] 创建 systemd 服务..."
cat > "${SERVICE_FILE}" << 'EOF'
[Unit]
Description=NAS Service Management Panel
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=nasmanager
Group=nasmanager
WorkingDirectory=/opt/nasmanager
ExecStart=/usr/bin/python3 /opt/nasmanager/app.py
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${APP_NAME}"

echo "[7/7] 启动服务..."
systemctl start "${APP_NAME}" || {
    echo "启动失败，查看日志: journalctl -u ${APP_NAME} -n 50 --no-pager"
    exit 1
}

echo ""
echo "============================================"
echo " 安装完成!"
echo "============================================"
echo ""
echo "访问地址: http://$(hostname -I | awk '{print $1}'):5000"
echo ""
echo "管理命令:"
echo "  sudo systemctl status ${APP_NAME}    # 查看状态"
echo "  sudo systemctl restart ${APP_NAME}   # 重启"
echo "  sudo systemctl stop ${APP_NAME}      # 停止"
echo "  sudo journalctl -u ${APP_NAME} -f    # 查看日志"
echo ""
echo "配置文件: ${CONFIG_DIR}/config.yaml"
echo "  - 编辑后无需重启，系统会自动检测变更"
echo ""
