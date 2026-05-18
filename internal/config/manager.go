package config

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager 负责配置的加载、保存和热重载
type Manager struct {
	configPath string
	config     *Config
	modTime    time.Time
	mu         sync.RWMutex
}

// NewManager 创建一个新的配置管理器
func NewManager(configPath string) *Manager {
	m := &Manager{
		configPath: configPath,
		config:     DefaultConfig(),
	}
	m.Load()
	return m
}

// Load 从文件加载配置
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// 配置文件不存在，使用默认配置并保存
		if err := m.saveLocked(); err != nil {
			return err
		}
		return nil
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	m.config = &cfg
	info, _ := os.Stat(m.configPath)
	if info != nil {
		m.modTime = info.ModTime()
	}

	return nil
}

// Save 保存配置到文件
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return err
	}

	info, _ := os.Stat(m.configPath)
	if info != nil {
		m.modTime = info.ModTime()
	}

	return nil
}

// ReloadIfChanged 如果配置文件有变化则重新加载
func (m *Manager) ReloadIfChanged() error {
	m.mu.RLock()
	modTime := m.modTime
	m.mu.RUnlock()

	info, err := os.Stat(m.configPath)
	if err != nil {
		return nil
	}

	if info.ModTime().After(modTime) {
		return m.Load()
	}

	return nil
}

// Get 获取当前配置
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// 返回副本，防止外部修改
	cfg := *m.config
	servicesCopy := make([]Service, len(cfg.Services))
	copy(servicesCopy, cfg.Services)
	cfg.Services = servicesCopy

	categoriesCopy := make(map[string]string)
	for k, v := range cfg.Categories {
		categoriesCopy[k] = v
	}
	cfg.Categories = categoriesCopy

	return &cfg
}

// Update 更新配置
func (m *Manager) Update(cfg *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg
}

// GetServices 获取服务列表
func (m *Manager) GetServices() []Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Services
}

// GetExcludeServices 获取排除服务列表
func (m *Manager) GetExcludeServices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.ExcludeServices
}

// GetCategories 获取分类映射
func (m *Manager) GetCategories() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Categories
}

// GetUsers 获取用户列表
func (m *Manager) GetUsers() []UserConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Users
}

// GetSessionTTL 获取会话过期时间（小时）
func (m *Manager) GetSessionTTL() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config.SessionTTL <= 0 {
		return 72
	}
	return m.config.SessionTTL
}
