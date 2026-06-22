package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwaggerUIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerSwaggerUI(router)

	specReq := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	specRec := httptest.NewRecorder()
	router.ServeHTTP(specRec, specReq)
	if specRec.Code != http.StatusOK {
		t.Fatalf("openapi spec expected 200, got %d", specRec.Code)
	}
	if !strings.Contains(specRec.Body.String(), "openapi: 3.1.0") {
		t.Fatalf("openapi spec body does not look like the bundled spec")
	}

	uiReq := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	uiRec := httptest.NewRecorder()
	router.ServeHTTP(uiRec, uiReq)
	if uiRec.Code != http.StatusMovedPermanently {
		t.Fatalf("swagger ui redirect expected 301, got %d", uiRec.Code)
	}

	indexReq := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	indexRec := httptest.NewRecorder()
	router.ServeHTTP(indexRec, indexReq)
	if indexRec.Code != http.StatusOK {
		t.Fatalf("swagger ui index expected 200, got %d", indexRec.Code)
	}
	if !strings.Contains(indexRec.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("swagger ui body does not include SwaggerUIBundle")
	}
}
