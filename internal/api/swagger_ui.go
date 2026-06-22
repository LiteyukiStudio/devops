package api

import (
	"net/http"

	"github.com/LiteyukiStudio/devops/openapi"
	"github.com/gin-gonic/gin"
	"github.com/swaggest/swgui/v5emb"
)

func registerSwaggerUI(router *gin.Engine) {
	router.GET("/openapi.yaml", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", openapi.SpecYAML)
	})
	router.GET("/swagger", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusMovedPermanently, "/swagger/")
	})
	router.Any("/swagger/*any", gin.WrapH(v5emb.New("Liteyuki DevOps API", "/openapi.yaml", "/swagger/")))
}
