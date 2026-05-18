package api

import (
	"embed"
	"html/template"
	"net/http"
)

// SetupRouter 设置路由
func SetupRouter(handlers *Handlers, webFS embed.FS) *http.ServeMux {
	mux := http.NewServeMux()

	// 静态文件服务
	fs := http.FileServer(http.FS(webFS))
	mux.Handle("/static/", fs)

	// HTML 模板
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, webFS, "web/templates/index.html")
	})

	mux.HandleFunc("/terminal", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, webFS, "web/templates/terminal.html")
	})

	// API 路由
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

func renderTemplate(w http.ResponseWriter, fs embed.FS, path string) {
	tmpl, err := template.ParseFS(fs, path)
	if err != nil {
		http.Error(w, "模板加载失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, nil)
}
