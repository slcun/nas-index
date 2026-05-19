package service

import "nas-manager/internal/config"

// ServiceInfo 包含服务的详细信息
type ServiceInfo struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Description   string   `json:"description"`
	Port          int      `json:"port,omitempty"`
	Path          string   `json:"path,omitempty"`
	Web           bool     `json:"web"`
	ActiveState   string   `json:"active_state"`
	UnitFileState string   `json:"unit_file_state"`
	Managed       bool     `json:"managed"`
	Tags          []string `json:"tags,omitempty"`
	Group         string   `json:"group,omitempty"`
}

// SystemServiceInfo 系统中可用的服务信息（用于添加服务时选择）
type SystemServiceInfo struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	UnitFileState string `json:"unit_file_state"`
	Configured    bool   `json:"configured"`
}

// mergeServiceConfig 合并配置和自动发现的服务信息
func mergeServiceConfig(cfgService config.Service, autoInfo *ServiceInfo) *ServiceInfo {
	result := &ServiceInfo{
		Name:        cfgService.Name,
		DisplayName: cfgService.DisplayName,
		Description: cfgService.Description,
		Port:        cfgService.Port,
		Path:        cfgService.Path,
		Web:         cfgService.Web,
		Managed:     true,
		Tags:        cfgService.Tags,
		Group:       cfgService.Group,
	}

	if autoInfo != nil {
		result.ActiveState = autoInfo.ActiveState
		result.UnitFileState = autoInfo.UnitFileState
		if result.Description == "" {
			result.Description = autoInfo.Description
		}
		if len(result.Tags) == 0 && len(autoInfo.Tags) > 0 {
			result.Tags = autoInfo.Tags
		}
		if result.Group == "" && autoInfo.Group != "" {
			result.Group = autoInfo.Group
		}
	}

	if result.ActiveState == "" {
		result.ActiveState = "unknown"
	}
	if result.UnitFileState == "" {
		result.UnitFileState = "unknown"
	}

	return result
}
