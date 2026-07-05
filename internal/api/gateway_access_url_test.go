package api

import (
	"testing"

	"github.com/LiteyukiStudio/devops/internal/model"
)

func TestGatewayRouteAccessURLUsesPublicScheme(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "/admin", TLSMode: "http-only"}

	if got := gatewayRouteAccessURL(route, "https", 443); got != "https://app.example.com/admin" {
		t.Fatalf("access url = %q", got)
	}
}

func TestGatewayRouteAccessURLNormalizesPathAndScheme(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "admin"}

	if got := gatewayRouteAccessURL(route, "ftp", 80); got != "http://app.example.com/admin" {
		t.Fatalf("access url = %q", got)
	}
}

func TestGatewayRouteAccessURLOmitsRootPath(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "/"}

	if got := gatewayRouteAccessURL(route, "https", 443); got != "https://app.example.com" {
		t.Fatalf("access url = %q", got)
	}
}

func TestGatewayRouteAccessURLShowsNonStandardPublicPort(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "/"}

	if got := gatewayRouteAccessURL(route, "https", 9443); got != "https://app.example.com:9443" {
		t.Fatalf("access url = %q", got)
	}
	if got := gatewayRouteAccessURL(route, "http", 8080); got != "http://app.example.com:8080" {
		t.Fatalf("access url = %q", got)
	}
}

func TestNormalizeGatewayHostUsesClusterRootDomain(t *testing.T) {
	h := &Handlers{configs: &configCache{values: map[string]string{}}}
	cluster := model.RuntimeCluster{GatewayRootDomain: "Apps.Example.Com."}

	if got := h.normalizeGatewayHost("demo", cluster, ""); got != "demo.apps.example.com" {
		t.Fatalf("host = %q", got)
	}
}

func TestNormalizeGatewayHostUsesSelectedDomainSuffix(t *testing.T) {
	h := &Handlers{configs: &configCache{values: map[string]string{}}}
	cluster := model.RuntimeCluster{GatewayDomainSuffixesRaw: "apps.example.com\ninternal.example.com"}

	if got := h.normalizeGatewayHost("demo", cluster, "internal.example.com"); got != "demo.internal.example.com" {
		t.Fatalf("host = %q", got)
	}
}

func TestGatewayRootDomainFallsBackToLegacyConfig(t *testing.T) {
	h := &Handlers{configs: &configCache{values: map[string]string{"gateway.rootDomain": "legacy.example.com"}}}

	if got := h.gatewayRootDomain(model.RuntimeCluster{}); got != "legacy.example.com" {
		t.Fatalf("root domain = %q", got)
	}
}

func TestNormalizeGatewayDomainSuffixesUsesExplicitValuesOnly(t *testing.T) {
	got := normalizeGatewayDomainSuffixes([]string{"Apps.Example.Com.", "internal.example.com", "apps.example.com"}, "legacy.example.com", "fallback.example.com")
	want := []string{"apps.example.com", "internal.example.com"}
	if len(got) != len(want) {
		t.Fatalf("suffixes = %#v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("suffixes = %#v", got)
		}
	}
}
