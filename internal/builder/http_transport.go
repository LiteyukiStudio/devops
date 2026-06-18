package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPTransport struct {
	apiURL         string
	token          string
	name           string
	labels         []string
	executor       string
	maxConcurrency int
	client         *http.Client
}

func NewHTTPTransport(options Options) (*HTTPTransport, error) {
	if strings.TrimSpace(options.APIURL) == "" {
		return nil, errors.New("builder api url is required")
	}
	if strings.TrimSpace(options.Token) == "" {
		return nil, errors.New("builder token is required")
	}
	return &HTTPTransport{
		apiURL:         strings.TrimRight(options.APIURL, "/"),
		token:          strings.TrimSpace(options.Token),
		name:           strings.TrimSpace(options.Name),
		labels:         options.Labels,
		executor:       strings.TrimSpace(options.Executor),
		maxConcurrency: options.MaxConcurrency,
		client:         &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *HTTPTransport) Heartbeat(ctx context.Context, heartbeat Heartbeat) error {
	return t.post(ctx, "/api/v1/builder/heartbeat", heartbeat, nil)
}

func (t *HTTPTransport) Claim(ctx context.Context, currentConcurrency int) (Task, error) {
	var task Task
	err := t.post(ctx, "/api/v1/builder/tasks/claim", Heartbeat{
		Name:               t.name,
		Labels:             t.labels,
		Executor:           t.executor,
		MaxConcurrency:     t.maxConcurrency,
		CurrentConcurrency: currentConcurrency,
	}, &task)
	return task, err
}

func (t *HTTPTransport) Renew(ctx context.Context, jobID string, leaseToken string, executor ExecutorRef) error {
	return t.post(ctx, t.taskPath(jobID, leaseToken, "renew"), executor, nil)
}

func (t *HTTPTransport) AppendLogs(ctx context.Context, jobID string, leaseToken string, content string) error {
	return t.post(ctx, t.taskPath(jobID, leaseToken, "logs"), map[string]string{"content": content}, nil)
}

func (t *HTTPTransport) Progress(ctx context.Context, jobID string, leaseToken string, progress Progress) error {
	return t.post(ctx, t.taskPath(jobID, leaseToken, "progress"), progress, nil)
}

func (t *HTTPTransport) Complete(ctx context.Context, jobID string, leaseToken string, result Result) error {
	return t.post(ctx, t.taskPath(jobID, leaseToken, "complete"), result, nil)
}

func (t *HTTPTransport) Fail(ctx context.Context, jobID string, leaseToken string, message string) error {
	return t.post(ctx, t.taskPath(jobID, leaseToken, "fail"), map[string]string{"message": message}, nil)
}

func (t *HTTPTransport) AppendHookLogs(ctx context.Context, hookRunID string, leaseToken string, content string) error {
	return t.post(ctx, t.hookPath(hookRunID, leaseToken, "logs"), map[string]string{"content": content}, nil)
}

func (t *HTTPTransport) CompleteHook(ctx context.Context, hookRunID string, leaseToken string, result HookResult) error {
	return t.post(ctx, t.hookPath(hookRunID, leaseToken, "complete"), result, nil)
}

func (t *HTTPTransport) SubscribeCancel(ctx context.Context, jobID string, leaseToken string) (<-chan struct{}, func(), error) {
	cancelled := make(chan struct{})
	stop := make(chan struct{})
	ticker := time.NewTicker(2 * time.Second)
	cleanup := func() {
		ticker.Stop()
		select {
		case <-stop:
		default:
			close(stop)
		}
	}
	go func() {
		defer cleanup()
		for {
			canceled, err := t.cancelled(ctx, jobID, leaseToken)
			if err == nil && canceled {
				close(cancelled)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return cancelled, cleanup, nil
}

func (t *HTTPTransport) Close() error {
	return nil
}

func (t *HTTPTransport) taskPath(jobID string, leaseToken string, action string) string {
	values := url.Values{}
	values.Set("leaseToken", leaseToken)
	return fmt.Sprintf("/api/v1/builder/tasks/%s/%s?%s", url.PathEscape(jobID), action, values.Encode())
}

func (t *HTTPTransport) hookPath(hookRunID string, leaseToken string, action string) string {
	values := url.Values{}
	values.Set("leaseToken", leaseToken)
	return fmt.Sprintf("/api/v1/builder/hooks/%s/%s?%s", url.PathEscape(hookRunID), action, values.Encode())
}

func (t *HTTPTransport) post(ctx context.Context, path string, payload any, output any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.apiURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent && output != nil {
		return errNoTask
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("builder api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if output == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(output)
}

func (t *HTTPTransport) cancelled(ctx context.Context, jobID string, leaseToken string) (bool, error) {
	values := url.Values{}
	values.Set("leaseToken", leaseToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/builder/tasks/%s/cancelled?%s", t.apiURL, url.PathEscape(jobID), values.Encode()), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+t.token)
	resp, err := t.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return false, fmt.Errorf("builder api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var output struct {
		Cancelled bool `json:"cancelled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return false, err
	}
	return output.Cancelled, nil
}
