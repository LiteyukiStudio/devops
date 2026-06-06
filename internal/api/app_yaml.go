package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

func (h *Handlers) ParseApplicationConfig(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin", "developer"); !ok {
		return
	}

	var input parseApplicationConfigInput
	if !bindJSON(ctx, &input) {
		return
	}
	if strings.TrimSpace(input.Content) == "" {
		writeError(ctx, http.StatusBadRequest, "请输入 .devops/app.yaml 内容")
		return
	}

	var config devopsAppConfig
	if err := yaml.Unmarshal([]byte(input.Content), &config); err != nil {
		writeError(ctx, http.StatusBadRequest, "app.yaml 解析失败")
		return
	}

	sourceType := fallback(config.Source.Type, "repository")
	response := applicationInput{
		Slug:           config.Slug,
		Name:           fallback(config.Name, config.Slug),
		SourceType:     sourceType,
		RepositoryURL:  config.Source.RepositoryURL,
		ImageReference: config.Source.Image,
		DockerfilePath: fallback(config.Build.Dockerfile, "Dockerfile"),
		BuildContext:   fallback(config.Build.Context, "."),
		ServicePort:    fallbackInt(config.Service.Port, 8080),
	}
	if response.Slug == "" || response.Name == "" {
		writeError(ctx, http.StatusBadRequest, "app.yaml 至少需要 name 或 slug")
		return
	}
	ctx.JSON(http.StatusOK, response)
}

type parseApplicationConfigInput struct {
	Content string `json:"content" binding:"required"`
}

type devopsAppConfig struct {
	Name   string `yaml:"name"`
	Slug   string `yaml:"slug"`
	Source struct {
		Type          string `yaml:"type"`
		RepositoryURL string `yaml:"repositoryUrl"`
		Image         string `yaml:"image"`
	} `yaml:"source"`
	Build struct {
		Dockerfile string `yaml:"dockerfile"`
		Context    string `yaml:"context"`
	} `yaml:"build"`
	Service struct {
		Port int `yaml:"port"`
	} `yaml:"service"`
}
