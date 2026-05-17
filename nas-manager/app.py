import os
import socket
from flask import Flask, jsonify, render_template, request

from config_manager import ConfigManager
from service_manager import ServiceManager

app = Flask(__name__)

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CONFIG_PATH = os.path.join(BASE_DIR, 'config.yaml')

config_mgr = ConfigManager(CONFIG_PATH)
svc_mgr = ServiceManager(config_mgr)


def get_host_info():
    hostname = socket.gethostname()
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(('10.255.255.255', 1))
        ip = s.getsockname()[0]
        s.close()
    except Exception:
        ip = '127.0.0.1'
    return {'hostname': hostname, 'ip': ip}


@app.route('/')
def index():
    return render_template('index.html')


@app.route('/terminal')
def terminal():
    return render_template('terminal.html')


@app.route('/api/services')
def api_services():
    services = svc_mgr.list_services()
    categories = config_mgr.get_categories()
    return jsonify({'services': services, 'categories': categories})


@app.route('/api/services/<path:name>')
def api_service_detail(name):
    service = svc_mgr.get_service(name)
    if not service:
        return jsonify({'error': 'Service not found'}), 404
    return jsonify(service)


@app.route('/api/services/<path:name>/start', methods=['POST'])
def api_start_service(name):
    success, message = svc_mgr.start_service(name)
    return jsonify({'success': success, 'message': message})


@app.route('/api/services/<path:name>/stop', methods=['POST'])
def api_stop_service(name):
    success, message = svc_mgr.stop_service(name)
    return jsonify({'success': success, 'message': message})


@app.route('/api/services/<path:name>/restart', methods=['POST'])
def api_restart_service(name):
    success, message = svc_mgr.restart_service(name)
    return jsonify({'success': success, 'message': message})


@app.route('/api/services/<path:name>/logs')
def api_service_logs(name):
    lines = request.args.get('lines', 50, type=int)
    logs = svc_mgr.get_logs(name, lines)
    return jsonify({'name': name, 'logs': logs})


@app.route('/api/host/info')
def api_host_info():
    return jsonify(get_host_info())


@app.route('/api/config', methods=['GET', 'PUT'])
def api_config():
    if request.method == 'PUT':
        data = request.get_json(force=True)
        if data:
            config_mgr.save()
            return jsonify({'success': True, 'message': '配置已更新'})
        return jsonify({'success': False, 'message': '无效的配置数据'}), 400
    return jsonify(config_mgr._config)


if __name__ == '__main__':
    info = get_host_info()
    print(f'NAS 管理面板启动中...')
    print(f'  主机名: {info["hostname"]}')
    print(f'  IP 地址: {info["ip"]}')
    print(f'  访问地址: http://{info["ip"]}:5000')
    print(f'  配置文件: {CONFIG_PATH}')
    print()
    app.run(host='0.0.0.0', port=5000, debug=False)
