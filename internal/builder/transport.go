package builder

import (
	"context"
	"errors"
)

const (
	TransportHTTP = "http"
)

var errNoTask = errors.New("no builder task")

type Transport interface {
	Heartbeat(ctx context.Context, heartbeat Heartbeat) error
	Claim(ctx context.Context, currentConcurrency int) (Task, error)
	Renew(ctx context.Context, jobID string, leaseToken string, executor ExecutorRef) error
	AppendLogs(ctx context.Context, jobID string, leaseToken string, content string) error
	Progress(ctx context.Context, jobID string, leaseToken string, progress Progress) error
	Complete(ctx context.Context, jobID string, leaseToken string, result Result) error
	Fail(ctx context.Context, jobID string, leaseToken string, message string) error
	AppendHookLogs(ctx context.Context, hookRunID string, leaseToken string, content string) error
	CompleteHook(ctx context.Context, hookRunID string, leaseToken string, result HookResult) error
	Close() error
}

type CancelSubscriber interface {
	SubscribeCancel(ctx context.Context, jobID string, leaseToken string) (<-chan struct{}, func(), error)
}

type Heartbeat struct {
	AgentID            string   `json:"agentId"`
	Name               string   `json:"name"`
	Labels             []string `json:"labels"`
	Scopes             []string `json:"scopes"`
	Executor           string   `json:"executor"`
	MaxConcurrency     int      `json:"maxConcurrency"`
	CurrentConcurrency int      `json:"currentConcurrency"`
}

type ExecutorRef struct {
	ID   string `json:"executorId"`
	Name string `json:"executorName"`
}

func NewTransport(options Options) (Transport, error) {
	if options.Transport != "" && options.Transport != TransportHTTP {
		return nil, errors.New("only http builder transport is supported")
	}
	return NewHTTPTransport(options)
}
