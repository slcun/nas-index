package service

import (
	"nas-manager/internal/config"
	"sort"
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

// ListServices 列出所有已配置的服务
func (m *Manager) ListServices() []*ServiceInfo {
	if !m.systemdAvail {
		return m.listDemoServices()
	}

	m.configMgr.ReloadIfChanged()

	cfgServices := m.configMgr.GetServices()

	var result []*ServiceInfo
	for _, cfgSvc := range cfgServices {
		autoInfo, _ := getServiceDetail(cfgSvc.Name)
		merged := mergeServiceConfig(cfgSvc, autoInfo)
		result = append(result, merged)
	}

	return result
}

// ListSystemServices 列出系统中所有可用的服务（用于添加服务时选择）
func (m *Manager) ListSystemServices() []SystemServiceInfo {
	if !m.systemdAvail {
		return listDemoSystemServices()
	}

	excludeServices := m.configMgr.GetExcludeServices()
	excludeMap := make(map[string]bool)
	for _, name := range excludeServices {
		excludeMap[name] = true
	}

	configuredMap := make(map[string]bool)
	for _, svc := range m.configMgr.GetServices() {
		configuredMap[svc.Name] = true
	}

	autoServices := make(map[string]*ServiceInfo)

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

	etcServices := scanEtcSystemd(excludeMap)
	for name, info := range etcServices {
		if _, exists := autoServices[name]; !exists {
			autoServices[name] = info
		}
	}

	result := make([]SystemServiceInfo, 0, len(autoServices))
	for name, info := range autoServices {
		result = append(result, SystemServiceInfo{
			Name:          name,
			Description:   info.Description,
			UnitFileState: info.UnitFileState,
			Configured:    configuredMap[name],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Configured != result[j].Configured {
			return !result[i].Configured
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// GetService 获取单个服务
func (m *Manager) GetService(name string) *ServiceInfo {
	services := m.ListServices()
	for _, svc := range services {
		if svc.Name == name {
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
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"媒体", "流媒体"},
		Group:         "家庭娱乐",
	},
	{
		Name:          "sonarr.service",
		DisplayName:   "Sonarr",
		Description:   "电视节目管理",
		Port:          8989,
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"媒体", "PVR"},
		Group:         "家庭娱乐",
	},
	{
		Name:          "radarr.service",
		DisplayName:   "Radarr",
		Description:   "电影管理",
		Port:          7878,
		ActiveState:   "inactive",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"媒体", "PVR"},
		Group:         "家庭娱乐",
	},
	{
		Name:          "qbittorrent.service",
		DisplayName:   "qBittorrent",
		Description:   "BT 下载工具",
		Port:          8080,
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"下载", "BT"},
		Group:         "下载管理",
	},
	{
		Name:          "cockpit.service",
		DisplayName:   "Cockpit",
		Description:   "Web 系统管理",
		Port:          9090,
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"系统", "管理"},
	},
	{
		Name:          "docker.service",
		DisplayName:   "Docker",
		Description:   "容器引擎",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           false,
		Managed:       true,
		Tags:          []string{"系统", "容器"},
	},
	{
		Name:          "syncthing.service",
		DisplayName:   "Syncthing",
		Description:   "文件同步",
		Port:          8384,
		ActiveState:   "inactive",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"同步", "备份"},
	},
	{
		Name:          "transmission.service",
		DisplayName:   "Transmission",
		Description:   "BT 下载工具",
		Port:          9091,
		ActiveState:   "inactive",
		UnitFileState: "disabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"下载", "BT"},
		Group:         "下载管理",
	},
	{
		Name:          "prowlarr.service",
		DisplayName:   "Prowlarr",
		Description:   "索引器管理",
		Port:          9696,
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"媒体", "索引"},
		Group:         "家庭娱乐",
	},
	{
		Name:          "immich.service",
		DisplayName:   "Immich",
		Description:   "照片管理",
		Port:          2283,
		Path:          "/photos",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"媒体", "照片"},
		Group:         "家庭娱乐",
	},
	{
		Name:          "nginx.service",
		DisplayName:   "Nginx 文件管理器",
		Description:   "Web 文件管理",
		Port:          5001,
		Path:          "/file",
		ActiveState:   "active",
		UnitFileState: "enabled",
		Web:           true,
		Managed:       true,
		Tags:          []string{"文件", "Web"},
	},
}

// listDemoSystemServices 返回 demo 模式下的系统服务列表
func listDemoSystemServices() []SystemServiceInfo {
	configuredMap := make(map[string]bool)
	for _, svc := range demoServices {
		configuredMap[svc.Name] = true
	}

	extra := []SystemServiceInfo{
		{Name: "apache2.service", Description: "Apache HTTP Server", UnitFileState: "disabled", Configured: false},
		{Name: "mysql.service", Description: "MySQL Database", UnitFileState: "disabled", Configured: false},
		{Name: "postgresql.service", Description: "PostgreSQL Database", UnitFileState: "disabled", Configured: false},
		{Name: "redis.service", Description: "Redis Server", UnitFileState: "disabled", Configured: false},
		{Name: "samba.service", Description: "Samba File Sharing", UnitFileState: "disabled", Configured: false},
		{Name: "nfs-server.service", Description: "NFS Server", UnitFileState: "disabled", Configured: false},
		{Name: "homeassistant.service", Description: "Home Assistant", UnitFileState: "disabled", Configured: false},
		{Name: "plexmediaserver.service", Description: "Plex Media Server", UnitFileState: "disabled", Configured: false},
	}

	result := make([]SystemServiceInfo, 0, len(demoServices)+len(extra))
	for _, svc := range demoServices {
		result = append(result, SystemServiceInfo{
			Name:          svc.Name,
			Description:   svc.Description,
			UnitFileState: svc.UnitFileState,
			Configured:    true,
		})
	}
	result = append(result, extra...)

	return result
}

func (m *Manager) listDemoServices() []*ServiceInfo {
	cfgServices := m.configMgr.GetServices()
	if len(cfgServices) == 0 {
		return demoServices
	}

	var result []*ServiceInfo
	processed := make(map[string]bool)

	for _, cfgSvc := range cfgServices {
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

	return result
}
