package service

import (
	"net/http"
	"testing"
)

func TestRequiredAccessTokenScopeRequiresWriteForReleaseRuntimeExec(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
	}{
		{
			name:   "terminal websocket",
			path:   "/api/v1/projects/:projectId/releases/:releaseId/terminal",
			method: http.MethodGet,
		},
		{
			name:   "exec command",
			path:   "/api/v1/projects/:projectId/releases/:releaseId/exec",
			method: http.MethodPost,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			required := RequiredAccessTokenScope(test.path, test.method)
			if required != "project:write" {
				t.Fatalf("RequiredAccessTokenScope(%q, %q) = %q, want project:write", test.path, test.method, required)
			}
			if AccessTokenAllows("project:read", required) {
				t.Fatal("expected project:read token to be denied")
			}
			if !AccessTokenAllows("project:write", required) {
				t.Fatal("expected project:write token to be allowed")
			}
		})
	}
}

func TestRequiredAccessTokenScopeKeepsReleaseRuntimeLogsReadOnly(t *testing.T) {
	required := RequiredAccessTokenScope("/api/v1/projects/:projectId/releases/:releaseId/runtime-logs", http.MethodGet)

	if required != "project:read" {
		t.Fatalf("runtime logs required scope = %q, want project:read", required)
	}
	if !AccessTokenAllows("project:read", required) {
		t.Fatal("expected project:read token to read runtime logs")
	}
}
