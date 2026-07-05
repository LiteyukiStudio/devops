package notification

import "time"

func TestEvent(now time.Time) Event {
	if now.IsZero() {
		now = time.Now()
	}
	return Event{
		ID:       "notification-test",
		Type:     "notification.test",
		Severity: SeverityInfo,
		Project: EntityRef{
			ID:   "prj_test",
			Name: "Demo Project Space",
			Slug: "demo-space",
		},
		Application: EntityRef{
			ID:   "app_test",
			Name: "Demo Application",
			Slug: "demo-app",
		},
		DeploymentTarget: EntityRef{
			ID:   "dplt_test",
			Name: "production",
			Slug: "prod",
		},
		Build: BuildContext{
			ID:      "build_test",
			Status:  "failed",
			Message: "Dockerfile build failed during dependency install",
			Image:   "registry.example.com/demo/demo-app:test",
			GitRef:  "refs/heads/main",
			GitSHA:  "0123456789abcdef",
		},
		Release: ReleaseContext{
			ID:       "rel_test",
			Status:   "failed",
			Revision: 12,
			ImageRef: "registry.example.com/demo/demo-app:test",
			Message:  "Rollout did not become ready before deadline",
		},
		Hook: HookContext{
			ID:      "hook_test",
			Name:    "pre-release-check",
			Phase:   "pre_deploy",
			Status:  "failed",
			Message: "Migration command exited with status 1",
		},
		Gateway: GatewayContext{
			ID:      "gwr_test",
			Domain:  "demo-app.apps.example.com",
			Path:    "/",
			Status:  "accepted",
			Message: "HTTPRoute accepted by Gateway",
		},
		Actor: ActorContext{
			ID:    "usr_test",
			Name:  "Platform Admin",
			Email: "admin@example.com",
		},
		Links: map[string]string{
			"application": "https://devops.example.com/applications/app_test",
			"release":     "https://devops.example.com/releases/rel_test",
		},
		OccurredAt: now,
		Message:    "Liteyuki DevOps test notification with preset template variables.",
	}
}
