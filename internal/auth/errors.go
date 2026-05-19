package auth

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrRateLimited        = errors.New("登录尝试过于频繁，请稍后再试")
)

// loginAttempt 记录某个 IP 的登录失败次数
type loginAttempt struct {
	count    int
	lastFail time.Time
}

// loginRateLimiter 登录限流器
type loginRateLimiter struct {
	mu          sync.RWMutex
	attempts    map[string]*loginAttempt
	maxFails    int
	banDuration time.Duration
}

// newLoginRateLimiter 创建登录限流器
func newLoginRateLimiter(maxFails int, banDuration time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		attempts:    make(map[string]*loginAttempt),
		maxFails:    maxFails,
		banDuration: banDuration,
	}
}

// isLimited 检查 IP 是否被限流
func (r *loginRateLimiter) isLimited(ip string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	attempt, exists := r.attempts[ip]
	if !exists {
		return false
	}

	if attempt.count >= r.maxFails {
		if time.Since(attempt.lastFail) < r.banDuration {
			return true
		}
	}

	return false
}

// recordFail 记录一次登录失败
func (r *loginRateLimiter) recordFail(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	attempt, exists := r.attempts[ip]
	if !exists {
		attempt = &loginAttempt{}
		r.attempts[ip] = attempt
	}

	attempt.count++
	attempt.lastFail = time.Now()
}

// reset 重置 IP 的失败计数
func (r *loginRateLimiter) reset(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.attempts, ip)
}
