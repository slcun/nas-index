package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// checkSystemd 检查系统是否使用 systemd
func checkSystemd() bool {
	cmd := exec.Command("systemctl", "--version")
	return cmd.Run() == nil
}

// listUnitFiles 列出所有服务单元文件
func listUnitFiles() ([]map[string]string, error) {
	cmd := exec.Command("systemctl", "list-unit-files", "--type=service", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var services []map[string]string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			services = append(services, map[string]string{
				"name":             parts[0],
				"unit_file_state":  parts[1],
			})
		}
	}
	return services, nil
}

// getServiceDetail 获取服务详情
func getServiceDetail(name string) (*ServiceInfo, error) {
	cmd := exec.Command("systemctl", "show", "-p", "Names,Description,LoadState,ActiveState,SubState,UnitFileState", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := &ServiceInfo{
		Name: name,
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		switch key {
		case "Names":
			info.Name = value
		case "Description":
			info.Description = value
		case "ActiveState":
			info.ActiveState = value
		case "UnitFileState":
			info.UnitFileState = value
		}
	}

	// 生成显示名称
	info.DisplayName = nameToDisplayName(name)

	// 猜测是否为 Web 服务
	info.Web = guessIsWeb(info)

	// 默认分类
	info.Category = "other"

	return info, nil
}

// getActiveState 获取服务活动状态
func getActiveState(name string) string {
	cmd := exec.Command("systemctl", "is-active", name)
	output, _ := cmd.Output()
	state := strings.TrimSpace(string(output))
	if state == "" {
		return "inactive"
	}
	return state
}

// startService 启动服务
func startService(name string) (bool, string) {
	cmd := exec.Command("sudo", "systemctl", "start", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = "操作失败"
		}
		return false, msg
	}
	return true, "操作成功"
}

// stopService 停止服务
func stopService(name string) (bool, string) {
	cmd := exec.Command("sudo", "systemctl", "stop", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = "操作失败"
		}
		return false, msg
	}
	return true, "操作成功"
}

// restartService 重启服务
func restartService(name string) (bool, string) {
	cmd := exec.Command("sudo", "systemctl", "restart", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = "操作失败"
		}
		return false, msg
	}
	return true, "操作成功"
}

// getLogs 获取服务日志
func getLogs(name string, lines int) string {
	cmd := exec.Command("systemctl", "--no-pager", "-n", fmt.Sprintf("%d", lines), "-u", name)
	output, _ := cmd.Output()
	return string(output)
}

// scanEtcSystemd 扫描 /etc/systemd/system 目录
func scanEtcSystemd(exclude map[string]bool) map[string]*ServiceInfo {
	result := make(map[string]*ServiceInfo)
	etcDir := "/etc/systemd/system"

	if _, err := os.Stat(etcDir); os.IsNotExist(err) {
		return result
	}

	entries, err := os.ReadDir(etcDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".service") {
			continue
		}
		if exclude[name] {
			continue
		}

		info, err := getServiceDetail(name)
		if err == nil {
			result[name] = info
		}
	}

	return result
}

// nameToDisplayName 将服务名转换为显示名称
func nameToDisplayName(name string) string {
	if strings.HasSuffix(name, ".service") {
		name = name[:len(name)-8]
	}
	name = strings.ReplaceAll(name, "@", "")
	name = strings.ReplaceAll(name, "-", " ")
	// 简单的首字母大写
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

// guessIsWeb 猜测是否为 Web 服务
func guessIsWeb(info *ServiceInfo) bool {
	webKeywords := []string{
		"web", "http", "api", "ui", "gui", "dashboard", "admin",
		"cms", "jenkins", "jellyfin", "sonarr", "radarr", "qbittorrent",
		"transmission", "cockpit", "syncthing", "grafana", "prometheus",
		"portainer", "nginx", "apache", "immich", "prowlarr",
	}
	nameLower := strings.ToLower(info.Name)
	descLower := strings.ToLower(info.Description)

	for _, kw := range webKeywords {
		if strings.Contains(nameLower, kw) || strings.Contains(descLower, kw) {
			return true
		}
	}
	return false
}
