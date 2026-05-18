package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

const maxConnections = 10

var (
	activeConnections int
	connectionsMutex  sync.Mutex
)

// Terminal 代表一个终端会话
type Terminal struct {
	cmd     *exec.Cmd
	closed  bool
	closeMu sync.Mutex
}

// NewTerminal 创建一个新的终端
func NewTerminal() (*Terminal, error) {
	connectionsMutex.Lock()
	if activeConnections >= maxConnections {
		connectionsMutex.Unlock()
		return nil, fmt.Errorf("连接数已达上限")
	}
	activeConnections++
	connectionsMutex.Unlock()

	// 为了简化，我们在所有平台使用管道方式
	return newSimpleTerminal()
}

func newSimpleTerminal() (*Terminal, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd.exe"
		} else {
			shell = "/bin/bash"
		}
	}

	args := []string{}
	if runtime.GOOS != "windows" {
		args = append(args, "-l")
	}

	cmd := exec.Command(shell, args...)

	// 设置环境变量
	env := os.Environ()
	newEnv := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "WS_") {
			newEnv = append(newEnv, e)
		}
	}
	cmd.Env = newEnv

	return &Terminal{
		cmd: cmd,
	}, nil
}

// Read 从终端读取（为了兼容，这里暂时返回 EOF）
func (t *Terminal) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("read not supported in simple mode")
}

// Write 向终端写入（为了兼容）
func (t *Terminal) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("write not supported in simple mode")
}

// Resize 调整终端大小
func (t *Terminal) Resize(cols, rows int) error {
	return nil
}

// Close 关闭终端
func (t *Terminal) Close() error {
	t.closeMu.Lock()
	defer t.closeMu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	connectionsMutex.Lock()
	activeConnections--
	connectionsMutex.Unlock()

	return nil
}
