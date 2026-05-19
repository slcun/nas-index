package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"nas-manager/internal/config"
	"nas-manager/internal/host"
	"nas-manager/internal/service"
)

// Handlers 包含所有 API 处理函数
type Handlers struct {
	configMgr  *config.Manager
	serviceMgr *service.Manager
}

// NewHandlers 创建一个新的 API 处理对象
func NewHandlers(configMgr *config.Manager, serviceMgr *service.Manager) *Handlers {
	return &Handlers{
		configMgr:  configMgr,
		serviceMgr: serviceMgr,
	}
}

// GetServices 获取所有已配置的服务
func (h *Handlers) GetServices(w http.ResponseWriter, r *http.Request) {
	services := h.serviceMgr.ListServices()

	allTags := make(map[string]bool)
	allGroups := make(map[string]bool)
	for _, s := range services {
		for _, tag := range s.Tags {
			allTags[tag] = true
		}
		if s.Group != "" {
			allGroups[s.Group] = true
		}
	}

	tagsList := make([]string, 0, len(allTags))
	for tag := range allTags {
		tagsList = append(tagsList, tag)
	}

	groupsList := make([]string, 0, len(allGroups))
	for group := range allGroups {
		groupsList = append(groupsList, group)
	}

	type response struct {
		Services []*service.ServiceInfo `json:"services"`
		Tags     []string               `json:"tags"`
		Groups   []string               `json:"groups"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{
		Services: services,
		Tags:     tagsList,
		Groups:   groupsList,
	})
}

// GetSystemServices 获取系统中所有可用的服务（用于添加服务时选择）
func (h *Handlers) GetSystemServices(w http.ResponseWriter, r *http.Request) {
	services := h.serviceMgr.ListSystemServices()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": services,
	})
}

// GetService 获取单个服务
func (h *Handlers) GetService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	svc := h.serviceMgr.GetService(name)
	if svc == nil {
		http.Error(w, "服务未找到", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(svc)
}

// AddService 添加一个服务配置
func (h *Handlers) AddService(w http.ResponseWriter, r *http.Request) {
	var svc config.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	if svc.Name == "" {
		http.Error(w, "服务名称不能为空", http.StatusBadRequest)
		return
	}

	if err := h.configMgr.AddService(svc); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "服务已添加",
	})
}

// UpdateService 更新一个服务配置
func (h *Handlers) UpdateService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	var svc config.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	if err := h.configMgr.UpdateService(name, svc); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "服务已更新",
	})
}

// DeleteService 删除一个服务配置
func (h *Handlers) DeleteService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	if err := h.configMgr.RemoveService(name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "服务已删除",
	})
}

// StartService 启动服务
func (h *Handlers) StartService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	success, message := h.serviceMgr.StartService(name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": message,
	})
}

// StopService 停止服务
func (h *Handlers) StopService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	success, message := h.serviceMgr.StopService(name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": message,
	})
}

// RestartService 重启服务
func (h *Handlers) RestartService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	success, message := h.serviceMgr.RestartService(name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": message,
	})
}

// GetServiceLogs 获取服务日志
func (h *Handlers) GetServiceLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "缺少服务名称", http.StatusBadRequest)
		return
	}

	linesStr := r.URL.Query().Get("lines")
	lines := 50
	if linesStr != "" {
		if l, err := strconv.Atoi(linesStr); err == nil {
			lines = l
		}
	}

	logs := h.serviceMgr.GetLogs(name, lines)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name": name,
		"logs": logs,
	})
}

// GetHostInfo 获取主机信息
func (h *Handlers) GetHostInfo(w http.ResponseWriter, r *http.Request) {
	info := host.GetInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// GetConfig 获取配置
func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.configMgr.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateConfig 更新配置
func (h *Handlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "无效的配置数据", http.StatusBadRequest)
		return
	}

	h.configMgr.Update(&cfg)
	if err := h.configMgr.Save(); err != nil {
		http.Error(w, "保存配置失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "配置已更新",
	})
}
