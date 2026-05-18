package auth

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"
)

const (
	cookieName = "nas_session"
)

// Middleware 返回认证中间件
func (a *Auth) Middleware(webFS embed.FS, publicPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			for _, prefix := range publicPaths {
				if strings.HasPrefix(path, prefix) || path == prefix {
					next.ServeHTTP(w, r)
					return
				}
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil {
				a.handleUnauthorized(w, r, webFS)
				return
			}

			username, ok := a.ValidateSession(cookie.Value)
			if !ok {
				http.SetCookie(w, &http.Cookie{
					Name:   cookieName,
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				})
				a.handleUnauthorized(w, r, webFS)
				return
			}

			r.Header.Set("X-User", username)
			next.ServeHTTP(w, r)
		})
	}
}

// handleUnauthorized 处理未认证请求
func (a *Auth) handleUnauthorized(w http.ResponseWriter, r *http.Request, webFS embed.FS) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"未授权访问，请先登录"}`))
		return
	}

	if !a.HasUsers() {
		http.Redirect(w, r, "/login?setup=1", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// HandleLoginPage 渲染登录页面
func (a *Auth) HandleLoginPage(webFS embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isSetup := r.URL.Query().Get("setup") == "1"

		tmpl, err := template.ParseFS(webFS, "web/templates/login.html")
		if err != nil {
			log.Printf("模板解析失败: %v", err)
			http.Error(w, "模板加载失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, map[string]interface{}{
			"IsSetup":  isSetup,
			"HasUsers": a.HasUsers(),
		})
	}
}

// HandleLogin 处理登录请求
func (a *Auth) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		clientIP := r.RemoteAddr

		token, err := a.Authenticate(username, password, clientIP)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(a.sessionTTL.Seconds()),
		})

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true}`))
	}
}

// HandleRegister 处理注册请求
func (a *Auth) HandleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		if a.HasUsers() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"已存在用户，不允许注册"}`))
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		if err := a.Register(username, password); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		token, err := a.Authenticate(username, password, r.RemoteAddr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"注册成功但自动登录失败"}`))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(a.sessionTTL.Seconds()),
		})

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true}`))
	}
}

// HandleLogout 处理登出请求
func (a *Auth) HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(cookieName); err == nil {
			a.Logout(cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:   cookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// HandleChangePassword 处理修改密码请求
func (a *Auth) HandleChangePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
			return
		}

		username := r.Header.Get("X-User")
		if username == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"未授权"}`))
			return
		}

		oldPassword := r.FormValue("old_password")
		newPassword := r.FormValue("new_password")

		if err := a.ChangePassword(username, oldPassword, newPassword); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"` + err.Error() + `"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"message":"密码修改成功"}`))
	}
}

// HandleCheckAuth 检查当前认证状态
func (a *Auth) HandleCheckAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get("X-User")
		w.Header().Set("Content-Type", "application/json")
		if username != "" {
			w.Write([]byte(`{"authenticated":true,"username":"` + username + `"}`))
		} else {
			w.Write([]byte(`{"authenticated":false}`))
		}
	}
}
