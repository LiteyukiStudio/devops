package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TypeDeployRun         = "deploy:run"
	TypeGatewayApply      = "gateway:apply"
	TypeApplicationDelete = "application:delete"
	TypeGitAccountRefresh = "git:accounts:refresh"
	TypeSyncStatus        = "sync:status"

	QueueDeploy = "deploy"
	QueueLight  = "light"
)

type DeployRunPayload struct {
	Envelope  TaskEnvelope `json:"envelope"`
	ReleaseID string       `json:"releaseId"`
	ProjectID string       `json:"projectId"`
	ActorID   string       `json:"actorId"`
}

type GatewayApplyPayload struct {
	Envelope       TaskEnvelope `json:"envelope"`
	GatewayRouteID string       `json:"gatewayRouteId"`
	ProjectID      string       `json:"projectId"`
	ActorID        string       `json:"actorId"`
}

type ApplicationDeletePayload struct {
	Envelope      TaskEnvelope `json:"envelope"`
	ApplicationID string       `json:"applicationId"`
	ProjectID     string       `json:"projectId"`
	ActorID       string       `json:"actorId"`
	DeleteData    bool         `json:"deleteData"`
}

type GitAccountRefreshPayload struct {
	Envelope TaskEnvelope `json:"envelope"`
	ActorID  string       `json:"actorId"`
}

type TaskEnvelope struct {
	TaskID      string    `json:"taskId"`
	TaskType    string    `json:"taskType"`
	DedupeKey   string    `json:"dedupeKey"`
	ActorID     string    `json:"actorId"`
	ResourceRef string    `json:"resourceRef"`
	TraceID     string    `json:"traceId"`
	Attempt     int       `json:"attempt"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Client struct {
	client *asynq.Client
}

type EnqueuePolicy struct {
	Queue     string
	MaxRetry  int
	Timeout   time.Duration
	Retention time.Duration
	Unique    time.Duration
}

func NewClient(redisAddr string) *Client {
	return &Client{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) EnqueueDeployRun(ctx context.Context, payload DeployRunPayload) (*asynq.TaskInfo, error) {
	task, err := NewDeployRunTask(payload)
	if err != nil {
		return nil, err
	}

	return c.enqueueWithPolicy(ctx, task, PolicyForType(TypeDeployRun))
}

func (c *Client) EnqueueGatewayApply(ctx context.Context, payload GatewayApplyPayload) (*asynq.TaskInfo, error) {
	task, err := NewGatewayApplyTask(payload)
	if err != nil {
		return nil, err
	}

	return c.enqueueWithPolicy(ctx, task, PolicyForType(TypeGatewayApply))
}

func (c *Client) EnqueueApplicationDelete(ctx context.Context, payload ApplicationDeletePayload) (*asynq.TaskInfo, error) {
	task, err := NewApplicationDeleteTask(payload)
	if err != nil {
		return nil, err
	}

	return c.enqueueWithPolicy(ctx, task, PolicyForType(TypeApplicationDelete))
}

func (c *Client) EnqueueGitAccountRefresh(ctx context.Context, payload GitAccountRefreshPayload) (*asynq.TaskInfo, error) {
	task, err := NewGitAccountRefreshTask(payload)
	if err != nil {
		return nil, err
	}

	return c.enqueueWithPolicy(ctx, task, PolicyForType(TypeGitAccountRefresh))
}

func (c *Client) enqueueWithPolicy(ctx context.Context, task *asynq.Task, policy EnqueuePolicy) (*asynq.TaskInfo, error) {
	return c.client.EnqueueContext(
		ctx,
		task,
		asynq.Queue(policy.Queue),
		asynq.MaxRetry(policy.MaxRetry),
		asynq.Timeout(policy.Timeout),
		asynq.Retention(policy.Retention),
		asynq.Unique(policy.Unique),
	)
}

func PolicyForType(taskType string) EnqueuePolicy {
	switch taskType {
	case TypeDeployRun:
		return EnqueuePolicy{Queue: QueueDeploy, MaxRetry: 3, Timeout: 30 * time.Minute, Retention: 24 * time.Hour, Unique: 30 * time.Minute}
	case TypeGatewayApply:
		return EnqueuePolicy{Queue: QueueDeploy, MaxRetry: 3, Timeout: 10 * time.Minute, Retention: 24 * time.Hour, Unique: 10 * time.Minute}
	case TypeApplicationDelete:
		return EnqueuePolicy{Queue: QueueDeploy, MaxRetry: 3, Timeout: 15 * time.Minute, Retention: 24 * time.Hour, Unique: 10 * time.Minute}
	case TypeGitAccountRefresh:
		return EnqueuePolicy{Queue: QueueLight, MaxRetry: 2, Timeout: 10 * time.Minute, Retention: 24 * time.Hour, Unique: 5 * time.Minute}
	default:
		return EnqueuePolicy{Queue: QueueLight, MaxRetry: 1, Timeout: 5 * time.Minute, Retention: 24 * time.Hour, Unique: 1 * time.Minute}
	}
}

func NewDeployRunTask(payload DeployRunPayload) (*asynq.Task, error) {
	if strings.TrimSpace(payload.ReleaseID) == "" {
		return nil, errors.New("release id is required")
	}
	if strings.TrimSpace(payload.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}

	payload.Envelope = ensureEnvelope(payload.Envelope, TypeDeployRun, payload.ActorID, payload.ProjectID, payload.ReleaseID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeDeployRun, data), nil
}

func NewGatewayApplyTask(payload GatewayApplyPayload) (*asynq.Task, error) {
	if strings.TrimSpace(payload.GatewayRouteID) == "" {
		return nil, errors.New("gateway route id is required")
	}
	if strings.TrimSpace(payload.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}

	payload.Envelope = ensureEnvelope(payload.Envelope, TypeGatewayApply, payload.ActorID, payload.ProjectID, payload.GatewayRouteID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGatewayApply, data), nil
}

func NewApplicationDeleteTask(payload ApplicationDeletePayload) (*asynq.Task, error) {
	if strings.TrimSpace(payload.ApplicationID) == "" {
		return nil, errors.New("application id is required")
	}
	if strings.TrimSpace(payload.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}

	payload.Envelope = ensureEnvelope(payload.Envelope, TypeApplicationDelete, payload.ActorID, payload.ProjectID, payload.ApplicationID)
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeApplicationDelete, data), nil
}

func NewGitAccountRefreshTask(payload GitAccountRefreshPayload) (*asynq.Task, error) {
	payload.Envelope = ensureEnvelope(payload.Envelope, TypeGitAccountRefresh, payload.ActorID, "system", "git-accounts")
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGitAccountRefresh, data), nil
}

func ensureEnvelope(envelope TaskEnvelope, taskType string, actorID string, scope string, resourceID string) TaskEnvelope {
	if strings.TrimSpace(envelope.TaskType) == "" {
		envelope.TaskType = taskType
	}
	if strings.TrimSpace(envelope.ActorID) == "" {
		envelope.ActorID = strings.TrimSpace(actorID)
	}
	if strings.TrimSpace(envelope.ResourceRef) == "" {
		envelope.ResourceRef = strings.TrimSpace(resourceID)
	}
	if strings.TrimSpace(envelope.DedupeKey) == "" {
		envelope.DedupeKey = taskType + ":" + strings.TrimSpace(scope) + ":" + strings.TrimSpace(resourceID)
	}
	if strings.TrimSpace(envelope.TaskID) == "" {
		envelope.TaskID = envelope.DedupeKey
	}
	if strings.TrimSpace(envelope.TraceID) == "" {
		envelope.TraceID = envelope.TaskID
	}
	return envelope
}
