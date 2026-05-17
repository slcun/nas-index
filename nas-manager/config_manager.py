import os
import yaml
import time
import threading

class ConfigManager:

    def __init__(self, config_path):
        self.config_path = os.path.abspath(config_path)
        self._config = {}
        self._mtime = 0
        self._lock = threading.Lock()
        self.load()

    def load(self):
        if not os.path.exists(self.config_path):
            self._config = self._default_config()
            return
        try:
            with open(self.config_path, 'r', encoding='utf-8') as f:
                data = yaml.safe_load(f) or {}
            with self._lock:
                self._config = data
            self._mtime = os.path.getmtime(self.config_path)
        except Exception as e:
            print(f'Config load error: {e}')
            if not self._config:
                self._config = self._default_config()

    def reload_if_changed(self):
        try:
            mtime = os.path.getmtime(self.config_path)
            if mtime > self._mtime:
                self.load()
        except OSError:
            pass

    def get(self, key, default=None):
        with self._lock:
            return self._config.get(key, default)

    def get_services(self):
        return self.get('services', [])

    def get_exclude_services(self):
        return set(self.get('exclude_services', []))

    def get_categories(self):
        return self.get('categories', {})

    def save(self):
        with self._lock:
            with open(self.config_path, 'w', encoding='utf-8') as f:
                yaml.dump(self._config, f, allow_unicode=True, default_flow_style=False)
            self._mtime = os.path.getmtime(self.config_path)

    @staticmethod
    def _default_config():
        return {
            'services': [],
            'exclude_services': [],
            'categories': {
                'media': '媒体中心',
                'files': '文件管理',
                'download': '下载工具',
                'system': '系统工具',
                'backup': '备份与同步',
                'tools': '效率与工具',
                'other': '其他',
            }
        }
