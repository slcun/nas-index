package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"nas-manager/internal/api"
	"nas-manager/internal/auth"
	"nas-manager/internal/config"
	"nas-manager/internal/host"
	"nas-manager/internal/service"
	"nas-manager/internal/terminal"
)

//go:embed web/static web/templates
var webFS embed.FS

const (
	version = "1.0.0"
)

var (
	configPath  string
	port        int
	wsPort      int
	install     bool
	uninstall   bool
	showVersion bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.IntVar(&port, "port", 5000, "HTTP 服务端口")
	flag.IntVar(&wsPort, "ws-port", 5001, "WebSocket 服务端口")
	flag.BoolVar(&install, "install", false, "安装为系统服务")
	flag.BoolVar(&uninstall, "uninstall", false, "卸载系统服务")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("NAS Manager v%s\n", version)
		return
	}

	if install {
		doInstall()
		return
	}

	if uninstall {
		doUninstall()
		return
	}

	// 确定配置文件路径
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// 创建配置管理器
	configMgr := config.NewManager(configPath)

	// 创建认证管理器
	authUsers := configMgr.GetUsers()
	sessionTTLOne := configMgr.GetSessionTTL()
	authUsersConverted := make([]auth.User, len(authUsers))
	for i, u := range authUsers {
		authUsersConverted[i] = auth.User{Name: u.Name, PasswordHash: u.PasswordHash}
	}
	authMgr := auth.NewAuth(authUsersConverted, time.Duration(sessionTTLOne)*time.Hour)

	// 定期清理过期会话
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			authMgr.CleanExpiredSessions()
		}
	}()

	// 创建服务管理器
	serviceMgr := service.NewManager(configMgr)

	// 创建 API 处理
	handlers := api.NewHandlers(configMgr, serviceMgr)

	// 设置路由
	mux := api.SetupRouter(handlers, authMgr, webFS)

	// 添加 WebSocket 路由
	mux.HandleFunc("/ws", terminal.HandleWebSocket)

	// 应用认证中间件
	handler := api.SetupAuthMiddleware(mux, authMgr, webFS)

	// 获取主机信息
	info := host.GetInfo()

	// 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("NAS 管理面板启动中...\n")
	fmt.Printf("  主机名: %s\n", info.Hostname)
	fmt.Printf("  IP 地址: %s\n", info.IP)
	fmt.Printf("  访问地址: http://%s:%d\n", info.IP, port)
	fmt.Printf("  配置文件: %s\n", configPath)
	fmt.Printf("\n")

	// 启动 WebSocket 服务
	go func() {
		wsAddr := fmt.Sprintf(":%d", wsPort)
		log.Printf("WebSocket 服务启动在 %s", wsAddr)
		if err := http.ListenAndServe(wsAddr, http.HandlerFunc(terminal.HandleWebSocket)); err != nil {
			log.Printf("WebSocket 服务启动失败: %v", err)
		}
	}()

	log.Printf("HTTP 服务启动在 %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func getDefaultConfigPath() string {
	// 检查当前目录是否有配置文件
	if _, err := os.Stat("config.yaml"); err == nil {
		abs, _ := filepath.Abs("config.yaml")
		return abs
	}

	// 检查用户配置目录
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".nas-manager")
		os.MkdirAll(configDir, 0755)
		return filepath.Join(configDir, "config.yaml")
	}

	return "config.yaml"
}

func doInstall() {
	fmt.Println("安装功能将在未来版本中实现")
}

func doUninstall() {
	fmt.Println("卸载功能将在未来版本中实现")
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
