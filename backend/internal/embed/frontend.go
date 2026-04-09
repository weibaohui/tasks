package embed

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed ui/dist/*
var embeddedFiles embed.FS

// GetFrontendFS 获取前端文件系统（用于嵌入）
func GetFrontendFS() fs.FS {
	return embeddedFiles
}

// SetupFrontendRoutes 设置前端静态文件路由
// 在 Gin Engine 上注册前端 SPA 路由
func SetupFrontendRoutes(engine *gin.Engine) {
	frontendFS := GetFrontendFS()

	// 创建子文件系统，去掉 ui/dist 前缀
	distFS, err := fs.Sub(frontendFS, "ui/dist")
	if err != nil {
		// 如果嵌入失败，直接返回
		return
	}

	// 创建文件服务器
	fileServer := http.FileServer(http.FS(distFS))

	// 使用 NoRoute 处理所有未匹配的路由（SPA fallback）
	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路由和 WebSocket 路由不走前端
		if path == "/api" || strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// 尝试打开请求的文件
		file, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// 文件不存在，返回 index.html（SPA 路由）
			indexHTML, err := fs.ReadFile(frontendFS, "ui/dist/index.html")
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
			return
		}
		file.Close()

		// 文件存在，使用文件服务器提供
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
