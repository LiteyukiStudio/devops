package model

import "testing"

func TestApplyPlatformDeploymentTargetDefaultsForGatewayTrafficProbe(t *testing.T) {
	target := ApplyPlatformDeploymentTargetDefaults(
		Project{SystemKey: PlatformSystemProjectKey},
		Application{Identifier: GatewayTrafficProbeApplicationIdentifier},
		DeploymentTarget{},
	)
	if target.ServiceAccountName != GatewayTrafficProbeServiceAccountName {
		t.Fatalf("ServiceAccountName = %q, want %q", target.ServiceAccountName, GatewayTrafficProbeServiceAccountName)
	}
	if target.AutomountServiceAccountToken != GatewayTrafficProbeAutomountServiceToken {
		t.Fatalf("AutomountServiceAccountToken = %q, want %q", target.AutomountServiceAccountToken, GatewayTrafficProbeAutomountServiceToken)
	}
}

func TestApplyPlatformDeploymentTargetDefaultsDoesNotTouchNormalApplication(t *testing.T) {
	target := ApplyPlatformDeploymentTargetDefaults(
		Project{SystemKey: ""},
		Application{Identifier: GatewayTrafficProbeApplicationIdentifier},
		DeploymentTarget{ServiceAccountName: "custom", AutomountServiceAccountToken: "false"},
	)
	if target.ServiceAccountName != "custom" {
		t.Fatalf("ServiceAccountName = %q, want custom", target.ServiceAccountName)
	}
	if target.AutomountServiceAccountToken != "false" {
		t.Fatalf("AutomountServiceAccountToken = %q, want false", target.AutomountServiceAccountToken)
	}
}
