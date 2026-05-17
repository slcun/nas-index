import subprocess
import os
import re

DEMO_SERVICES = [
    {'name': 'jellyfin.service', 'display_name': 'Jellyfin', 'description': '媒体服务器', 'port': 8096, 'category': 'media', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'sonarr.service', 'display_name': 'Sonarr', 'description': '电视节目管理', 'port': 8989, 'category': 'media', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'radarr.service', 'display_name': 'Radarr', 'description': '电影管理', 'port': 7878, 'category': 'media', 'active_state': 'inactive', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'qbittorrent.service', 'display_name': 'qBittorrent', 'description': 'BT 下载工具', 'port': 8080, 'category': 'download', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'cockpit.service', 'display_name': 'Cockpit', 'description': 'Web 系统管理', 'port': 9090, 'category': 'system', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'docker.service', 'display_name': 'Docker', 'description': '容器引擎', 'category': 'system', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': False, 'managed': True},
    {'name': 'syncthing.service', 'display_name': 'Syncthing', 'description': '文件同步', 'port': 8384, 'category': 'backup', 'active_state': 'inactive', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'transmission.service', 'display_name': 'Transmission', 'description': 'BT 下载工具', 'port': 9091, 'category': 'download', 'active_state': 'inactive', 'unit_file_state': 'disabled', 'web': True, 'managed': True},
    {'name': 'prowlarr.service', 'display_name': 'Prowlarr', 'description': '索引器管理', 'port': 9696, 'category': 'other', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'immich.service', 'display_name': 'Immich', 'description': '照片管理', 'port': 2283, 'path': '/photos', 'category': 'media', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
    {'name': 'nginx.service', 'display_name': 'Nginx 文件管理器', 'description': 'Web 文件管理', 'port': 5001, 'path': '/file', 'category': 'files', 'active_state': 'active', 'unit_file_state': 'enabled', 'web': True, 'managed': True},
]


class ServiceManager:

    def __init__(self, config_manager):
        self.config = config_manager
        self._systemd_available = self._check_systemd()

    def _check_systemd(self):
        try:
            subprocess.run(['systemctl', '--version'], capture_output=True, timeout=5)
            return True
        except (FileNotFoundError, subprocess.TimeoutExpired):
            return False

    def list_services(self):
        if not self._systemd_available:
            return self._merge_demo_with_config()

        self.config.reload_if_changed()
        all_units = self._list_unit_files()
        manual_map = {s['name']: s for s in self.config.get_services()}
        exclude = self.config.get_exclude_services()

        candidates = []
        for unit in all_units:
            name = unit['name']
            state = unit['unit_file_state']
            if name in manual_map or name not in exclude:
                candidates.append(unit)

        auto_map = {}
        for unit in candidates:
            detail = self._get_service_detail(unit['name'])
            if detail:
                auto_map[unit['name']] = detail

        self._scan_etc_systemd(auto_map, exclude)

        result = []
        processed = set()

        for manual in self.config.get_services():
            name = manual['name']
            auto = auto_map.get(name, {})
            merged = {**auto, **manual}
            merged['managed'] = True
            merged.setdefault('active_state', 'unknown')
            result.append(merged)
            processed.add(name)

        for name, auto in auto_map.items():
            if name not in processed and name not in exclude:
                auto['managed'] = auto.get('unit_file_state') in ('enabled', 'static')
                auto.setdefault('web', self._guess_is_web(auto))
                auto.setdefault('category', 'other')
                result.append(auto)
                processed.add(name)

        return result

    def get_service(self, name):
        services = self.list_services()
        for s in services:
            if s['name'] == name:
                s['active_state'] = self._get_active_state(name)
                return s
        return None

    def start_service(self, name):
        try:
            result = self._run_systemctl('start', name, sudo=True)
            success = result.returncode == 0
            message = result.stderr.strip() if result.stderr else ('操作成功' if success else '操作失败')
            return success, message
        except Exception as e:
            return False, str(e)

    def stop_service(self, name):
        try:
            result = self._run_systemctl('stop', name, sudo=True)
            success = result.returncode == 0
            message = result.stderr.strip() if result.stderr else ('操作成功' if success else '操作失败')
            return success, message
        except Exception as e:
            return False, str(e)

    def restart_service(self, name):
        try:
            result = self._run_systemctl('restart', name, sudo=True)
            success = result.returncode == 0
            message = result.stderr.strip() if result.stderr else ('操作成功' if success else '操作失败')
            return success, message
        except Exception as e:
            return False, str(e)

    def get_logs(self, name, lines=50):
        try:
            result = self._run_systemctl('--no-pager', '-n', str(lines), '-u', name)
            return result.stdout
        except Exception as e:
            return f'Error: {e}'

    def _get_active_state(self, name):
        result = self._run_systemctl('is-active', name)
        return result.stdout.strip() if result.returncode == 0 else 'inactive'

    def _list_unit_files(self):
        result = self._run_systemctl('list-unit-files', '--type=service', '--no-legend')
        services = []
        for line in result.stdout.strip().split('\n'):
            line = line.strip()
            if not line:
                continue
            parts = line.split()
            if len(parts) >= 2:
                services.append({'name': parts[0], 'unit_file_state': parts[1]})
        return services

    def _get_service_detail(self, name):
        result = self._run_systemctl('show', '-p',
            'Names,Description,LoadState,ActiveState,SubState,UnitFileState',
            name)
        if result.returncode != 0:
            return None
        detail = {}
        for line in result.stdout.strip().split('\n'):
            if '=' in line:
                key, value = line.split('=', 1)
                detail[key.lower()] = value
        return {
            'name': detail.get('names', name),
            'display_name': self._name_to_display(name),
            'description': detail.get('description', ''),
            'active_state': detail.get('activestate', 'unknown'),
            'sub_state': detail.get('substate', 'unknown'),
            'unit_file_state': detail.get('unitfilestate', 'unknown'),
        }

    def _scan_etc_systemd(self, auto_map, exclude):
        etc_dir = '/etc/systemd/system'
        if not os.path.isdir(etc_dir):
            return
        try:
            for entry in os.listdir(etc_dir):
                if entry.endswith('.service') and entry not in exclude and entry not in auto_map:
                    fpath = os.path.join(etc_dir, entry)
                    if os.path.isfile(fpath) and not os.path.islink(fpath):
                        detail = self._get_service_detail(entry)
                        if detail:
                            auto_map[entry] = detail
        except PermissionError:
            pass

    def _run_systemctl(self, *args, sudo=False):
        cmd = []
        if sudo:
            cmd.append('sudo')
        cmd.append('systemctl')
        cmd.extend(args)
        try:
            return subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        except subprocess.TimeoutExpired:
            raise RuntimeError(f'Command timed out: {" ".join(cmd)}')

    def _name_to_display(self, name):
        if name.endswith('.service'):
            name = name[:-8]
        name = name.replace('@', '')
        return name.replace('-', ' ').title()

    def _guess_is_web(self, service):
        web_keywords = ['web', 'http', 'api', 'ui', 'gui', 'dashboard', 'admin',
                       'cms', 'jenkins', 'jellyfin', 'sonarr', 'radarr', 'qbittorrent',
                       'transmission', 'cockpit', 'syncthing', 'grafana', 'prometheus',
                       'portainer', 'nginx', 'apache', 'immich', 'prowlarr']
        name = service.get('name', '').lower()
        desc = service.get('description', '').lower()
        for kw in web_keywords:
            if kw in name or kw in desc:
                return True
        return False

    def _merge_demo_with_config(self):
        manual_map = {s['name']: s for s in self.config.get_services()}
        merged = []
        seen = set()

        for manual in self.config.get_services():
            name = manual['name']
            demo = next((d for d in DEMO_SERVICES if d['name'] == name), {})
            merged.append({**demo, **manual, 'managed': True})
            seen.add(name)

        for demo in DEMO_SERVICES:
            if demo['name'] not in seen:
                merged.append(demo)
            seen.add(demo['name'])

        return merged
