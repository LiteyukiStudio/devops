package notification

import (
	"strings"
	"testing"
	"time"
)

func TestTestEventProvidesCommonTemplateVariables(t *testing.T) {
	event := TestEvent(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	message, err := renderMessage(event, Template{
		Body: "{{.Event.Project.Name}} {{.Event.Application.Name}} {{.Event.DeploymentTarget.Name}} {{.Event.Build.ID}} {{.Event.Release.ID}} {{.Event.Hook.Name}} {{.Event.Gateway.Domain}}",
	}, nil)
	if err != nil {
		t.Fatalf("renderMessage returned error: %v", err)
	}
	for _, expected := range []string{"Demo Project Space", "Demo Application", "production", "build_test", "rel_test", "pre-release-check", "demo-app.apps.example.com"} {
		if !strings.Contains(message.Body, expected) {
			t.Fatalf("test event body %q does not contain %q", message.Body, expected)
		}
	}
}
