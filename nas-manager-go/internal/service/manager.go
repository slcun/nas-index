package service

import (
	"nas-manager/internal/config"
)

// Manager 负责服务的管理
type Manager struct {
	configMgr    *config.Manager
	systemdAvail bool
}

// NewManager 创建一个新的服务管理器
func NewManager(configMgr *config.Manager) *Manager {
	return &Manager{
		configMgr:    configMgr,
		systemdAvail: checkSystemd(),
	}
}

// ListServices 列出所有服务
func (m *Manager) ListServices() []*ServiceInfo {
	if !m.systemdAvail {
		return m.listDemoServices()
	}

	m.configMgr.ReloadIfChanged()

	// 获取配置中的服务
	cfgServices := m.configMgr.GetServices()
	excludeServices := m.configMgr.GetExcludeServices()
	excludeMap := make(map[string]bool)
	for _, name := range excludeServices {
		excludeMap[name] = true
	}

	// 获取自动发现的服务
	autoServices := make(map[string]*ServiceInfo)

	// 从 systemctl list-unit-files 获取
	units, _ := listUnitFiles()
	for _, unit := range units {
		name := unit["name"]
		if excludeMap[name] {
			continue
		}
		info, _ := getServiceDetail(name)
		if info != nil {
			autoServices[name] = info
		}
	}

	// 从 /etc/systemd/system 目录扫描
	etcServices := scanEtcSystemd(excludeMap)
	for name, info := range etcServices {
		if _, exists := autoServices[name]; !exists {
			autoServices[name] = info
		}
	}

	var result []*ServiceInfo
	processed := make(map[string]bool)

	// 先添加配置中的服务（优先级高）
	for _, cfgSvc := range cfgServices {
		autoInfo := autoServices[cfgSvc.Name]
		merged := mergeServiceConfig(cfgSvc, autoInfo)
		result = append(result, merged)
		processed[cfgSvc.Name] = true
	}

	// 再添加自动发现的服务
	for name, autoInfo := range autoServices {
		if processed[name] {
			continue
		}
		autoInfo.Managed = autoInfo.UnitFileState == "enabled" || autoInfo.UnitFileState == "static"
		result = append(result, autoInfo)
		processed[name] = true
	}

	return result
}

// GetService 获取单个服务
func (m *Manager) GetService(name string) *ServiceInfo {
	services := m.ListServices()
	for _, svc := range services {
		if svc.Name == name {
			// 更新最新的状态
			if m.systemdAvail {
				svc.ActiveState = getActiveState(name)
			}
			return svc
		}
	}
	return nil
}

// StartService 启动服务
func (m *Manager) StartService(name string) (bool, string) {
	if !m.systemdAvail {
		return true, "Demo 模式：操作成功"
	}
	return startService(name)
}

// StopService 停止服务
func (m *Manager) StopService(name string) (bool, string) {
	if !m.systemdAvail {
		return true, "Demo 模式：操作成功"
	}
	return stopService(name)
}

// RestartService 重启服务
func (m *Manager) RestartService(name string) (bool, string) {
	if !m.systemdAvail {
		return true, "Demo 模式：操作成功"
	}
	return restartService(name)
}

// GetLogs 获取服务日志
func (m *Manager) GetLogs(name string, lines int) string {
	if !m.systemdAvail {
		return "Demo 模式：这是模拟的日志输出\n"
	}
	return getLogs(name, lines)
}

// Demo 服务数据
var demoServices = []*ServiceInfo{
	{
		Name:          "jellyfin.service",
		DisplayName:   "Jellyfin",
		Description:   "媒体服务器",
		Port:          8096,
		Category:      "media",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "sonarr.service",
		DisplayName:   "Sonarr",
		Description:   "电视节目管理",
		Port:          8989,
		Category:      "media",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "radarr.service",
		DisplayName:   "Radarr",
		Description:   "电影管理",
		Port:          7878,
		Category:      "media",
		ActiveState:   "inactive",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "qbittorrent.service",
		DisplayName:   "qBittorrent",
		Description:   "BT 下载工具",
		Port:          8080,
		Category:      "download",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "cockpit.service",
		DisplayName:   "Cockpit",
		Description:   "Web 系统管理",
		Port:          9090,
		Category:      "system",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "docker.service",
		DisplayName:   "Docker",
		Description:   "容器引擎",
		Category:      "system",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           false,
		Managed:       true,
	},
	{
		Name:          "syncthing.service",
		DisplayName:   "Syncthing",
		Description:   "文件同步",
		Port:          8384,
		Category:      "backup",
		ActiveState:   "inactive",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "transmission.service",
		DisplayName:   "Transmission",
		Description:   "BT 下载工具",
		Port:          9091,
		Category:      "download",
		ActiveState:   "inactive",
		UnitFileState: "disabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "prowlarr.service",
		DisplayName:   "Prowlarr",
		Description:   "索引器管理",
		Port:          9696,
		Category:      "other",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "immich.service",
		DisplayName:   "Immich",
		Description:   "照片管理",
		Port:          2283,
		Path:          "/photos",
		Category:      "media",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
	{
		Name:          "nginx.service",
		DisplayName:   "Nginx 文件管理器",
		Description:   "Web 文件管理",
		Port:          5001,
		Path:          "/file",
		Category:      "files",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
	},
}

func (m *Manager) listDemoServices() []*ServiceInfo {
	cfgServices := m.configMgr.GetServices()
	var result []*ServiceInfo
	processed := make(map[string]bool)

	// 先合并配置中的服务
	for _, cfgSvc := range cfgServices {
		// 查找匹配的 demo 服务
		var demoInfo *ServiceInfo
		for _, demo := range demoServices {
			if demo.Name == cfgSvc.Name {
				demoInfo = demo
				break
			}
		}
		merged := mergeServiceConfig(cfgSvc, demoInfo)
		result = append(result, merged)
		processed[cfgSvc.Name] = true
	}

	// 再添加剩余的 demo 服务
	for _, demo := range demoServices {
		if !processed[demo.Name] {
			result = append(result, demo)
			processed[demo.Name] = true
		}
	}

	return result
}
