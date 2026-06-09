package kubernetes

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestApplyGatewayIngressCreatesIngress(t *testing.T) {
	client := NewClientForInterface(fake.NewSimpleClientset())
	spec := GatewayIngressSpec{
		Name:          "api-dev",
		Namespace:     "project-demo",
		ProjectID:     "prj_demo",
		ApplicationID: "app_api",
		EnvironmentID: "env_dev",
		RouteID:       "gwr_api",
		Host:          "api.example.com",
		Path:          "api",
		ServiceName:   "api-dev",
		ServicePort:   8080,
		TLSSecretName: "api-dev-tls",
	}

	if err := client.ApplyGatewayIngress(context.Background(), spec); err != nil {
		t.Fatalf("ApplyGatewayIngress returned error: %v", err)
	}

	ingress, err := client.client.NetworkingV1().Ingresses(spec.Namespace).Get(context.Background(), spec.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get ingress: %v", err)
	}
	if ingress.Spec.Rules[0].Host != spec.Host {
		t.Fatalf("host = %q", ingress.Spec.Rules[0].Host)
	}
	path := ingress.Spec.Rules[0].HTTP.Paths[0]
	if path.Path != "/api" || path.Backend.Service.Name != spec.ServiceName || path.Backend.Service.Port.Number != spec.ServicePort {
		t.Fatalf("path = %#v", path)
	}
	if len(ingress.Spec.TLS) != 1 || ingress.Spec.TLS[0].SecretName != spec.TLSSecretName {
		t.Fatalf("tls = %#v", ingress.Spec.TLS)
	}
	expectedLabels := map[string]string{
		ManagedByLabel:            ManagedByValue,
		ApplicationNameKey:        spec.ServiceName,
		ProjectIDLabel:            spec.ProjectID,
		ApplicationIDLabel:        spec.ApplicationID,
		EnvironmentIDLabel:        spec.EnvironmentID,
		GatewayRouteIDLabel:       spec.RouteID,
		legacyGatewayRouteIDLabel: spec.RouteID,
	}
	for key, value := range expectedLabels {
		if ingress.Labels[key] != value {
			t.Fatalf("label %s = %q, want %q in %#v", key, ingress.Labels[key], value, ingress.Labels)
		}
	}
}

func TestApplyGatewayIngressUpdatesExistingIngress(t *testing.T) {
	client := NewClientForInterface(fake.NewSimpleClientset())
	spec := GatewayIngressSpec{
		Name:        "api-dev",
		Namespace:   "project-demo",
		ProjectID:   "prj_demo",
		RouteID:     "gwr_api",
		Host:        "api.example.com",
		Path:        "/",
		ServiceName: "api-dev",
		ServicePort: 8080,
	}
	if err := client.ApplyGatewayIngress(context.Background(), spec); err != nil {
		t.Fatalf("ApplyGatewayIngress returned error: %v", err)
	}
	spec.ServicePort = 3000
	if err := client.ApplyGatewayIngress(context.Background(), spec); err != nil {
		t.Fatalf("ApplyGatewayIngress returned error on replay: %v", err)
	}

	list, err := client.client.NetworkingV1().Ingresses(spec.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list ingresses: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("ingress count = %d", len(list.Items))
	}
	port := list.Items[0].Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number
	if port != 3000 {
		t.Fatalf("service port = %d", port)
	}
}
