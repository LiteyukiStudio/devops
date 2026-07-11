package builder

import (
	"strings"
	"testing"
)

func TestBuildkitProgressFromRawJSONLineReadsVertex(t *testing.T) {
	line := `{"vertexes":[{"digest":"sha256:1","name":"RUN pnpm install --frozen-lockfile","started":"2026-06-08T10:00:00Z"}]}`
	got := buildkitProgressFromRawJSONLine(line)
	if got.Key != "run_command" || got.Name != "RUN pnpm install --frozen-lockfile" || !got.Started {
		t.Fatalf("progress = %#v", got)
	}
}

func TestBuildkitProgressFromRawJSONLineReadsStatus(t *testing.T) {
	line := `{"statuses":[{"id":"context","vertex":"sha256:2","name":"transferring context: 24.1MB","current":24100000,"total":30100000}]}`
	got := buildkitProgressFromRawJSONLine(line)
	if got.Key != "upload_build_context" || got.Vertex != "sha256:2" {
		t.Fatalf("progress = %#v", got)
	}
}

func TestBuildkitProgressFromRawJSONLineIgnoresPlainLogs(t *testing.T) {
	if got := buildkitProgressFromRawJSONLine("plain application output"); got.Key != "" {
		t.Fatalf("progress = %#v", got)
	}
}

func TestBuildkitProgressFromPlainLogLineReadsClone(t *testing.T) {
	got := buildkitProgressFromRawJSONLine("Cloning into 'source'...")
	if got.Key != "clone_repository" || !got.Started {
		t.Fatalf("progress = %#v", got)
	}
}

func TestBuildkitProgressFromPlainLogLineReadsBuildStep(t *testing.T) {
	got := buildkitProgressFromRawJSONLine("#1 [internal] load build definition from Dockerfile")
	if got.Key != "load_dockerfile" || got.Name != "[internal] load build definition from Dockerfile" {
		t.Fatalf("progress = %#v", got)
	}
}

func TestBuildkitProgressFromPlainLogLineReadsAuth(t *testing.T) {
	got := buildkitProgressFromRawJSONLine("#3 [auth] library/alpine:pull token for registry-1.docker.io")
	if got.Key != "registry_auth" {
		t.Fatalf("progress = %#v", got)
	}
}

func TestBuildkitProgressFromPlainLogLineIgnoresDoneOnly(t *testing.T) {
	if got := buildkitProgressFromRawJSONLine("#1 DONE 0.1s"); got.Key != "" {
		t.Fatalf("progress = %#v", got)
	}
}

func TestHandleHookControlLineRendersAndDispatchesHookEvents(t *testing.T) {
	var hookLogs []string
	var hookResults []HookResult
	hookLabels := HookLabelsByRunID([]HookPayload{{ID: "hrun_1", Name: "hello", Phase: "preBuild"}})

	renderedLog, isLogControl := HandleHookControlLine(
		"::luna-devops-hook-log::hrun_1::SGVsbG8gV29ybGQ=",
		hookLabels,
		func(_ string, content string) error {
			hookLogs = append(hookLogs, content)
			return nil
		},
		func(_ string, result HookResult) error {
			hookResults = append(hookResults, result)
			return nil
		},
	)

	renderedComplete, isCompleteControl := HandleHookControlLine(
		"::luna-devops-hook-complete::hrun_1::true::0::aG9vayBzdWNjZWVkZWQ=",
		hookLabels,
		func(_ string, content string) error {
			hookLogs = append(hookLogs, content)
			return nil
		},
		func(_ string, result HookResult) error {
			hookResults = append(hookResults, result)
			return nil
		},
	)

	if !isLogControl || !isCompleteControl {
		t.Fatalf("expected hook control lines to be recognized")
	}
	if strings.Contains(renderedLog, "::luna-devops-hook") || strings.Contains(renderedComplete, "::luna-devops-hook") {
		t.Fatalf("rendered control output leaked raw hook markers: %q %q", renderedLog, renderedComplete)
	}
	if renderedLog != "[preBuild: hello] Hello World" {
		t.Fatalf("rendered hook log = %q", renderedLog)
	}
	if renderedComplete != "[preBuild: hello] hook succeeded" {
		t.Fatalf("rendered hook complete = %q", renderedComplete)
	}
	if len(hookLogs) != 1 || hookLogs[0] != "Hello World" {
		t.Fatalf("hook logs = %#v", hookLogs)
	}
	if len(hookResults) != 1 || !hookResults[0].Succeeded || hookResults[0].ExitCode != 0 {
		t.Fatalf("hook results = %#v", hookResults)
	}
}
