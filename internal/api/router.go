package api

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"nas-manager/internal/auth"
)

// SetupRouter 设置路由
func SetupRouter(handlers *Handlers, authMgr *auth.Auth, webFS embed.FS) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/login", authMgr.HandleLoginPage(webFS))
	mux.HandleFunc("/api/auth/login", authMgr.HandleLogin())
	mux.HandleFunc("/api/auth/register", authMgr.HandleRegister())
	mux.HandleFunc("/api/auth/check", authMgr.HandleCheckAuth())
	mux.HandleFunc("/api/auth/change-password", authMgr.HandleChangePassword())
	mux.HandleFunc("/logout", authMgr.HandleLogout())

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(func() fs.FS {
		sub, _ := fs.Sub(webFS, "web/static")
		return sub
	}()))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, webFS, "web/templates/index.html")
	})

	mux.HandleFunc("/terminal", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, webFS, "web/templates/terminal.html")
	})

	mux.HandleFunc("GET /api/services", handlers.GetServices)
	mux.HandleFunc("GET /api/services/{name}", handlers.GetService)
	mux.HandleFunc("POST /api/services/{name}/start", handlers.StartService)
	mux.HandleFunc("POST /api/services/{name}/stop", handlers.StopService)
	mux.HandleFunc("POST /api/services/{name}/restart", handlers.RestartService)
	mux.HandleFunc("GET /api/services/{name}/logs", handlers.GetServiceLogs)
	mux.HandleFunc("GET /api/host/info", handlers.GetHostInfo)
	mux.HandleFunc("GET /api/config", handlers.GetConfig)
	mux.HandleFunc("PUT /api/config", handlers.UpdateConfig)

	return mux
}

// SetupAuthMiddleware 为路由添加认证中间件
func SetupAuthMiddleware(mux *http.ServeMux, authMgr *auth.Auth, webFS embed.FS) http.Handler {
	publicPaths := []string{
		"/login",
		"/api/auth/login",
		"/api/auth/register",
		"/static/",
		"/ws",
	}

	return authMgr.Middleware(webFS, publicPaths)(mux)
}

func renderTemplate(w http.ResponseWriter, fs embed.FS, path string) {
	tmpl, err := template.ParseFS(fs, path)
	if err != nil {
		http.Error(w, "模板加载失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, nil)
}
