package worker

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/tasks"
)

func TestNewRunnerDefaultsBuildJobOptions(t *testing.T) {
	runner := NewRunner(nil, Options{})
	if runner.deployRolloutTimeoutSeconds != 600 {
		t.Fatalf("deployRolloutTimeoutSeconds = %d", runner.deployRolloutTimeoutSeconds)
	}
	if runner.certManagerClusterIssuer != "letsencrypt-http01" {
		t.Fatalf("certManagerClusterIssuer = %q", runner.certManagerClusterIssuer)
	}
}

func TestNewRunnerUsesBuildJobOptions(t *testing.T) {
	runner := NewRunner(nil, Options{
		DeployRolloutTimeoutSeconds: 120,
		CertManagerClusterIssuer:    "letsencrypt-staging",
	})
	if runner.deployRolloutTimeoutSeconds != 120 {
		t.Fatalf("deployRolloutTimeoutSeconds = %d", runner.deployRolloutTimeoutSeconds)
	}
	if runner.certManagerClusterIssuer != "letsencrypt-staging" {
		t.Fatalf("certManagerClusterIssuer = %q", runner.certManagerClusterIssuer)
	}
}

func TestPeriodicTaskSpecsIncludeGitRefresh(t *testing.T) {
	specs, err := periodicTaskSpecs()
	if err != nil {
		t.Fatalf("periodicTaskSpecs returned error: %v", err)
	}
	found := false
	for _, spec := range specs {
		if spec.Task.Type() == tasks.TypeGitAccountRefresh {
			found = spec.Cron == "@every 5m" && spec.Queue == tasks.QueueLight
		}
	}
	if !found {
		t.Fatalf("specs = %#v", specs)
	}
}

func TestTaskEnvelopeFromPayloadReadsEnvelope(t *testing.T) {
	task, err := tasks.NewDeployRunTask(tasks.DeployRunPayload{ReleaseID: "rel_1", ProjectID: "prj_1", ActorID: "usr_1"})
	if err != nil {
		t.Fatalf("NewDeployRunTask returned error: %v", err)
	}
	envelope := taskEnvelopeFromPayload(task.Type(), task.Payload())
	if envelope.TaskType != tasks.TypeDeployRun || envelope.ResourceRef != "rel_1" || envelope.ActorID != "usr_1" {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestTaskEnvelopeFromPayloadFallsBackForLegacyPayload(t *testing.T) {
	envelope := taskEnvelopeFromPayload(tasks.TypeSyncStatus, []byte("{}"))
	if envelope.TaskType != tasks.TypeSyncStatus || envelope.TaskID != tasks.TypeSyncStatus || envelope.DedupeKey != tasks.TypeSyncStatus {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestProjectNamespaceUsesProjectSlug(t *testing.T) {
	got := projectNamespace(model.Project{Slug: "Demo_App"})
	if got != "project-demo-app" {
		t.Fatalf("namespace = %q", got)
	}
}

func TestProjectNamespaceCapsDNSLabelLength(t *testing.T) {
	got := projectNamespace(model.Project{Slug: strings.Repeat("a", 80)})
	if len(got) > 63 {
		t.Fatalf("namespace too long: %q", got)
	}
}

func TestDeploymentNamespaceUsesEnvironmentNamespace(t *testing.T) {
	got := deploymentNamespace(model.Project{Slug: "demo"}, model.Environment{Namespace: " Prod_App "})
	if got != "prod-app" {
		t.Fatalf("namespace = %q", got)
	}
}

func TestApplicationResourceNameUsesApplicationAndEnvironmentSlug(t *testing.T) {
	got := applicationResourceName(model.Application{Slug: "api_server"}, model.Environment{Slug: "Prod"})
	if got != "api-server-prod" {
		t.Fatalf("resource name = %q", got)
	}
}

func TestGatewayIngressSpecTargetsApplicationService(t *testing.T) {
	spec := gatewayIngressSpec(
		model.GatewayRoute{ID: "gwr_ABC_123", Host: "api.example.com", Path: "api", ServicePort: 8080, TLSMode: "http-challenge"},
		model.Project{ID: "prj_demo"},
		model.Application{Slug: "api"},
		model.Environment{Slug: "dev"},
		"project-demo",
	)
	if spec.Name != "liteyuki-gateway-gwr-abc-123" || spec.ServiceName != "api-dev" || spec.Path != "api" {
		t.Fatalf("spec = %#v", spec)
	}
	if spec.TLSSecretName != "tls-api-example-com" {
		t.Fatalf("tls secret = %q", spec.TLSSecretName)
	}
}

func TestGatewayIngressSpecOmitsTLSForHTTPOnly(t *testing.T) {
	spec := gatewayIngressSpec(
		model.GatewayRoute{ID: "gwr_1", Host: "api.example.com", TLSMode: "http-only"},
		model.Project{ID: "prj_demo"},
		model.Application{Slug: "api", ServicePort: 3000},
		model.Environment{Slug: "dev"},
		"project-demo",
	)
	if spec.TLSSecretName != "" || spec.ServicePort != 3000 {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestGatewayCertificateSpecUsesRouteTLSSecret(t *testing.T) {
	spec := gatewayCertificateSpec(
		model.GatewayRoute{ID: "gwr_1", Host: "api.example.com", TLSMode: "http-challenge"},
		model.Project{ID: "prj_demo"},
		"project-demo",
		"letsencrypt-staging",
	)
	if spec.Name != "liteyuki-gateway-gwr-1" || spec.SecretName != "tls-api-example-com" || spec.ClusterIssuer != "letsencrypt-staging" {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestGatewayDNSStatusVerifiesCNAME(t *testing.T) {
	runner := NewRunner(nil, Options{})
	runner.dnsResolver = fakeCNameResolver{cname: "gateway.example.com."}

	status := runner.gatewayDNSStatus(context.Background(), model.GatewayRoute{Host: "app.example.com", CNAMETarget: "gateway.example.com"})
	if status != "verified" {
		t.Fatalf("status = %q", status)
	}
}

func TestGatewayDNSStatusFailsOnMismatch(t *testing.T) {
	runner := NewRunner(nil, Options{})
	runner.dnsResolver = fakeCNameResolver{err: fmt.Errorf("not found")}

	status := runner.gatewayDNSStatus(context.Background(), model.GatewayRoute{Host: "app.example.com", CNAMETarget: "gateway.example.com"})
	if status != "failed" {
		t.Fatalf("status = %q", status)
	}
}

func TestParseKeyValueMapSupportsJSONObject(t *testing.T) {
	got, err := parseKeyValueMap(`{"APP_ENV":"prod","REPLICAS":2}`)
	if err != nil {
		t.Fatalf("parseKeyValueMap returned error: %v", err)
	}
	if got["APP_ENV"] != "prod" || got["REPLICAS"] != "2" {
		t.Fatalf("values = %#v", got)
	}
}

type fakeCNameResolver struct {
	cname string
	err   error
}

func (r fakeCNameResolver) LookupCNAME(context.Context, string) (string, error) {
	return r.cname, r.err
}

func TestParseKeyValueMapSupportsEnvLines(t *testing.T) {
	got, err := parseKeyValueMap("APP_ENV=prod\n# comment\nLOG_LEVEL=info")
	if err != nil {
		t.Fatalf("parseKeyValueMap returned error: %v", err)
	}
	if got["APP_ENV"] != "prod" || got["LOG_LEVEL"] != "info" {
		t.Fatalf("values = %#v", got)
	}
}

func TestApplicationResourcesSpecAppliesDefaults(t *testing.T) {
	spec, err := applicationResourcesSpec(
		model.Release{ImageRef: "registry.example.com/acme/api:v1"},
		model.Project{ID: "prj_demo", Slug: "demo"},
		model.Application{ID: "app_api", Slug: "api"},
		model.Environment{ID: "env_dev", Slug: "dev", EnvVars: `{"APP_ENV":"dev"}`, ConfigRefs: "LOG_LEVEL=debug", SecretRefs: "TOKEN=secret"},
		"project-demo",
		120,
	)
	if err != nil {
		t.Fatalf("applicationResourcesSpec returned error: %v", err)
	}
	if spec.Name != "api-dev" || spec.Namespace != "project-demo" || spec.ServicePort != 8080 || spec.Replicas != 1 || spec.RolloutTimeoutSeconds != 120 {
		t.Fatalf("spec defaults = %#v", spec)
	}
	if spec.ConfigData["APP_ENV"] != "dev" || spec.ConfigData["LOG_LEVEL"] != "debug" || spec.SecretData["TOKEN"] != "secret" {
		t.Fatalf("spec data = config:%#v secret:%#v", spec.ConfigData, spec.SecretData)
	}
}

func TestReleaseFinishUpdatesIncludesTerminalFields(t *testing.T) {
	finishedAt := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	updates := releaseFinishUpdates("succeeded", "rollout completed", finishedAt)
	if updates["status"] != "succeeded" || updates["message"] != "rollout completed" {
		t.Fatalf("updates = %#v", updates)
	}
	gotFinishedAt, ok := updates["finished_at"].(*time.Time)
	if !ok || !gotFinishedAt.Equal(finishedAt) {
		t.Fatalf("finished_at = %#v", updates["finished_at"])
	}
}

func TestGitAccountDueForWorkerRefresh(t *testing.T) {
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	soon := now.Add(4 * time.Minute)
	later := now.Add(10 * time.Minute)
	if !gitAccountDueForWorkerRefresh(model.GitAccount{Status: "connected", RefreshTokenRef: "secret", ExpiresAt: &soon}, now) {
		t.Fatal("expected account expiring soon to be due")
	}
	if gitAccountDueForWorkerRefresh(model.GitAccount{Status: "connected", RefreshTokenRef: "secret", ExpiresAt: &later}, now) {
		t.Fatal("expected account outside refresh window to be skipped")
	}
	if gitAccountDueForWorkerRefresh(model.GitAccount{Status: "expired", RefreshTokenRef: "secret", ExpiresAt: &soon}, now) {
		t.Fatal("expected expired account to be skipped")
	}
}

func TestGitAccountDueForWorkerRefreshSkipsAfterSuccessfulRefresh(t *testing.T) {
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	refreshedExpiry := now.Add(1 * time.Hour)
	account := model.GitAccount{Status: "connected", RefreshTokenRef: "secret", ExpiresAt: &refreshedExpiry}
	if gitAccountDueForWorkerRefresh(account, now) {
		t.Fatal("expected refreshed account to be skipped on replay")
	}
}
