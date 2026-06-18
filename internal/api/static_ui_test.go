package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestStaticUIServesIndexWithoutRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	files := fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><title>Liteyuki DevOps</title>"),
		},
		"assets/app.js": {
			Data: []byte("console.log('ok')"),
		},
	}
	router := gin.New()
	registerStaticUI(router, files)

	for _, path := range []string{"/", "/index.html", "/projects/prj_1/apps/app_1"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200, got %d", path, rec.Code)
		}
		if location := rec.Header().Get("Location"); location != "" {
			t.Fatalf("GET %s should not redirect, got Location %q", path, location)
		}
		if !strings.Contains(rec.Body.String(), "Liteyuki DevOps") {
			t.Fatalf("GET %s expected index body, got %q", path, rec.Body.String())
		}
	}
}

func TestStaticUIServesAssetsAndSkipsAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	files := fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><title>Liteyuki DevOps</title>"),
		},
		"assets/app.js": {
			Data: []byte("console.log('ok')"),
		},
	}
	router := gin.New()
	registerStaticUI(router, files)

	assetReq := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	assetRec := httptest.NewRecorder()
	router.ServeHTTP(assetRec, assetReq)
	if assetRec.Code != http.StatusOK {
		t.Fatalf("asset expected 200, got %d", assetRec.Code)
	}
	if !strings.Contains(assetRec.Body.String(), "console.log") {
		t.Fatalf("asset expected file body, got %q", assetRec.Body.String())
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	apiRec := httptest.NewRecorder()
	router.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusNotFound {
		t.Fatalf("api route expected 404, got %d", apiRec.Code)
	}
}
