package config

// Service 定义了 NAS 服务的配置结构
type Service struct {
	Name        string `yaml:"name" json:"name"`
	DisplayName string `yaml:"display_name" json:"display_name"`
	Description string `yaml:"description" json:"description"`
	Port        int    `yaml:"port,omitempty" json:"port,omitempty"`
	Path        string `yaml:"path,omitempty" json:"path,omitempty"`
	Category    string `yaml:"category" json:"category"`
	Web         bool   `yaml:"web" json:"web"`
}

// UserConfig 定义了 Web 用户的配置结构
type UserConfig struct {
	Name         string `yaml:"name" json:"name"`
	PasswordHash string `yaml:"password" json:"-"`
}

// Config 定义了整个应用的配置结构
type Config struct {
	Services        []Service         `yaml:"services" json:"services"`
	ExcludeServices []string          `yaml:"exclude_services" json:"exclude_services"`
	Categories      map[string]string `yaml:"categories" json:"categories"`
	Users           []UserConfig      `yaml:"users,omitempty" json:"users,omitempty"`
	SessionTTL      int               `yaml:"session_ttl,omitempty" json:"session_ttl,omitempty"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Services: []Service{},
		ExcludeServices: []string{},
		Categories: map[string]string{
			"media":    "媒体中心",
			"files":    "文件管理",
			"download": "下载工具",
			"system":   "系统工具",
			"backup":   "备份与同步",
			"tools":    "效率与工具",
			"other":    "其他",
		},
	}
}
