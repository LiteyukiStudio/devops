package dashboard

import (
	"testing"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
)

func TestAggregateAttentionGroupsConsecutiveFailuresAndStopsAtRecovery(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	events := []model.PlatformEvent{
		{ID: "evt_latest_failure", Type: "build.failed", Category: "build", Severity: "error", Status: "failed", ProjectID: "prj_1", ApplicationID: "app_1", OccurredAt: now},
		{ID: "evt_previous_failure", Type: "build.failed", Category: "build", Severity: "error", Status: "failed", ProjectID: "prj_1", ApplicationID: "app_1", OccurredAt: now.Add(-time.Minute)},
		{ID: "evt_recovered", Type: "build.succeeded", Category: "build", Severity: "info", Status: "succeeded", ProjectID: "prj_1", ApplicationID: "app_1", OccurredAt: now.Add(-2 * time.Minute)},
		{ID: "evt_old_failure", Type: "build.failed", Category: "build", Severity: "error", Status: "failed", ProjectID: "prj_1", ApplicationID: "app_1", OccurredAt: now.Add(-3 * time.Minute)},
	}

	items := aggregateAttention(events, entityReferences{projects: map[string]EntityRef{}, apps: map[string]EntityRef{}, targets: map[string]EntityRef{}}, now.Add(-time.Hour))
	if len(items) != 1 {
		t.Fatalf("attention items = %d, want 1", len(items))
	}
	if items[0].Occurrences != 2 {
		t.Fatalf("occurrences = %d, want 2 consecutive failures", items[0].Occurrences)
	}
}

func TestAggregateAttentionOmitsRecoveredResource(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	events := []model.PlatformEvent{
		{ID: "evt_success", Type: "release.succeeded", Category: "release", Severity: "info", Status: "succeeded", ProjectID: "prj_1", DeploymentTargetID: "dplt_1", OccurredAt: now},
		{ID: "evt_failure", Type: "release.failed", Category: "release", Severity: "error", Status: "failed", ProjectID: "prj_1", DeploymentTargetID: "dplt_1", OccurredAt: now.Add(-time.Minute)},
	}

	items := aggregateAttention(events, entityReferences{projects: map[string]EntityRef{}, apps: map[string]EntityRef{}, targets: map[string]EntityRef{}}, now.Add(-time.Hour))
	if len(items) != 0 {
		t.Fatalf("attention items = %#v, want none after recovery", items)
	}
}

func TestReadinessStatus(t *testing.T) {
	tests := []struct {
		available int
		total     int
		want      string
	}{
		{available: 0, total: 0, want: "unavailable"},
		{available: 0, total: 2, want: "unavailable"},
		{available: 1, total: 2, want: "degraded"},
		{available: 2, total: 2, want: "ready"},
	}
	for _, test := range tests {
		if got := readinessStatus(test.available, test.total); got != test.want {
			t.Fatalf("readinessStatus(%d, %d) = %q, want %q", test.available, test.total, got, test.want)
		}
	}
}
