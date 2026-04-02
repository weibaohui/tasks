package embed

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed ui/dist/*
var embeddedFiles embed.FS

// GetFrontendFS 获取前端文件系统（用于嵌入）
func GetFrontendFS() fs.FS {
	return embeddedFiles
}

// SetupFrontendRoutes 设置前端静态文件路由
// 返回一个处理前端路由的 http.Handler，用于在 API 路由之后注册
func SetupFrontendRoutes() http.Handler {
	frontendFS := GetFrontendFS()

	// 创建子文件系统，去掉 ui/dist 前缀
	distFS, err := fs.Sub(frontendFS, "ui/dist")
	if err != nil {
		// 如果嵌入失败，返回空处理器
		return http.NotFoundHandler()
	}

	// 创建文件服务器
	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 尝试打开请求的文件
		file, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// 文件不存在，返回 index.html（SPA 路由）
			indexHTML, err := fs.ReadFile(frontendFS, "ui/dist/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
			return
		}
		file.Close()

		// 文件存在，使用文件服务器提供
		fileServer.ServeHTTP(w, r)
	})
}
