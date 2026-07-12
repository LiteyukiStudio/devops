package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
)

func TestRequireInteractiveSessionRejectsBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/projects/prj/applications/app/deployment-targets/dplt/data-export", nil)
	ctx.Request.Header.Set("Authorization", "Bearer token-with-data-export-scope")

	if requireInteractiveSession(ctx) {
		t.Fatal("expected bearer token request to require an interactive session")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["code"] != "auth.interactive_session_required" {
		t.Fatalf("code = %v, want auth.interactive_session_required", response["code"])
	}
}

func TestAuthorizeDeploymentTargetDataExportRejectsBearerTokenBeforeAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/projects/prj/applications/app/deployment-targets/dplt/data-export/authorize", nil)
	ctx.Request.Header.Set("Authorization", "Bearer token-with-deployment-wildcard-scope")

	(&Handlers{}).AuthorizeDeploymentTargetDataExport(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["code"] != "auth.interactive_session_required" {
		t.Fatalf("code = %v, want auth.interactive_session_required", response["code"])
	}
}

func TestRequireInteractiveSessionAllowsCookieAuthenticationToContinue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/projects/prj/applications/app/deployment-targets/dplt/data-export", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sess_test"})

	if !requireInteractiveSession(ctx) {
		t.Fatal("expected cookie-authenticated request to continue to session validation")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected response status = %d", recorder.Code)
	}
}

func TestRequireInteractiveSessionRejectsMissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/projects/prj/applications/app/deployment-targets/dplt/data-export", nil)

	if requireInteractiveSession(ctx) {
		t.Fatal("expected request without a session cookie to be rejected")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["code"] != "auth.session.missing" {
		t.Fatalf("code = %v, want auth.session.missing", response["code"])
	}
}

func TestDataExportTicketIsBoundAndConsumedOnce(t *testing.T) {
	handlers := &Handlers{mode: "test"}
	authorization := testDataExportAuthorization()
	session := model.UserSession{ID: "ses_export", UserID: authorization.user.ID}
	ticket, expiresAt, err := handlers.issueDataExportTicket(context.Background(), authorization, session)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	if ticket == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("invalid ticket response: ticket=%q expiresAt=%v", ticket, expiresAt)
	}

	valid, err := handlers.consumeDataExportTicket(context.Background(), ticket, authorization, session)
	if err != nil || !valid {
		t.Fatalf("consume ticket: valid=%v err=%v", valid, err)
	}
	valid, err = handlers.consumeDataExportTicket(context.Background(), ticket, authorization, session)
	if err != nil || valid {
		t.Fatalf("consumed ticket was reusable: valid=%v err=%v", valid, err)
	}
}

func TestDataExportTicketRejectsDifferentBindingAndIsStillConsumed(t *testing.T) {
	handlers := &Handlers{mode: "test"}
	authorization := testDataExportAuthorization()
	session := model.UserSession{ID: "ses_export_binding", UserID: authorization.user.ID}
	ticket, _, err := handlers.issueDataExportTicket(context.Background(), authorization, session)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	otherTarget := authorization
	otherTarget.target.ID = "dplt_other"
	valid, err := handlers.consumeDataExportTicket(context.Background(), ticket, otherTarget, session)
	if err != nil || valid {
		t.Fatalf("ticket accepted a different target: valid=%v err=%v", valid, err)
	}
	valid, err = handlers.consumeDataExportTicket(context.Background(), ticket, authorization, session)
	if err != nil || valid {
		t.Fatalf("binding mismatch did not atomically consume ticket: valid=%v err=%v", valid, err)
	}
}

func TestDataExportTicketUsesRedisAtomicallyInProduction(t *testing.T) {
	server := miniredis.RunT(t)
	handlers := &Handlers{mode: "production", rateLimiter: newRateLimiter(server.Addr())}
	t.Cleanup(func() { _ = handlers.rateLimiter.redis.Close() })
	authorization := testDataExportAuthorization()
	session := model.UserSession{ID: "ses_export_redis", UserID: authorization.user.ID}
	ticket, _, err := handlers.issueDataExportTicket(context.Background(), authorization, session)
	if err != nil {
		t.Fatalf("issue Redis ticket: %v", err)
	}
	if len(ticket) < 2 || ticket[:2] != "r_" {
		t.Fatalf("ticket = %q, want Redis-backed prefix", ticket)
	}

	valid, err := handlers.consumeDataExportTicket(context.Background(), ticket, authorization, session)
	if err != nil || !valid {
		t.Fatalf("consume Redis ticket: valid=%v err=%v", valid, err)
	}
	valid, err = handlers.consumeDataExportTicket(context.Background(), ticket, authorization, session)
	if err != nil || valid {
		t.Fatalf("Redis ticket was reusable: valid=%v err=%v", valid, err)
	}
}

func TestDataExportTicketFailsClosedWithoutRedisInProduction(t *testing.T) {
	handlers := &Handlers{mode: "production"}
	authorization := testDataExportAuthorization()
	session := model.UserSession{ID: "ses_export_no_redis", UserID: authorization.user.ID}
	if _, _, err := handlers.issueDataExportTicket(context.Background(), authorization, session); err == nil {
		t.Fatal("expected production ticket issuance without Redis to fail closed")
	}
}

func testDataExportAuthorization() deploymentTargetDataExportAuthorization {
	return deploymentTargetDataExportAuthorization{
		user:    model.User{ID: "usr_export"},
		project: model.Project{ID: "prj_export"},
		app:     model.Application{ID: "app_export"},
		target:  model.DeploymentTarget{ID: "dplt_export"},
	}
}
