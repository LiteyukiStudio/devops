package builder

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed executor/run.sh
var executorRunScript string

const (
	hookPhasePrePull   = "prePull"
	hookPhasePostPull  = "postPull"
	hookPhasePreBuild  = "preBuild"
	hookPhasePostBuild = "postBuild"
	hookPhasePrePush   = "prePush"
	hookPhasePostPush  = "postPush"
)

var buildHookPhases = []string{
	hookPhasePrePull,
	hookPhasePostPull,
	hookPhasePreBuild,
	hookPhasePostBuild,
	hookPhasePrePush,
	hookPhasePostPush,
}

type Options struct {
	APIURL            string
	Token             string
	Transport         string
	AgentID           string
	Name              string
	Executor          string
	ExecutorImage     string
	Labels            []string
	MaxConcurrency    int
	Scopes            []string
	PollInterval      time.Duration
	WorkspaceRoot     string
	WorkspaceHostRoot string
	NPMRegistry       string
	CacheEnabled      bool
	CacheTag          string
	DockerBinary      string
}

type Agent struct {
	options   Options
	transport Transport
	mu        sync.Mutex
	running   int
}

func New(options Options) *Agent {
	if options.MaxConcurrency <= 0 {
		options.MaxConcurrency = 1
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 5 * time.Second
	}
	if strings.TrimSpace(options.Executor) == "" {
		options.Executor = "docker"
	}
	if strings.TrimSpace(options.Transport) == "" {
		options.Transport = TransportHTTP
	}
	if strings.TrimSpace(options.ExecutorImage) == "" {
		options.ExecutorImage = "moby/buildkit:v0.24.0-rootless"
	}
	if strings.TrimSpace(options.WorkspaceRoot) == "" {
		options.WorkspaceRoot = "/builder-workspace"
	}
	if strings.TrimSpace(options.DockerBinary) == "" {
		options.DockerBinary = "docker"
	}
	return &Agent{options: options}
}

func (a *Agent) Run(ctx context.Context) error {
	agentName := strings.TrimSpace(a.options.Name)
	if agentName == "" {
		agentName = "local-builder"
	}
	a.options.Name = agentName
	a.options.AgentID = agentName
	a.options.Transport = TransportHTTP
	transport, err := NewTransport(a.options)
	if err != nil {
		return err
	}
	a.transport = transport
	defer func() {
		if err := a.transport.Close(); err != nil {
			log.Printf("builder transport close failed: %v", err)
		}
	}()
	if err := os.MkdirAll(a.options.WorkspaceRoot, 0o700); err != nil {
		return err
	}
	ticker := time.NewTicker(a.options.PollInterval)
	defer ticker.Stop()
	for {
		current := a.currentConcurrency()
		if err := a.heartbeat(ctx, current); err != nil {
			log.Printf("builder heartbeat failed: %v", err)
		}
		for a.currentConcurrency() < a.options.MaxConcurrency {
			task, ok, err := a.claim(ctx)
			if err != nil {
				log.Printf("builder claim failed: %v", err)
				break
			}
			if !ok {
				break
			}
			a.incrementConcurrency()
			go func() {
				defer a.decrementConcurrency()
				a.runTask(ctx, task)
			}()
		}
		if a.currentConcurrency() >= a.options.MaxConcurrency {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a *Agent) runTask(ctx context.Context, task Task) {
	log.Printf("builder task claimed: job=%s run=%s image=%s", task.JobID, task.BuildRunID, task.Registry.ImageRef)
	if err := a.heartbeat(ctx, a.currentConcurrency()); err != nil {
		log.Printf("builder heartbeat failed: %v", err)
	}
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if subscriber, ok := a.transport.(CancelSubscriber); ok {
		cancelled, cleanup, err := subscriber.SubscribeCancel(taskCtx, task.JobID, task.LeaseToken)
		if err != nil {
			log.Printf("builder cancel subscribe failed: %v", err)
		}
		if cancelled != nil {
			defer cleanup()
			go func() {
				select {
				case <-cancelled:
					log.Printf("builder task cancel received: job=%s run=%s", task.JobID, task.BuildRunID)
					cancel()
				case <-taskCtx.Done():
				}
			}()
		}
	}
	a.renew(taskCtx, task)
	go a.renewTaskLeaseLoop(taskCtx, task)
	result, logs, err := a.executeDockerTask(taskCtx, task, func(content string) error {
		return a.appendLogs(taskCtx, task.JobID, task.LeaseToken, content)
	}, func(progress Progress) error {
		return a.progress(taskCtx, task.JobID, task.LeaseToken, progress)
	}, func(hookRunID string, content string) error {
		return a.transport.AppendHookLogs(taskCtx, hookRunID, task.LeaseToken, content)
	}, func(hookRunID string, result HookResult) error {
		return a.transport.CompleteHook(taskCtx, hookRunID, task.LeaseToken, result)
	})
	if err != nil {
		if errors.Is(taskCtx.Err(), context.Canceled) {
			log.Printf("builder task canceled locally: job=%s run=%s", task.JobID, task.BuildRunID)
			if failErr := a.fail(ctx, task.JobID, task.LeaseToken, "canceled by user"); failErr != nil {
				log.Printf("builder cancel ack failed: %v", failErr)
			}
			return
		}
		message := err.Error()
		if logs != "" {
			message = firstLine(logs, message)
		}
		if failErr := a.fail(ctx, task.JobID, task.LeaseToken, message); failErr != nil {
			log.Printf("builder fail report failed: %v", failErr)
		}
		return
	}
	if err := a.complete(ctx, task.JobID, task.LeaseToken, result); err != nil {
		log.Printf("builder complete report failed: %v", err)
	}
}

func (a *Agent) heartbeat(ctx context.Context, current int) error {
	return a.transport.Heartbeat(ctx, Heartbeat{
		AgentID:            a.options.AgentID,
		Name:               a.options.Name,
		Labels:             a.options.Labels,
		Scopes:             a.options.Scopes,
		Executor:           a.options.Executor,
		MaxConcurrency:     a.options.MaxConcurrency,
		CurrentConcurrency: current,
	})
}

func (a *Agent) claim(ctx context.Context) (Task, bool, error) {
	task, err := a.transport.Claim(ctx, a.currentConcurrency())
	if errors.Is(err, errNoTask) {
		return Task{}, false, nil
	}
	return task, task.JobID != "", err
}

func (a *Agent) currentConcurrency() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *Agent) incrementConcurrency() {
	a.mu.Lock()
	a.running++
	a.mu.Unlock()
}

func (a *Agent) decrementConcurrency() {
	a.mu.Lock()
	if a.running > 0 {
		a.running--
	}
	a.mu.Unlock()
}

func (a *Agent) renew(ctx context.Context, task Task) {
	if strings.TrimSpace(task.LeaseToken) == "" {
		return
	}
	if err := a.transport.Renew(ctx, task.JobID, task.LeaseToken, ExecutorRef{Name: executorContainerName(task.JobID)}); err != nil {
		log.Printf("builder task lease renew failed: job=%s err=%v", task.JobID, err)
	}
}

func (a *Agent) renewTaskLeaseLoop(ctx context.Context, task Task) {
	interval := 10 * time.Second
	if a.options.PollInterval > 0 && a.options.PollInterval < interval {
		interval = a.options.PollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.renew(ctx, task)
		}
	}
}

func (a *Agent) appendLogs(ctx context.Context, jobID string, leaseToken string, content string) error {
	return a.transport.AppendLogs(ctx, jobID, leaseToken, content)
}

func (a *Agent) progress(ctx context.Context, jobID string, leaseToken string, progress Progress) error {
	if strings.TrimSpace(progress.Key) == "" {
		return nil
	}
	return a.transport.Progress(ctx, jobID, leaseToken, progress)
}

func (a *Agent) complete(ctx context.Context, jobID string, leaseToken string, result Result) error {
	return a.transport.Complete(ctx, jobID, leaseToken, result)
}

func (a *Agent) fail(ctx context.Context, jobID string, leaseToken string, message string) error {
	return a.transport.Fail(ctx, jobID, leaseToken, message)
}

func (a *Agent) executeDockerTask(ctx context.Context, task Task, onLog func(string) error, onProgress func(Progress) error, onHookLog func(string, string) error, onHookComplete func(string, HookResult) error) (Result, string, error) {
	if a.options.Executor != "docker" && a.options.Executor != "docker-dind" {
		return Result{}, "", fmt.Errorf("unsupported builder executor: %s", a.options.Executor)
	}
	workspace, err := os.MkdirTemp(a.options.WorkspaceRoot, task.JobID+"-")
	if err != nil {
		return Result{}, "", err
	}
	defer os.RemoveAll(workspace)
	volumeWorkspace := a.executorVolumeWorkspace(workspace)
	scriptPath := filepath.Join(workspace, "run.sh")
	resultPath := filepath.Join(workspace, "result.json")
	if err := os.WriteFile(scriptPath, []byte(executorScript()), 0o700); err != nil {
		return Result{}, "", err
	}
	hookIDsByPhase, err := writeHookScripts(workspace, task.Build.Hooks)
	if err != nil {
		return Result{}, "", err
	}
	args := []string{
		"run",
		"--name", executorContainerName(task.JobID),
		"--privileged",
		"--security-opt", "seccomp=unconfined",
		"--entrypoint", "/bin/sh",
		"-v", volumeWorkspace + ":/workspace",
		"-e", "GIT_CLONE_URL=" + task.Repository.CloneURL,
		"-e", "GIT_ACCESS_TOKEN=" + task.Repository.AccessToken,
		"-e", "SOURCE_BRANCH=" + task.Repository.SourceBranch,
		"-e", "SOURCE_TAG=" + task.Repository.SourceTag,
		"-e", "SOURCE_COMMIT=" + task.Repository.SourceCommit,
		"-e", "LITEYUKI_PROJECT_ID=" + task.ProjectID,
		"-e", "LITEYUKI_APPLICATION_ID=" + task.ApplicationID,
		"-e", "LITEYUKI_DEPLOYMENT_TARGET_ID=" + task.DeploymentTargetID,
		"-e", "LITEYUKI_BUILD_RUN_ID=" + task.BuildRunID,
		"-e", "LITEYUKI_BUILD_JOB_ID=" + task.JobID,
		"-e", "DOCKERFILE_PATH=" + task.Build.DockerfilePath,
		"-e", "BUILD_CONTEXT=" + task.Build.BuildContext,
		"-e", "BUILD_DIRECTORY=" + task.Build.BuildDirectory,
		"-e", "CACHE_ENABLED=" + boolEnvValue(a.options.CacheEnabled),
		"-e", "CACHE_TAG=" + stringDefault(strings.TrimSpace(a.options.CacheTag), "buildcache"),
		"-e", "NPM_REGISTRY=" + a.options.NPMRegistry,
		"-e", "REGISTRY_ENDPOINT=" + task.Registry.Endpoint,
		"-e", "REGISTRY_USERNAME=" + task.Registry.Username,
		"-e", "REGISTRY_PASSWORD=" + task.Registry.Password,
		"-e", "IMAGE_REF=" + task.Registry.ImageRef,
		"-e", "IMAGE_NAME_PREFIX=" + task.Registry.ImageNamePrefix,
		"-e", "IMAGE_TAG_TEMPLATE=" + task.Registry.ImageTagTemplate,
		"-e", "PRE_PULL_HOOK_IDS=" + strings.Join(hookIDsByPhase[hookPhasePrePull], ","),
		"-e", "POST_PULL_HOOK_IDS=" + strings.Join(hookIDsByPhase[hookPhasePostPull], ","),
		"-e", "PRE_BUILD_HOOK_IDS=" + strings.Join(hookIDsByPhase[hookPhasePreBuild], ","),
		"-e", "POST_BUILD_HOOK_IDS=" + strings.Join(hookIDsByPhase[hookPhasePostBuild], ","),
		"-e", "PRE_PUSH_HOOK_IDS=" + strings.Join(hookIDsByPhase[hookPhasePrePush], ","),
		"-e", "POST_PUSH_HOOK_IDS=" + strings.Join(hookIDsByPhase[hookPhasePostPush], ","),
	}
	buildEnv := normalizedBuildEnv(task.Build.Env)
	if strings.TrimSpace(a.options.NPMRegistry) != "" {
		if _, ok := buildEnv["NPM_REGISTRY"]; !ok {
			buildEnv["NPM_REGISTRY"] = strings.TrimSpace(a.options.NPMRegistry)
		}
		if _, ok := buildEnv["npm_config_registry"]; !ok {
			buildEnv["npm_config_registry"] = strings.TrimSpace(a.options.NPMRegistry)
		}
	}
	buildEnvKeys := make([]string, 0, len(buildEnv))
	for key := range buildEnv {
		buildEnvKeys = append(buildEnvKeys, key)
	}
	sort.Strings(buildEnvKeys)
	args = append(args, "-e", "BUILD_ENV_KEYS="+strings.Join(buildEnvKeys, ","))
	for _, key := range buildEnvKeys {
		args = append(args, "-e", key+"="+buildEnv[key])
	}
	args = append(args, a.options.ExecutorImage, "/workspace/run.sh")
	cmd := exec.CommandContext(ctx, a.options.DockerBinary, args...)
	containerName := executorContainerName(task.JobID)
	a.removeExecutorContainer(containerName)
	defer a.removeExecutorContainer(containerName)
	cancelCleanupStop := make(chan struct{})
	cancelCleanupDone := make(chan struct{})
	go func() {
		defer close(cancelCleanupDone)
		select {
		case <-ctx.Done():
			_ = exec.Command(a.options.DockerBinary, "rm", "-f", containerName).Run()
		case <-cancelCleanupStop:
		}
	}()
	output := newConcurrentBuffer()
	streamer := newLogStreamer(ctx, output, task.Build.Hooks, onLog, onProgress, onHookLog, onHookComplete)
	defer streamer.Close()
	cmd.Stdout = streamer
	cmd.Stderr = streamer
	err = cmd.Run()
	close(cancelCleanupStop)
	<-cancelCleanupDone
	streamer.Close()
	result := Result{ImageRef: task.Registry.ImageRef}
	if data, readErr := os.ReadFile(resultPath); readErr == nil {
		_ = json.Unmarshal(data, &result)
	}
	if err != nil && ctx.Err() == nil && !a.executorContainerExists(containerName) {
		return result, output.String(), errors.New("executor_lost")
	}
	return result, output.String(), err
}

func (a *Agent) executorContainerExists(containerName string) bool {
	cmd := exec.Command(a.options.DockerBinary, "inspect", containerName)
	return cmd.Run() == nil
}

func (a *Agent) removeExecutorContainer(containerName string) {
	_ = exec.Command(a.options.DockerBinary, "rm", "-f", containerName).Run()
}

func (a *Agent) executorVolumeWorkspace(workspace string) string {
	hostRoot := strings.TrimSpace(a.options.WorkspaceHostRoot)
	if hostRoot == "" {
		return workspace
	}
	rel, err := filepath.Rel(a.options.WorkspaceRoot, workspace)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return workspace
	}
	return filepath.Join(hostRoot, rel)
}

func writeHookScripts(workspace string, hooks []HookPayload) (map[string][]string, error) {
	output := make(map[string][]string, len(buildHookPhases))
	for _, phase := range buildHookPhases {
		output[phase] = []string{}
	}
	if len(hooks) == 0 {
		return output, nil
	}
	hookDir := filepath.Join(workspace, "hooks")
	if err := os.MkdirAll(hookDir, 0o700); err != nil {
		return output, err
	}
	for _, hook := range hooks {
		hookID := strings.TrimSpace(hook.ID)
		if hookID == "" || strings.TrimSpace(hook.Script) == "" {
			continue
		}
		phase := strings.TrimSpace(hook.Phase)
		if !isBuildHookPhase(phase) {
			continue
		}
		if err := os.WriteFile(filepath.Join(hookDir, hookID+".sh"), []byte(hook.Script), 0o700); err != nil {
			return output, err
		}
		if err := os.WriteFile(filepath.Join(hookDir, hookID+".meta"), []byte(hookMetadataEnv(hook)), 0o600); err != nil {
			return output, err
		}
		output[phase] = append(output[phase], hookID)
	}
	return output, nil
}

func isBuildHookPhase(phase string) bool {
	for _, item := range buildHookPhases {
		if phase == item {
			return true
		}
	}
	return false
}

func hookMetadataEnv(hook HookPayload) string {
	shell := strings.TrimSpace(hook.Shell)
	if shell != "bash" {
		shell = "sh"
	}
	timeoutSeconds := hook.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	return fmt.Sprintf("HOOK_NAME=%s\nHOOK_SHELL=%s\nHOOK_TIMEOUT_SECONDS=%d\nHOOK_FAILURE_POLICY=%s\n",
		shellQuote(hook.Name),
		shellQuote(shell),
		timeoutSeconds,
		shellQuote(hook.FailurePolicy),
	)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func executorContainerName(buildID string) string {
	id := strings.TrimSpace(buildID)
	if id == "" {
		id = "unknown"
	}
	var builder strings.Builder
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	return "liteyuki-devops-buildtask_" + builder.String()
}

func executorScript() string {
	return executorRunScript
}

func defaultAgentID() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "builder-local"
	}
	return "builder-" + strings.ToLower(strings.ReplaceAll(hostname, "_", "-"))
}

func firstLine(content string, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return fallback
}

func boolEnvValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func stringDefault(value string, defaultValue string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func normalizedBuildEnv(values map[string]string) map[string]string {
	output := make(map[string]string)
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if isBuildEnvKey(key) && !strings.HasPrefix(key, "LITEYUKI_") {
			output[key] = value
		}
	}
	return output
}

func isBuildEnvKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, char := range value {
		if index == 0 {
			if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' {
				continue
			}
			return false
		}
		if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}

type concurrentBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func newConcurrentBuffer() *concurrentBuffer {
	return &concurrentBuffer{}
}

func (b *concurrentBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *concurrentBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type logStreamer struct {
	ctx             context.Context
	output          io.Writer
	onLog           func(string) error
	onProgress      func(Progress) error
	onHookLog       func(string, string) error
	onHookComplete  func(string, HookResult) error
	hookLabels      map[string]string
	mu              sync.Mutex
	pending         bytes.Buffer
	lineBuffer      bytes.Buffer
	logLineBuffer   bytes.Buffer
	lastProgressKey string
	done            chan struct{}
	closeOnce       sync.Once
}

func newLogStreamer(ctx context.Context, output io.Writer, hooks []HookPayload, onLog func(string) error, onProgress func(Progress) error, onHookLog func(string, string) error, onHookComplete func(string, HookResult) error) *logStreamer {
	streamer := &logStreamer{
		ctx:            ctx,
		output:         output,
		onLog:          onLog,
		onProgress:     onProgress,
		onHookLog:      onHookLog,
		onHookComplete: onHookComplete,
		hookLabels:     hookLabelsByRunID(hooks),
		done:           make(chan struct{}),
		pending:        bytes.Buffer{},
	}
	go streamer.flushLoop()
	return streamer
}

func (s *logStreamer) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if _, err := s.output.Write(data); err != nil {
		return 0, err
	}
	s.mu.Lock()
	s.consumeProgressLocked(data)
	s.consumeLogLinesLocked(data)
	shouldFlush := s.pending.Len() >= 8192
	s.mu.Unlock()
	if shouldFlush {
		s.flush()
	}
	return len(data), nil
}

func (s *logStreamer) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		s.flushProgressLine()
		s.flushLogLine()
		s.flush()
	})
}

func (s *logStreamer) flushLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.flush()
		case <-s.done:
			return
		case <-s.ctx.Done():
			s.flush()
			return
		}
	}
}

func (s *logStreamer) flush() {
	if s.onLog == nil {
		return
	}
	s.mu.Lock()
	content := s.pending.String()
	s.pending.Reset()
	s.mu.Unlock()
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return
	}
	if err := s.onLog(content); err != nil {
		log.Printf("builder realtime log upload failed: %v", err)
	}
}

func (s *logStreamer) consumeProgressLocked(data []byte) {
	if s.onProgress == nil {
		return
	}
	for _, value := range data {
		if value == '\n' {
			line := s.lineBuffer.String()
			s.lineBuffer.Reset()
			s.emitProgressLineLocked(line)
			continue
		}
		_ = s.lineBuffer.WriteByte(value)
	}
}

func (s *logStreamer) flushProgressLine() {
	if s.onProgress == nil {
		return
	}
	s.mu.Lock()
	line := s.lineBuffer.String()
	s.lineBuffer.Reset()
	s.emitProgressLineLocked(line)
	s.mu.Unlock()
}

func (s *logStreamer) emitProgressLineLocked(line string) {
	_, _ = s.emitHookControlLine(line)
	if s.onProgress == nil {
		return
	}
	progress := buildkitProgressFromRawJSONLine(line)
	if progress.Key == "" || progress.Key == s.lastProgressKey {
		return
	}
	s.lastProgressKey = progress.Key
	if err := s.onProgress(progress); err != nil {
		log.Printf("builder progress upload failed: %v", err)
	}
}

func (s *logStreamer) consumeLogLinesLocked(data []byte) {
	for _, value := range data {
		if value == '\n' {
			line := s.logLineBuffer.String()
			s.logLineBuffer.Reset()
			s.emitBuildLogLineLocked(line)
			continue
		}
		_ = s.logLineBuffer.WriteByte(value)
	}
}

func (s *logStreamer) flushLogLine() {
	s.mu.Lock()
	line := s.logLineBuffer.String()
	s.logLineBuffer.Reset()
	s.emitBuildLogLineLocked(line)
	s.mu.Unlock()
}

func (s *logStreamer) emitBuildLogLineLocked(line string) {
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return
	}
	rendered, control := s.emitHookControlLine(line)
	if control {
		if rendered == "" {
			return
		}
		_, _ = s.pending.WriteString(rendered)
		_ = s.pending.WriteByte('\n')
		return
	}
	_, _ = s.pending.WriteString(line)
	_ = s.pending.WriteByte('\n')
}

func (s *logStreamer) emitHookControlLine(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "::liteyuki-hook-log::") && s.onHookLog != nil {
		parts := strings.SplitN(strings.TrimPrefix(line, "::liteyuki-hook-log::"), "::", 2)
		if len(parts) != 2 {
			return "", true
		}
		content, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", true
		}
		hookLog := string(content)
		if err := s.onHookLog(parts[0], hookLog); err != nil {
			log.Printf("builder hook log upload failed: %v", err)
		}
		return s.formatHookLog(parts[0], hookLog), true
	}
	if strings.HasPrefix(line, "::liteyuki-hook-complete::") && s.onHookComplete != nil {
		parts := strings.SplitN(strings.TrimPrefix(line, "::liteyuki-hook-complete::"), "::", 4)
		if len(parts) != 4 {
			return "", true
		}
		exitCode, _ := strconv.Atoi(parts[2])
		message, _ := base64.StdEncoding.DecodeString(parts[3])
		result := HookResult{
			Succeeded: parts[1] == "true",
			ExitCode:  exitCode,
			Message:   string(message),
		}
		if err := s.onHookComplete(parts[0], result); err != nil {
			log.Printf("builder hook status upload failed: %v", err)
		}
		return s.formatHookLog(parts[0], result.Message), true
	}
	return "", false
}

func (s *logStreamer) formatHookLog(hookRunID string, content string) string {
	content = strings.TrimRight(content, "\n")
	if strings.TrimSpace(content) == "" {
		return ""
	}
	label := s.hookLabels[hookRunID]
	if label == "" {
		label = hookRunID
	}
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		lines[index] = fmt.Sprintf("[%s] %s", label, strings.TrimRight(line, "\r"))
	}
	return strings.Join(lines, "\n")
}

func hookLabelsByRunID(hooks []HookPayload) map[string]string {
	labels := make(map[string]string, len(hooks))
	for _, hook := range hooks {
		hookID := strings.TrimSpace(hook.ID)
		if hookID == "" {
			continue
		}
		phase := strings.TrimSpace(hook.Phase)
		name := strings.TrimSpace(hook.Name)
		switch {
		case phase != "" && name != "":
			labels[hookID] = phase + ": " + name
		case phase != "":
			labels[hookID] = phase
		case name != "":
			labels[hookID] = name
		default:
			labels[hookID] = hookID
		}
	}
	return labels
}

type buildkitRawProgress struct {
	Vertexes []buildkitRawVertex  `json:"vertexes"`
	Statuses []buildkitRawStatus  `json:"statuses"`
	Logs     []buildkitRawLog     `json:"logs"`
	Warnings []buildkitRawWarning `json:"warnings"`
}

type buildkitRawVertex struct {
	Digest    string `json:"digest"`
	Name      string `json:"name"`
	Cached    bool   `json:"cached"`
	Error     string `json:"error"`
	Started   any    `json:"started"`
	Completed any    `json:"completed"`
}

type buildkitRawStatus struct {
	ID        string `json:"id"`
	Vertex    string `json:"vertex"`
	Name      string `json:"name"`
	Started   any    `json:"started"`
	Completed any    `json:"completed"`
}

type buildkitRawLog struct {
	Vertex string `json:"vertex"`
	Stream int    `json:"stream"`
	Data   string `json:"data"`
}

type buildkitRawWarning struct {
	Vertex string   `json:"vertex"`
	Short  string   `json:"short"`
	Detail []string `json:"detail"`
	URL    string   `json:"url"`
}

type Progress struct {
	Key       string `json:"key"`
	Name      string `json:"name,omitempty"`
	Vertex    string `json:"vertex,omitempty"`
	Cached    bool   `json:"cached,omitempty"`
	Started   bool   `json:"started,omitempty"`
	Completed bool   `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

func buildkitProgressFromRawJSONLine(line string) Progress {
	line = strings.TrimSpace(strings.TrimRight(line, "\r"))
	if line == "" {
		return Progress{}
	}
	if !strings.HasPrefix(line, "{") {
		return plainProgressFromLogLine(line)
	}
	var raw buildkitRawProgress
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Progress{}
	}
	for i := len(raw.Statuses) - 1; i >= 0; i-- {
		status := raw.Statuses[i]
		if key := buildkitProgressKey(status.Name); key != "" {
			return Progress{
				Key:       key,
				Name:      status.Name,
				Vertex:    status.Vertex,
				Started:   status.Started != nil,
				Completed: status.Completed != nil,
			}
		}
	}
	for i := len(raw.Vertexes) - 1; i >= 0; i-- {
		vertex := raw.Vertexes[i]
		if key := buildkitProgressKey(vertex.Name); key != "" {
			return Progress{
				Key:       key,
				Name:      vertex.Name,
				Vertex:    vertex.Digest,
				Cached:    vertex.Cached,
				Started:   vertex.Started != nil,
				Completed: vertex.Completed != nil,
				Error:     vertex.Error,
			}
		}
	}
	return Progress{}
}

func plainProgressFromLogLine(line string) Progress {
	value := strings.TrimSpace(strings.TrimRight(line, "\r"))
	if value == "" {
		return Progress{}
	}
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "cloning into"):
		return Progress{Key: "clone_repository", Name: value, Started: true}
	case strings.HasPrefix(lower, "already on ") || strings.HasPrefix(lower, "your branch is up to date"):
		return Progress{Key: "clone_repository", Name: value, Completed: true}
	}
	if !strings.HasPrefix(value, "#") {
		return Progress{}
	}
	name := plainBuildkitStepName(value)
	if name == "" {
		return Progress{}
	}
	key := buildkitProgressKey(name)
	if key == "" {
		return Progress{}
	}
	return Progress{
		Key:       key,
		Name:      name,
		Started:   !strings.Contains(lower, " done") && !strings.Contains(lower, "done "),
		Completed: strings.Contains(lower, " done") || strings.Contains(lower, "done "),
	}
}

func plainBuildkitStepName(line string) string {
	trimmed := strings.TrimSpace(line)
	firstSpace := strings.IndexByte(trimmed, ' ')
	if firstSpace < 0 || firstSpace+1 >= len(trimmed) {
		return ""
	}
	rest := strings.TrimSpace(trimmed[firstSpace+1:])
	if rest == "" || strings.HasPrefix(rest, "...") || strings.HasPrefix(strings.ToUpper(rest), "DONE ") {
		return ""
	}
	if fields := strings.Fields(rest); len(fields) > 0 {
		first := strings.ToUpper(fields[0])
		if strings.HasPrefix(first, "CACHED") || first == "DONE" || first == "ERROR" {
			return ""
		}
	}
	return rest
}

func buildkitProgressKey(name string) string {
	value := strings.ToLower(strings.TrimSpace(name))
	if value == "" {
		return ""
	}
	switch {
	case strings.Contains(value, "[auth]"):
		return "registry_auth"
	case strings.Contains(value, "load build definition"):
		return "load_dockerfile"
	case strings.Contains(value, "load metadata for"):
		return "pull_image_metadata"
	case strings.Contains(value, "load build context") || strings.Contains(value, "transferring context"):
		return "upload_build_context"
	case strings.Contains(value, "resolve image config") || strings.HasPrefix(value, "from "):
		return "pull_base_image"
	case strings.HasPrefix(value, "run ") || strings.Contains(value, " run "):
		return "run_command"
	case strings.Contains(value, "exporting to image") || strings.Contains(value, "exporting manifest") || strings.Contains(value, "exporting config"):
		return "export_image"
	case strings.Contains(value, "pushing layers"):
		return "push_image_layers"
	case strings.Contains(value, "pushing manifest"):
		return "push_image_manifest"
	default:
		return ""
	}
}

type Task struct {
	StreamID           string            `json:"-"`
	JobID              string            `json:"jobId"`
	LeaseToken         string            `json:"leaseToken"`
	LeaseUntil         time.Time         `json:"leaseUntil"`
	TargetBuilder      string            `json:"targetBuilder"`
	BuildRunID         string            `json:"buildRunId"`
	ProjectID          string            `json:"projectId"`
	ApplicationID      string            `json:"applicationId"`
	DeploymentTargetID string            `json:"deploymentTargetId"`
	Repository         RepositoryPayload `json:"repository"`
	Build              BuildPayload      `json:"build"`
	Registry           RegistryPayload   `json:"registry"`
}

type RepositoryPayload struct {
	CloneURL     string `json:"cloneUrl"`
	Owner        string `json:"owner"`
	Repo         string `json:"repo"`
	SourceBranch string `json:"sourceBranch"`
	SourceTag    string `json:"sourceTag"`
	SourceCommit string `json:"sourceCommit"`
	AccessToken  string `json:"accessToken"`
}

type BuildPayload struct {
	DockerfilePath string            `json:"dockerfilePath"`
	BuildContext   string            `json:"buildContext"`
	BuildDirectory string            `json:"buildDirectory"`
	Env            map[string]string `json:"env"`
	Hooks          []HookPayload     `json:"hooks"`
}

type HookPayload struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Phase          string `json:"phase"`
	Script         string `json:"script"`
	Shell          string `json:"shell"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	FailurePolicy  string `json:"failurePolicy"`
}

type HookResult struct {
	Succeeded bool   `json:"succeeded"`
	ExitCode  int    `json:"exitCode"`
	Message   string `json:"message"`
}

type RegistryPayload struct {
	Endpoint         string `json:"endpoint"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	ImageRef         string `json:"imageRef"`
	ImageNamePrefix  string `json:"imageNamePrefix"`
	ImageTagTemplate string `json:"imageTagTemplate"`
}

type Result struct {
	ImageRef          string `json:"imageRef"`
	ImageDigest       string `json:"imageDigest"`
	SourceCommit      string `json:"sourceCommit"`
	SourceAuthorName  string `json:"sourceAuthorName"`
	SourceAuthorEmail string `json:"sourceAuthorEmail"`
	Message           string `json:"message"`
}
