package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Session 表示一个用户会话
type Session struct {
	Token     string
	UserLogin string
	Expire    time.Time
}

// Auth 管理用户认证和会话
type Auth struct {
	sessions     map[string]*Session
	mu           sync.RWMutex
	sessionTTL   time.Duration
	rateLimiter  *loginRateLimiter
	systemdAvail bool
}

// NewAuth 创建认证管理器
func NewAuth(sessionTTL time.Duration, systemdAvail bool) *Auth {
	return &Auth{
		sessions:     make(map[string]*Session),
		sessionTTL:   sessionTTL,
		rateLimiter:  newLoginRateLimiter(10, 1*time.Minute),
		systemdAvail: systemdAvail,
	}
}

// Authenticate 验证系统用户名和密码，成功返回 session token
func (a *Auth) Authenticate(username, password, clientIP string) (token string, err error) {
	if a.rateLimiter.isLimited(clientIP) {
		return "", ErrRateLimited
	}

	if !authenticateSystemUser(username, password, a.systemdAvail) {
		a.rateLimiter.recordFail(clientIP)
		return "", ErrInvalidCredentials
	}

	a.rateLimiter.reset(clientIP)

	token = generateToken()
	session := &Session{
		Token:     token,
		UserLogin: username,
		Expire:    time.Now().Add(a.sessionTTL),
	}

	a.mu.Lock()
	a.sessions[token] = session
	a.mu.Unlock()

	return token, nil
}

// authenticateSystemUser 验证 Linux 系统用户密码
// 优先使用 Python PAM 模块验证，回退到 su + PTY 方式
func authenticateSystemUser(username, password string, systemdAvail bool) bool {
	if !systemdAvail {
		return username != "" && password != ""
	}

	if !userExists(username) {
		return false
	}

	if authViaPamPython(username, password) {
		return true
	}

	return authViaSuPty(username, password)
}

// authViaPamPython 使用 Python3 调用 PAM 模块验证密码
func authViaPamPython(username, password string) bool {
	script := `
import pam, sys
p = pam.pam()
try:
    result = p.authenticate(sys.argv[1], sys.argv[2])
    sys.exit(0 if result else 1)
except Exception:
    sys.exit(2)
`
	cmd := exec.Command("python3", "-c", script, username, password)
	err := cmd.Run()
	return err == nil
}

// killProcess 安全终止进程
func killProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}
}

// waitWithTimeout 等待子进程结束，超时则终止
func waitWithTimeout(done <-chan error, timeout time.Duration) error {
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("操作超时")
	}
}

// authViaSuPty 通过 su + PTY 方式验证密码（回退方案）
func authViaSuPty(username, password string) bool {
	cmd := exec.Command("su", "-l", username, "-c", "exit 0")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return false
	}
	defer ptmx.Close()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	buf := make([]byte, 8192)
	ptmx.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _ := ptmx.Read(buf)

	output := string(buf[:n])

	if !containsPasswordPrompt(output) {
		err := waitWithTimeout(done, 3*time.Second)
		if err != nil {
			killProcess(cmd)
		}
		return err == nil
	}

	ptmx.Write([]byte(password + "\n"))

	err = waitWithTimeout(done, 5*time.Second)
	if err != nil {
		killProcess(cmd)
	}
	return err == nil
}

// userExists 检查系统用户是否存在
func userExists(username string) bool {
	cmd := exec.Command("id", username)
	return cmd.Run() == nil
}

// containsPasswordPrompt 检查输出是否包含密码提示
func containsPasswordPrompt(output string) bool {
	prompts := []string{"Password:", "密码："}
	for _, p := range prompts {
		if strings.Contains(output, p) {
			return true
		}
	}
	return false
}

// ValidateSession 验证 session token，返回用户名
func (a *Auth) ValidateSession(token string) (username string, ok bool) {
	a.mu.RLock()
	session, exists := a.sessions[token]
	a.mu.RUnlock()

	if !exists {
		return "", false
	}

	if time.Now().After(session.Expire) {
		a.mu.Lock()
		delete(a.sessions, token)
		a.mu.Unlock()
		return "", false
	}

	return session.UserLogin, true
}

// Logout 删除会话
func (a *Auth) Logout(token string) {
	a.mu.Lock()
	delete(a.sessions, token)
	a.mu.Unlock()
}

// CleanExpiredSessions 清理过期会话
func (a *Auth) CleanExpiredSessions() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for token, session := range a.sessions {
		if now.After(session.Expire) {
			delete(a.sessions, token)
		}
	}
}

// generateToken 生成随机 session token
func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
