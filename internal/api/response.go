package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/LiteyukiStudio/devops/internal/config"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

func bindJSON(ctx *gin.Context, value any) bool {
	if err := ctx.ShouldBindJSON(value); err != nil {
		writeErrorCode(ctx, http.StatusBadRequest, "request.invalid_json", err.Error())
		return false
	}
	return true
}

func writeError(ctx *gin.Context, status int, message string) {
	writeErrorCode(ctx, status, defaultErrorCode(status), message)
}

func writeErrorKey(ctx *gin.Context, status int, language, key string) {
	ctx.JSON(status, gin.H{"code": key, "error": messageFor(language, key)})
}

func writeErrorCode(ctx *gin.Context, status int, code, detail string) {
	if code == "" {
		code = defaultErrorCode(status)
	}
	if config.RuntimeMode() == "development" {
		ctx.JSON(status, gin.H{"code": code, "error": detail, "detail": detail})
		return
	}
	ctx.JSON(status, gin.H{"code": code, "error": publicErrorMessage(status)})
}

func errorResponseMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()
		if len(ctx.Errors) == 0 || ctx.Writer.Written() {
			return
		}
		writeErrorCode(ctx, http.StatusInternalServerError, "internal_error", ctx.Errors.Last().Error())
	}
}

func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered any) {
		writeErrorCode(ctx, http.StatusInternalServerError, "internal_error", fmt.Sprint(recovered))
	})
}

func defaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "request.invalid"
	case http.StatusUnauthorized:
		return "auth.unauthorized"
	case http.StatusForbidden:
		return "auth.forbidden"
	case http.StatusNotFound:
		return "resource.not_found"
	case http.StatusConflict:
		return "resource.conflict"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusBadGateway:
		return "upstream.failed"
	default:
		if status >= 500 {
			return "internal_error"
		}
		return "request.failed"
	}
}

func publicErrorMessage(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "请求参数不正确"
	case http.StatusUnauthorized:
		return "请先登录"
	case http.StatusForbidden:
		return "没有权限执行该操作"
	case http.StatusNotFound:
		return "资源不存在"
	case http.StatusConflict:
		return "资源状态冲突"
	case http.StatusTooManyRequests:
		return "请求过于频繁，请稍后再试"
	case http.StatusBadGateway:
		return "上游服务调用失败，请稍后再试"
	default:
		if status >= 500 {
			return "服务暂时不可用，请稍后再试"
		}
		return "请求处理失败"
	}
}

func messageFor(language, key string) string {
	messages := localizedMessages[normalizeLanguage(language)]
	if message, ok := messages[key]; ok {
		return message
	}
	return localizedMessages["zh-CN"][key]
}

func requestLanguage(ctx *gin.Context) string {
	if strings.Contains(strings.ToLower(ctx.GetHeader("Accept-Language")), "en") {
		return "en-US"
	}
	return "zh-CN"
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func fallbackInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

func randomHex(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var localizedMessages = map[string]map[string]string{
	"zh-CN": {
		"auth.login.invalid":    "邮箱或密码不正确",
		"auth.session.missing":  "请先登录",
		"auth.session.expired":  "登录会话已过期，请重新登录",
		"auth.account.disabled": "账号不可用，请联系平台管理员",
		"config.admin.required": "只有平台管理员可以修改站点配置",
	},
	"en-US": {
		"auth.login.invalid":    "Email or password is incorrect",
		"auth.session.missing":  "Please sign in first",
		"auth.session.expired":  "Your session has expired. Please sign in again",
		"auth.account.disabled": "This account is unavailable. Contact a platform administrator",
		"config.admin.required": "Only platform administrators can update site settings",
	},
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
