package host

import (
	"net"
	"os"
)

// Info 包含主机信息
type Info struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

// GetInfo 获取主机信息
func GetInfo() *Info {
	hostname, _ := os.Hostname()

	return &Info{
		Hostname: hostname,
		IP:       getLocalIP(),
	}
}

func getLocalIP() string {
	// 尝试连接到一个外部地址来获取本地 IP
	conn, err := net.Dial("udp", "10.255.255.255:1")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
