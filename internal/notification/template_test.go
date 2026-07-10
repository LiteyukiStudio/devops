package notification

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMessageValidatesJSONBody(t *testing.T) {
	event := Event{
		ID:         "evt_1",
		Type:       "release.failed",
		Severity:   SeverityError,
		Message:    "rollout failed",
		OccurredAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Project:    EntityRef{Name: "demo"},
	}
	message, err := renderMessage(event, Template{
		Subject: "{{.Event.Type}}",
		Body:    "{{.Event.Message}}",
		JSON:    `{"text": {{json .Event.Message}}, "project": {{json .Event.Project.Name}}}`,
	}, nil)
	if err != nil {
		t.Fatalf("renderMessage returned error: %v", err)
	}
	if message.Subject != "release.failed" || message.Body != "rollout failed" {
		t.Fatalf("message = %#v", message)
	}
	if string(message.JSON) != `{"project":"demo","text":"rollout failed"}` {
		t.Fatalf("json = %s", string(message.JSON))
	}
}

func TestRenderMessageRejectsMissingTemplateKey(t *testing.T) {
	_, err := renderMessage(Event{}, Template{Body: "{{.Event.UnknownField}}"}, nil)
	if err == nil || !strings.Contains(err.Error(), "UnknownField") {
		t.Fatalf("error = %v", err)
	}
}

func TestRenderMessageRejectsInvalidJSONBody(t *testing.T) {
	_, err := renderMessage(Event{Message: "hello"}, Template{JSON: `{"text": "{{.Event.Message}}"`}, nil)
	if err == nil || !strings.Contains(err.Error(), "valid json") {
		t.Fatalf("error = %v", err)
	}
}

func TestRenderMessageSupportsDetailHelpers(t *testing.T) {
	event := TestEvent(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	message, err := renderMessage(event, Template{
		Body: `{{detailsTitle .Event}}
{{details .Event "zh"}}
{{link .Event.Links "primary"}}`,
	}, nil)
	if err != nil {
		t.Fatalf("renderMessage returned error: %v", err)
	}
	for _, want := range []string{
		"[info] notification.test",
		"项目空间: Demo Project Space / demo-space / prj_test",
		"构建详情",
		"发布详情",
		"访问入口详情",
		"https://devops.example.com/projects/prj_test/apps/app_test#tab=deployments",
	} {
		if !strings.Contains(message.Body, want) {
			t.Fatalf("body does not contain %q:\n%s", want, message.Body)
		}
	}
}
