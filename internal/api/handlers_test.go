package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/LiteyukiStudio/devops/internal/model"
)

func TestBootstrapStatusHidesDevLoginHintInProduction(t *testing.T) {
	t.Setenv("LOCAL_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("LOCAL_ADMIN_PASSWORD", "secret-password")

	status := bootstrapStatusResponse("production", false)

	if status["devLoginEnabled"] != false {
		t.Fatalf("expected dev login disabled in production, got %v", status["devLoginEnabled"])
	}
	if _, ok := status["devLoginHint"]; ok {
		t.Fatal("expected production bootstrap status to omit devLoginHint")
	}
}

func TestPaginationFromQueryDefaultsAndCapsPageSize(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/access-tokens?page=0&pageSize=999&sortBy=name&sortOrder=asc", nil)

	pagination := paginationFromQuery(ctx)

	if pagination.Page != 1 {
		t.Fatalf("Page = %d", pagination.Page)
	}
	if pagination.PageSize != 100 {
		t.Fatalf("PageSize = %d", pagination.PageSize)
	}
	if pagination.Offset() != 0 {
		t.Fatalf("Offset = %d", pagination.Offset())
	}
	if pagination.SortBy != "name" {
		t.Fatalf("SortBy = %q", pagination.SortBy)
	}
	if pagination.SortOrder != "asc" {
		t.Fatalf("SortOrder = %q", pagination.SortOrder)
	}
}

func TestPaginatedResponseCalculatesTotalPages(t *testing.T) {
	response := paginatedResponse([]string{"a", "b"}, 21, paginationParams{Page: 2, PageSize: 10, SortBy: "name", SortOrder: "asc"})

	if response["totalPages"] != 3 {
		t.Fatalf("totalPages = %v", response["totalPages"])
	}
	if response["total"] != int64(21) {
		t.Fatalf("total = %v", response["total"])
	}
	if response["sortBy"] != "name" || response["sortOrder"] != "asc" {
		t.Fatalf("sort response = %v/%v", response["sortBy"], response["sortOrder"])
	}
}

func TestOrderByClauseUsesWhitelist(t *testing.T) {
	pagination := paginationParams{SortBy: "name", SortOrder: "asc"}
	orderBy := orderByClause(pagination, map[string]string{"name": "name"}, "created_at")
	if orderBy != "name asc" {
		t.Fatalf("orderBy = %q", orderBy)
	}

	pagination = paginationParams{SortBy: "name;drop table users", SortOrder: "wat"}
	orderBy = orderByClause(pagination, map[string]string{"name": "name"}, "created_at")
	if orderBy != "created_at desc" {
		t.Fatalf("fallback orderBy = %q", orderBy)
	}
}

func TestBootstrapStatusIncludesDevLoginHintInDevelopment(t *testing.T) {
	t.Setenv("LOCAL_ADMIN_EMAIL", "Admin@Example.com")
	t.Setenv("LOCAL_ADMIN_PASSWORD", "secret-password")

	status := bootstrapStatusResponse("development", true)

	if status["devLoginEnabled"] != true {
		t.Fatalf("expected dev login enabled in development, got %v", status["devLoginEnabled"])
	}
	hint, ok := status["devLoginHint"].(gin.H)
	if !ok {
		t.Fatalf("expected devLoginHint map, got %T", status["devLoginHint"])
	}
	if hint["email"] != "admin@example.com" {
		t.Fatalf("expected normalized dev email, got %q", hint["email"])
	}
	if hint["password"] != "secret-password" {
		t.Fatalf("expected configured dev password, got %q", hint["password"])
	}
}

func TestAuthProviderResponseHidesStoredClientSecret(t *testing.T) {
	t.Setenv("SECRET_ENCRYPTION_KEY", "test-key")
	provider := model.AuthProvider{ClientSecretRef: storedSecretRef("super-secret")}

	output := authProviderResponse(provider)

	if output.ClientSecretRef != "" {
		t.Fatalf("expected stored client secret ref to be hidden, got %q", output.ClientSecretRef)
	}
	if !output.ClientSecretSet {
		t.Fatal("expected clientSecretSet to be true")
	}
}

func TestAuthProviderFromInputPreservesExistingSecret(t *testing.T) {
	t.Setenv("SECRET_ENCRYPTION_KEY", "test-key")
	existingSecretRef := storedSecretRef("old-secret")
	provider, ok := authProviderFromInput(authProviderInput{
		Type:      "oidc",
		Name:      "Casdoor",
		IssuerURL: "https://sso.example.com",
		ClientID:  "devops",
	}, "ap_existing", existingSecretRef)

	if !ok {
		t.Fatal("expected auth provider input to be valid")
	}
	if provider.ClientSecretRef != existingSecretRef {
		t.Fatalf("expected existing secret ref to be preserved, got %q", provider.ClientSecretRef)
	}
}

func TestResolveSecretSupportsStoredLiteralAndEnvRefs(t *testing.T) {
	t.Setenv("SECRET_ENCRYPTION_KEY", "test-key")
	t.Setenv("OIDC_TEST_SECRET", "env-secret")

	if secret := resolveSecret(storedSecretRef("stored-secret")); secret != "stored-secret" {
		t.Fatalf("expected stored secret, got %q", secret)
	}
	if secret := resolveSecret("literal:literal-secret"); secret != "literal-secret" {
		t.Fatalf("expected literal secret, got %q", secret)
	}
	if secret := resolveSecret("env:OIDC_TEST_SECRET"); secret != "env-secret" {
		t.Fatalf("expected env secret, got %q", secret)
	}
}

func TestVerifyGitWebhookSignatureSupportsGitHubAndGiteaHeaders(t *testing.T) {
	body := []byte(`{"after":"abc"}`)
	signature := hmacSHA256Hex(body, "secret")

	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+signature)
	if !verifyGitWebhookSignature(headers, body, "secret") {
		t.Fatal("expected GitHub webhook signature to verify")
	}

	headers = http.Header{}
	headers.Set("X-Gitea-Signature", signature)
	if !verifyGitWebhookSignature(headers, body, "secret") {
		t.Fatal("expected Gitea webhook signature to verify")
	}

	headers.Set("X-Gitea-Signature", "bad")
	if verifyGitWebhookSignature(headers, body, "secret") {
		t.Fatal("expected invalid webhook signature to fail")
	}
}

func TestGitWebhookCommitSHAReadsAfterField(t *testing.T) {
	sha := gitWebhookCommitSHA([]byte(`{"after":"abc123","sha":"ignored"}`))
	if sha != "abc123" {
		t.Fatalf("sha = %q", sha)
	}
}

func TestPathEscapePathPreservesPathSegments(t *testing.T) {
	escaped := pathEscapePath(".devops/app yaml.yml")
	if escaped != ".devops/app%20yaml.yml" {
		t.Fatalf("escaped path = %q", escaped)
	}
}

func TestFilterGitRepositoriesMatchesNameAndFullName(t *testing.T) {
	repos := []gitRepository{
		{Name: "api", FullName: "liteyuki/api"},
		{Name: "web", FullName: "liteyuki/web"},
	}
	filtered := filterGitRepositories(repos, "API")
	if len(filtered) != 1 || filtered[0].Name != "api" {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestGitOAuthEndpointDefaultsGitHub(t *testing.T) {
	endpoint, err := gitOAuthEndpoint(model.GitProvider{Type: "github"})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.AuthURL != "https://github.com/login/oauth/authorize" {
		t.Fatalf("auth url = %q", endpoint.AuthURL)
	}
}
