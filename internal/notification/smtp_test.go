package notification

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSMTPAdapterValidatesConfig(t *testing.T) {
	adapter := SMTPAdapter{}
	config := json.RawMessage(`{
		"host": "smtp.example.com",
		"port": 587,
		"security": "starttls",
		"from": "DevOps <devops@example.com>",
		"to": ["ops@example.com"]
	}`)
	if err := adapter.Validate(context.Background(), config, nil); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestSMTPAdapterRequiresRecipient(t *testing.T) {
	adapter := SMTPAdapter{}
	err := adapter.Validate(context.Background(), json.RawMessage(`{"host":"smtp.example.com","from":"devops@example.com"}`), nil)
	if err == nil {
		t.Fatal("expected missing recipient to fail")
	}
}

func TestSMTPAdapterRendersSubjectAndBody(t *testing.T) {
	adapter := SMTPAdapter{}
	event := Event{
		Type:       "release.failed",
		Severity:   SeverityError,
		Message:    "rollout failed",
		OccurredAt: time.Now(),
	}
	message, err := adapter.Render(context.Background(), event, Template{
		Subject: "[{{.Event.Severity}}] {{.Event.Type}}",
		Body:    "{{.Event.Message}}",
	}, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if message.Subject != "[error] release.failed" || message.Body != "rollout failed" {
		t.Fatalf("message = %#v", message)
	}
}
