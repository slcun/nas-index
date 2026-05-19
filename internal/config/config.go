package config

// Service 定义了 NAS 服务的配置结构
type Service struct {
	Name        string   `yaml:"name" json:"name"`
	DisplayName string   `yaml:"display_name" json:"display_name"`
	Description string   `yaml:"description" json:"description"`
	Port        int      `yaml:"port,omitempty" json:"port,omitempty"`
	Path        string   `yaml:"path,omitempty" json:"path,omitempty"`
	Web         bool     `yaml:"web" json:"web"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Group       string   `yaml:"group,omitempty" json:"group,omitempty"`
}

// Config 定义了整个应用的配置结构
type Config struct {
	Services        []Service `yaml:"services" json:"services"`
	ExcludeServices []string  `yaml:"exclude_services" json:"exclude_services"`
	SessionTTL      int       `yaml:"session_ttl,omitempty" json:"session_ttl,omitempty"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Services:        []Service{},
		ExcludeServices: []string{},
	}
}
