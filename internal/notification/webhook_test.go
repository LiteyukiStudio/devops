package notification

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestWebhookAdapterRendersURLHeadersAndBody(t *testing.T) {
	adapter := WebhookAdapter{}
	config := json.RawMessage(`{
		"method": "POST",
		"url": "https://8.8.8.8/hooks/{{.Secrets.Token}}",
		"headers": {
			"X-Project": "{{.Event.Project.Identifier}}"
		}
	}`)
	secrets := json.RawMessage(`{"Token":"token-ref"}`)
	event := Event{
		ID:         "evt_1",
		Type:       "hook.failed",
		Severity:   SeverityError,
		Message:    "migration failed",
		OccurredAt: time.Now(),
		Project:    EntityRef{Name: "Demo", Identifier: "demo"},
	}
	message, err := adapter.Render(context.Background(), event, Template{JSON: `{"text": {{json .Event.Message}}}`}, config, secrets, StaticSecretResolver{"token-ref": "abc"}, "")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if message.URL != "https://8.8.8.8/hooks/abc" {
		t.Fatalf("url = %q", message.URL)
	}
	if message.Headers["X-Project"] != "demo" {
		t.Fatalf("headers = %#v", message.Headers)
	}
	if string(message.JSON) != `{"text":"migration failed"}` {
		t.Fatalf("json = %s", string(message.JSON))
	}
}

func TestWebhookAdapterRejectsUnsafeMethod(t *testing.T) {
	adapter := WebhookAdapter{}
	err := adapter.Validate(context.Background(), json.RawMessage(`{"method":"GET","url":"https://example.com"}`), nil)
	if err == nil {
		t.Fatal("expected unsafe method to fail")
	}
}

func TestWebhookAdapterRegistry(t *testing.T) {
	registry := DefaultRegistry()
	if _, err := registry.Adapter(AdapterKindWebhook); err != nil {
		t.Fatalf("webhook adapter missing: %v", err)
	}
	if _, err := registry.Adapter(AdapterKindSMTP); err != nil {
		t.Fatalf("smtp adapter missing: %v", err)
	}
}
