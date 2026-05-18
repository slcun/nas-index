package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

const maxConnections = 10

var (
	activeConnections int
	connectionsMutex  sync.Mutex
)

// Terminal 代表一个真实的 PTY 终端会话
type Terminal struct {
	cmd    *exec.Cmd
	ptmx   *os.File
	closed bool
	mu     sync.Mutex
}

// NewTerminal 创建一个新的 PTY 终端
func NewTerminal(cols, rows uint16) (*Terminal, error) {
	connectionsMutex.Lock()
	if activeConnections >= maxConnections {
		connectionsMutex.Unlock()
		return nil, fmt.Errorf("连接数已达上限 (%d)", maxConnections)
	}
	activeConnections++
	connectionsMutex.Unlock()

	if runtime.GOOS == "windows" {
		t, err := newSimpleTerminal()
		if err != nil {
			connectionsMutex.Lock()
			activeConnections--
			connectionsMutex.Unlock()
		}
		return t, err
	}

	return newPtyTerminal(cols, rows)
}

// newPtyTerminal 在 Linux/macOS 上创建真实 PTY 终端
func newPtyTerminal(cols, rows uint16) (*Terminal, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell, "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
	if err != nil {
		connectionsMutex.Lock()
		activeConnections--
		connectionsMutex.Unlock()
		return nil, fmt.Errorf("启动 PTY 失败: %w", err)
	}

	return &Terminal{
		cmd:  cmd,
		ptmx: ptmx,
	}, nil
}

// newSimpleTerminal 在 Windows 上创建简单终端（回退方案）
func newSimpleTerminal() (*Terminal, error) {
	shell := "cmd.exe"
	cmd := exec.Command(shell)
	return &Terminal{
		cmd: cmd,
	}, nil
}

// Read 从 PTY 读取输出
func (t *Terminal) Read(p []byte) (int, error) {
	if t.ptmx == nil {
		return 0, fmt.Errorf("PTY 不可用")
	}
	return t.ptmx.Read(p)
}

// Write 向 PTY 写入输入
func (t *Terminal) Write(p []byte) (int, error) {
	if t.ptmx == nil {
		return 0, fmt.Errorf("PTY 不可用")
	}
	return t.ptmx.Write(p)
}

// Resize 调整终端大小
func (t *Terminal) Resize(cols, rows int) error {
	if t.ptmx == nil {
		return nil
	}
	return pty.Setsize(t.ptmx, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// Close 关闭终端
func (t *Terminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	if t.ptmx != nil {
		t.ptmx.Close()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Signal(syscall.SIGTERM)
		t.cmd.Wait()
	}

	connectionsMutex.Lock()
	activeConnections--
	connectionsMutex.Unlock()

	return nil
}
