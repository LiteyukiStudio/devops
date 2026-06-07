package security

import (
	"errors"
	"testing"
)

func TestPublicEgressPolicyBlocksPrivateAndSpecialIPs(t *testing.T) {
	policy := PublicEgressPolicy()
	blocked := []string{
		"http://0.0.0.0:8080",
		"http://127.0.0.1:8080",
		"http://10.0.0.1",
		"http://169.254.169.254",
		"http://[::1]:8080",
		"http://[fc00::1]",
	}

	for _, target := range blocked {
		if _, err := policy.ValidateURL(target); !errors.Is(err, ErrBlockedByPolicy) {
			t.Fatalf("%s should be blocked, got %v", target, err)
		}
	}
}

func TestAdminEgressPolicyAllowsPrivateIPs(t *testing.T) {
	policy := AdminEgressPolicy()

	if _, err := policy.ValidateURL("http://127.0.0.1:8080"); err != nil {
		t.Fatalf("admin policy should allow loopback: %v", err)
	}
}

func TestEgressPolicySupportsDomainAndPortControls(t *testing.T) {
	policy := PublicEgressPolicy()
	policy.ApplyIPFilterToNames = false
	policy.DomainBlockList = []string{"evil.test"}
	policy.AllowedPorts = []int{443}

	if _, err := policy.ValidateURL("https://api.example.com/v1"); err != nil {
		t.Fatalf("expected allowed domain: %v", err)
	}
	if _, err := policy.ValidateURL("https://evil.test/v1"); !errors.Is(err, ErrBlockedByPolicy) {
		t.Fatalf("expected domain to be blocked, got %v", err)
	}
	if _, err := policy.ValidateURL("http://api.example.com/v1"); !errors.Is(err, ErrBlockedByPolicy) {
		t.Fatalf("expected port to be blocked, got %v", err)
	}
}

func TestDomainAllowListBypassesReservedIPBlock(t *testing.T) {
	policy := PublicEgressPolicy()
	policy.DomainAllowList = []string{"localhost"}

	if _, err := PublicEgressPolicy().ValidateURL("http://localhost:8080"); !errors.Is(err, ErrBlockedByPolicy) {
		t.Fatalf("localhost should be blocked by default, got %v", err)
	}
	if _, err := policy.ValidateURL("http://localhost:8080"); err != nil {
		t.Fatalf("domain allowlist should permit reserved resolved ip: %v", err)
	}
}

func TestIPAllowAndBlockLists(t *testing.T) {
	allowPolicy := PublicEgressPolicy()
	allowPolicy.IPAllowList = []string{"127.0.0.1"}

	if _, err := allowPolicy.ValidateURL("http://127.0.0.1:8080"); err != nil {
		t.Fatalf("ip allowlist should permit reserved direct ip: %v", err)
	}

	blockPolicy := AdminEgressPolicy()
	blockPolicy.IPBlockList = []string{"127.0.0.1"}
	if _, err := blockPolicy.ValidateURL("http://127.0.0.1:8080"); !errors.Is(err, ErrBlockedByPolicy) {
		t.Fatalf("explicit ip blocklist should win, got %v", err)
	}
}

func TestEgressPolicyForRole(t *testing.T) {
	if EgressPolicyForRole("user").AllowPrivateNetwork {
		t.Fatal("normal user should not be allowed to access private network")
	}
	if !EgressPolicyForRole("platform_admin").AllowPrivateNetwork {
		t.Fatal("platform admin should be allowed to access private network")
	}
}
