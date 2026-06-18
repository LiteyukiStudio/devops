package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func registerStaticUI(router *gin.Engine, staticFS fs.FS) {
	if staticFS == nil {
		return
	}
	fileServer := http.FileServer(http.FS(staticFS))
	router.NoRoute(func(ctx *gin.Context) {
		if !staticUIRequestAllowed(ctx.Request) {
			ctx.Status(http.StatusNotFound)
			return
		}
		target := staticUIPath(ctx.Request.URL.Path)
		if staticUIFileExists(staticFS, target) {
			ctx.Request.URL.Path = "/" + target
			fileServer.ServeHTTP(ctx.Writer, ctx.Request)
			return
		}
		ctx.Request.URL.Path = "/index.html"
		fileServer.ServeHTTP(ctx.Writer, ctx.Request)
	})
}

func staticUIRequestAllowed(request *http.Request) bool {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return false
	}
	cleanPath := path.Clean("/" + strings.TrimSpace(request.URL.Path))
	return cleanPath != "/healthz" && !strings.HasPrefix(cleanPath, "/api/")
}

func staticUIPath(rawPath string) string {
	cleanPath := strings.TrimPrefix(path.Clean("/"+rawPath), "/")
	if cleanPath == "." || cleanPath == "" {
		return "index.html"
	}
	return cleanPath
}

func staticUIFileExists(files fs.FS, name string) bool {
	info, err := fs.Stat(files, name)
	return err == nil && !info.IsDir()
}
