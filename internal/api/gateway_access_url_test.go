package api

import (
	"testing"

	"github.com/LiteyukiStudio/devops/internal/model"
)

func TestGatewayRouteAccessURLUsesPublicScheme(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "/admin", TLSMode: "http-only"}

	if got := gatewayRouteAccessURL(route, "https"); got != "https://app.example.com/admin" {
		t.Fatalf("access url = %q", got)
	}
}

func TestGatewayRouteAccessURLNormalizesPathAndScheme(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "admin"}

	if got := gatewayRouteAccessURL(route, "ftp"); got != "http://app.example.com/admin" {
		t.Fatalf("access url = %q", got)
	}
}

func TestGatewayRouteAccessURLOmitsRootPath(t *testing.T) {
	route := model.GatewayRoute{Host: "app.example.com", Path: "/"}

	if got := gatewayRouteAccessURL(route, "https"); got != "https://app.example.com" {
		t.Fatalf("access url = %q", got)
	}
}
