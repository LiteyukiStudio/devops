package builder

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExecutorContainerName(t *testing.T) {
	got := executorContainerName("bldj_4bef0a3b")
	want := "liteyuki-devops-buildtask_bldj_4bef0a3b"
	if got != want {
		t.Fatalf("executorContainerName() = %q, want %q", got, want)
	}
}

func TestExecutorContainerNameSanitizesBuildID(t *testing.T) {
	got := executorContainerName(" build/job:1 ")
	want := "liteyuki-devops-buildtask_build-job-1"
	if got != want {
		t.Fatalf("executorContainerName() = %q, want %q", got, want)
	}
}

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

func TestLogStreamerRendersHookControlLinesForBuildLog(t *testing.T) {
	var buildLogs []string
	var hookLogs []string
	var hookResults []HookResult
	streamer := newLogStreamer(
		context.Background(),
		&bytes.Buffer{},
		[]HookPayload{{ID: "hrun_1", Name: "hello", Phase: "preBuild"}},
		func(content string) error {
			buildLogs = append(buildLogs, content)
			return nil
		},
		nil,
		func(_ string, content string) error {
			hookLogs = append(hookLogs, content)
			return nil
		},
		func(_ string, result HookResult) error {
			hookResults = append(hookResults, result)
			return nil
		},
	)
	_, _ = streamer.Write([]byte("before\n::liteyuki-hook-log::hrun_1::SGVsbG8gV29ybGQ=\n::liteyuki-hook-complete::hrun_1::true::0::aG9vayBzdWNjZWVkZWQ=\nafter\n"))
	streamer.Close()

	content := strings.Join(buildLogs, "\n")
	if strings.Contains(content, "::liteyuki-hook") {
		t.Fatalf("build log contains raw hook control line: %q", content)
	}
	if !strings.Contains(content, "[preBuild: hello] Hello World") {
		t.Fatalf("build log missing rendered hook log: %q", content)
	}
	if !strings.Contains(content, "[preBuild: hello] hook succeeded") {
		t.Fatalf("build log missing rendered hook complete message: %q", content)
	}
	if len(hookLogs) != 1 || hookLogs[0] != "Hello World" {
		t.Fatalf("hook logs = %#v", hookLogs)
	}
	if len(hookResults) != 1 || !hookResults[0].Succeeded || hookResults[0].ExitCode != 0 {
		t.Fatalf("hook results = %#v", hookResults)
	}
}
