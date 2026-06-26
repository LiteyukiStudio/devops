package authz

import "testing"

func TestProjectRoleAllowsAction(t *testing.T) {
	if !ProjectRoleAllows(ProjectRoleDeveloper, ActionDeploymentRelease) {
		t.Fatal("expected developer to release deployments")
	}
	if ProjectRoleAllows(ProjectRoleViewer, ActionDeploymentRelease) {
		t.Fatal("expected viewer to be blocked from deployment release")
	}
	if !ProjectRoleAllows(ProjectRoleAdmin, ActionSecretViewValue) {
		t.Fatal("expected admin to view secret values")
	}
	if ProjectRoleAllows(ProjectRoleDeveloper, ActionSecretViewValue) {
		t.Fatal("expected developer to be blocked from secret values")
	}
}

func TestProjectActionForLegacyRoles(t *testing.T) {
	action, ok := ProjectActionForLegacyRoles([]string{ProjectRoleDeveloper, ProjectRoleOwner, ProjectRoleAdmin})
	if !ok || action != ActionProjectWrite {
		t.Fatalf("legacy write roles mapped to %q, ok=%t", action, ok)
	}

	if !ProjectRoleAllowsLegacyRoles(ProjectRoleOwner, []string{ProjectRoleOwner}) {
		t.Fatal("expected owner-only legacy role check to allow owner")
	}
	if ProjectRoleAllowsLegacyRoles(ProjectRoleAdmin, []string{ProjectRoleOwner}) {
		t.Fatal("expected owner-only legacy role check to block admin")
	}
}

func TestAccessTokenScopeRules(t *testing.T) {
	if scope := NormalizeAccessTokenScope("deployment:exec,build:trigger"); scope != "deployment:exec,build:trigger" {
		t.Fatalf("normalized scope = %q", scope)
	}
	if AccessTokenAllows("project:write", string(ActionDeploymentExec)) {
		t.Fatal("expected project:write to be too broad for deployment exec")
	}
	if !AccessTokenAllows("deployment:*", string(ActionDeploymentExec)) {
		t.Fatal("expected deployment wildcard to allow deployment exec")
	}
	if UserCanCreateAccessTokenScope(PlatformRoleUser, "deployment:exec") {
		t.Fatal("expected regular user to be blocked from creating write scopes")
	}
	if !UserCanCreateAccessTokenScope(PlatformRoleUser, "build:trigger,deployment:release") {
		t.Fatal("expected regular user to create automation trigger scopes")
	}
	if !UserCanCreateAccessTokenScope(PlatformRoleUser, "project:read,build:read") {
		t.Fatal("expected regular user to create read scopes")
	}
}
