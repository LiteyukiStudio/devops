package notification

import (
	"context"
	"encoding/json"
	"time"
)

const (
	AdapterKindWebhook = "webhook"
	AdapterKindSMTP    = "smtp"

	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

type Event struct {
	ID               string
	Type             string
	Severity         string
	Locale           string
	Project          EntityRef
	Application      EntityRef
	DeploymentTarget EntityRef
	Build            BuildContext
	Release          ReleaseContext
	Hook             HookContext
	Gateway          GatewayContext
	Actor            ActorContext
	Links            map[string]string
	OccurredAt       time.Time
	Message          string
}

type EntityRef struct {
	ID   string
	Name string
	Slug string
}

type BuildContext struct {
	ID      string
	Status  string
	Message string
	Image   string
	GitRef  string
	GitSHA  string
}

type ReleaseContext struct {
	ID       string
	Status   string
	Revision int
	ImageRef string
	Message  string
}

type HookContext struct {
	ID      string
	Name    string
	Phase   string
	Status  string
	Message string
}

type GatewayContext struct {
	ID      string
	Domain  string
	Path    string
	Status  string
	Message string
}

type ActorContext struct {
	ID    string
	Name  string
	Email string
}

type Template struct {
	Subject string
	Body    string
	JSON    string
}

type RenderedMessage struct {
	Subject string
	Body    string
	JSON    []byte
	Method  string
	URL     string
	Headers map[string]string
}

type SendResult struct {
	StatusCode      int
	ResponseSnippet string
}

type Adapter interface {
	Kind() string
	Validate(ctx context.Context, config json.RawMessage, secretResolver SecretResolver) error
	Render(ctx context.Context, event Event, template Template, config json.RawMessage, secrets json.RawMessage, secretResolver SecretResolver, locale string) (RenderedMessage, error)
	Send(ctx context.Context, config json.RawMessage, secrets json.RawMessage, message RenderedMessage, secretResolver SecretResolver) (SendResult, error)
	Test(ctx context.Context, config json.RawMessage, secrets json.RawMessage, secretResolver SecretResolver) error
}

type SecretResolver interface {
	Resolve(ref string) string
}

type StaticSecretResolver map[string]string

func (r StaticSecretResolver) Resolve(ref string) string {
	return r[ref]
}
