package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersIncludeCSPAndProductionHSTS(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ENABLE_HSTS", "")

	recorder := httptest.NewRecorder()
	router := gin.New()
	router.Use(securityHeaders())
	router.GET("/", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	csp := recorder.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "frame-ancestors 'self'") {
		t.Fatalf("unexpected CSP header: %q", csp)
	}
	if got := recorder.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=31536000") {
		t.Fatalf("unexpected HSTS header: %q", got)
	}
}

func TestSecurityHeadersDoNotEnableHSTSInDevelopmentByDefault(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_ENABLE_HSTS", "")

	recorder := httptest.NewRecorder()
	router := gin.New()
	router.Use(securityHeaders())
	router.GET("/", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := recorder.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS header = %q", got)
	}
}
