package service

import "nas-manager/internal/config"

// ServiceInfo 包含服务的详细信息
type ServiceInfo struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	Port           int    `json:"port,omitempty"`
	Path           string `json:"path,omitempty"`
	Category       string `json:"category"`
	Web            bool   `json:"web"`
	ActiveState    string `json:"active_state"`
	UnitFileState  string `json:"unit_file_state"`
	Managed        bool   `json:"managed"`
}

// mergeServiceConfig 合并配置和自动发现的服务信息
func mergeServiceConfig(cfgService config.Service, autoInfo *ServiceInfo) *ServiceInfo {
	result := &ServiceInfo{
		Name:        cfgService.Name,
		DisplayName: cfgService.DisplayName,
		Description: cfgService.Description,
		Port:        cfgService.Port,
		Path:        cfgService.Path,
		Category:    cfgService.Category,
		Web:         cfgService.Web,
		Managed:     true,
	}

	if autoInfo != nil {
		result.ActiveState = autoInfo.ActiveState
		result.UnitFileState = autoInfo.UnitFileState
		if result.Description == "" {
			result.Description = autoInfo.Description
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
