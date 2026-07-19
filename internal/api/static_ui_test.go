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
			Data: []byte("<!doctype html><title>Luna DevOps</title>"),
		},
		"assets/app.js": {
			Data: []byte("console.log('ok')"),
		},
		"luna-devops-logo.svg": {
			Data: []byte("<svg></svg>"),
		},
	}
	router := gin.New()
	registerStaticUI(router, files, nil)

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
		if !strings.Contains(rec.Body.String(), "Luna DevOps") {
			t.Fatalf("GET %s expected index body, got %q", path, rec.Body.String())
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-cache, must-revalidate" {
			t.Fatalf("GET %s Cache-Control = %q", path, got)
		}
	}
}

func TestStaticUIServesAssetsAndSkipsAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	files := fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><title>Luna DevOps</title>"),
		},
		"assets/app.js": {
			Data: []byte("console.log('ok')"),
		},
		"luna-devops-logo.svg": {
			Data: []byte("<svg></svg>"),
		},
	}
	router := gin.New()
	registerStaticUI(router, files, nil)

	assetReq := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	assetRec := httptest.NewRecorder()
	router.ServeHTTP(assetRec, assetReq)
	if assetRec.Code != http.StatusOK {
		t.Fatalf("asset expected 200, got %d", assetRec.Code)
	}
	if !strings.Contains(assetRec.Body.String(), "console.log") {
		t.Fatalf("asset expected file body, got %q", assetRec.Body.String())
	}
	if got := assetRec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset Cache-Control = %q", got)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/luna-devops-logo.svg", nil)
	publicRec := httptest.NewRecorder()
	router.ServeHTTP(publicRec, publicReq)
	if publicRec.Code != http.StatusOK {
		t.Fatalf("public asset expected 200, got %d", publicRec.Code)
	}
	if got := publicRec.Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Fatalf("public asset Cache-Control = %q", got)
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	apiRec := httptest.NewRecorder()
	router.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusNotFound {
		t.Fatalf("api route expected 404, got %d", apiRec.Code)
	}
}

func TestStaticUIInjectsValidatedBrandThemeBeforeFirstPaint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	files := fstest.MapFS{
		"index.html": {
			Data: []byte(`<!doctype html><html data-brand-theme="__LUNA_DEVOPS_BRAND_THEME__"></html>`),
		},
	}

	for _, test := range []struct {
		name       string
		configured string
		want       string
	}{
		{name: "official preset", configured: "teal", want: `data-brand-theme="teal"`},
		{name: "invalid preset", configured: "url(javascript:bad)", want: `data-brand-theme="blue"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			registerStaticUI(router, files, func() string { return test.configured })
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

			if recorder.Code != http.StatusOK {
				t.Fatalf("GET / expected 200, got %d", recorder.Code)
			}
			if !strings.Contains(recorder.Body.String(), test.want) {
				t.Fatalf("index body %q does not contain %q", recorder.Body.String(), test.want)
			}
			if strings.Contains(recorder.Body.String(), brandThemeHTMLPlaceholder) {
				t.Fatalf("index body still contains theme placeholder: %q", recorder.Body.String())
			}
		})
	}
}
