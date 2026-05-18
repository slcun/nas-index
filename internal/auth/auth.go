package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 表示一个 Web 用户
type User struct {
	Name         string `yaml:"name" json:"name"`
	PasswordHash string `yaml:"password" json:"-"`
}

// Session 表示一个用户会话
type Session struct {
	Token     string
	UserLogin string
	Expire    time.Time
}

// Auth 管理用户认证和会话
type Auth struct {
	users       map[string]*User
	sessions    map[string]*Session
	mu          sync.RWMutex
	sessionTTL  time.Duration
	rateLimiter *loginRateLimiter
}

// NewAuth 创建认证管理器
func NewAuth(users []User, sessionTTL time.Duration) *Auth {
	a := &Auth{
		users:       make(map[string]*User),
		sessions:    make(map[string]*Session),
		sessionTTL:  sessionTTL,
		rateLimiter: newLoginRateLimiter(10, 1*time.Minute),
	}
	for i := range users {
		a.users[users[i].Name] = &users[i]
	}
	return a
}

// HasUsers 检查是否存在用户
func (a *Auth) HasUsers() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.users) > 0
}

// Authenticate 验证用户名和密码，成功返回 session token
func (a *Auth) Authenticate(username, password, clientIP string) (token string, err error) {
	if a.rateLimiter.isLimited(clientIP) {
		return "", ErrRateLimited
	}

	a.mu.RLock()
	user, exists := a.users[username]
	a.mu.RUnlock()

	if !exists {
		a.rateLimiter.recordFail(clientIP)
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
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

// Register 注册新用户（仅在无用户时允许首次注册，或已登录管理员添加）
func (a *Auth) Register(username, password string) error {
	if len(password) < 6 {
		return ErrPasswordTooShort
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.users[username]; exists {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	a.users[username] = &User{
		Name:         username,
		PasswordHash: string(hash),
	}
	return nil
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

// GetUsers 返回所有用户列表（用于配置持久化）
func (a *Auth) GetUsers() []User {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := make([]User, 0, len(a.users))
	for _, u := range a.users {
		users = append(users, *u)
	}
	return users
}

// ChangePassword 修改用户密码
func (a *Auth) ChangePassword(username, oldPassword, newPassword string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.users[username]
	if !exists {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	if len(newPassword) < 6 {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return nil
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
