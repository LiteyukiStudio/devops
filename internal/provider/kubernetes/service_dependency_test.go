package kubernetes

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestCheckServiceDependencyByPortName(t *testing.T) {
	ready := true
	notReady := false
	portName := "http"
	targetPort := int32(8080)
	clientset := kubefake.NewSimpleClientset(
		serviceDependencyService("ns-target", "dplt-api", portName, 80, targetPort),
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dplt-api-abc",
				Namespace: "ns-target",
				Labels:    map[string]string{discoveryv1.LabelServiceName: "dplt-api"},
			},
			Ports: []discoveryv1.EndpointPort{{Name: &portName, Port: &targetPort}},
			Endpoints: []discoveryv1.Endpoint{
				{Addresses: []string{"10.42.0.10"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
				{Addresses: []string{"10.42.0.11"}, Conditions: discoveryv1.EndpointConditions{Ready: &notReady}},
			},
		},
	)
	client := NewClientForInterface(clientset)

	diagnostic, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		SourceNamespace: "ns-source",
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-api",
		PortName:        portName,
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	if !diagnostic.Service.Exists || !diagnostic.Port.Resolved || diagnostic.Port.Port == nil {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
	if diagnostic.Port.Port.Port != 80 || diagnostic.Port.Port.TargetPort != "8080" {
		t.Fatalf("port = %#v", diagnostic.Port.Port)
	}
	if diagnostic.Endpoints.EndpointCount != 2 || diagnostic.Endpoints.ReadyEndpointCount != 1 || diagnostic.Endpoints.ReadyAddressCount != 1 {
		t.Fatalf("endpoints = %#v", diagnostic.Endpoints)
	}
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckServiceExists, ServiceDependencyCheckPassed)
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckServicePortResolved, ServiceDependencyCheckPassed)
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckEndpointReady, ServiceDependencyCheckPassed)
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckNetworkPolicyDetected, ServiceDependencyCheckPassed)
}

func TestCheckServiceDependencyByPortNumberAndNetworkPolicySummary(t *testing.T) {
	ready := true
	portName := "postgres"
	targetPort := int32(5432)
	client := NewClientForInterface(kubefake.NewSimpleClientset(
		serviceDependencyService("ns-shared", "dplt-postgres", portName, 5432, targetPort),
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dplt-postgres-abc",
				Namespace: "ns-shared",
				Labels:    map[string]string{discoveryv1.LabelServiceName: "dplt-postgres"},
			},
			Ports: []discoveryv1.EndpointPort{{Name: &portName, Port: &targetPort}},
			Endpoints: []discoveryv1.Endpoint{{
				Addresses:  []string{"10.42.0.20"},
				Conditions: discoveryv1.EndpointConditions{Ready: &ready},
			}},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "database-policy", Namespace: "ns-shared"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "postgres"}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
				Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "application-egress", Namespace: "ns-shared"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
				Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
			},
		},
	))

	diagnostic, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		SourceNamespace: "ns-shared",
		TargetNamespace: "ns-shared",
		ServiceName:     "dplt-postgres",
		PortNumber:      5432,
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	if !diagnostic.NetworkPolicies.Detected || diagnostic.NetworkPolicies.PolicyCount != 2 {
		t.Fatalf("network policies = %#v", diagnostic.NetworkPolicies)
	}
	if len(diagnostic.NetworkPolicies.TargetIngress) != 1 || diagnostic.NetworkPolicies.TargetIngress[0].Name != "database-policy" {
		t.Fatalf("target ingress = %#v", diagnostic.NetworkPolicies.TargetIngress)
	}
	if len(diagnostic.NetworkPolicies.SourceEgress) != 1 || diagnostic.NetworkPolicies.SourceEgress[0].Name != "application-egress" {
		t.Fatalf("source egress = %#v", diagnostic.NetworkPolicies.SourceEgress)
	}
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckNetworkPolicyDetected, ServiceDependencyCheckWarning)
}

func TestCheckServiceDependencyReportsMissingService(t *testing.T) {
	client := NewClientForInterface(kubefake.NewSimpleClientset())

	diagnostic, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-missing",
		PortName:        "http",
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	if diagnostic.Service.Exists || diagnostic.Port.Resolved {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckServiceExists, ServiceDependencyCheckFailed)
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckServicePortResolved, ServiceDependencyCheckFailed)
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckEndpointReady, ServiceDependencyCheckFailed)
}

func TestCheckServiceDependencyReportsMissingPort(t *testing.T) {
	client := NewClientForInterface(kubefake.NewSimpleClientset(
		serviceDependencyService("ns-target", "dplt-api", "http", 80, 8080),
	))

	diagnostic, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-api",
		PortName:        "metrics",
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	if !diagnostic.Service.Exists || diagnostic.Port.Resolved || len(diagnostic.Port.Available) != 1 {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckServicePortResolved, ServiceDependencyCheckFailed)
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckEndpointReady, ServiceDependencyCheckFailed)
}

func TestCheckServiceDependencyReportsNoReadyEndpoints(t *testing.T) {
	notReady := false
	portName := "http"
	targetPort := int32(8080)
	client := NewClientForInterface(kubefake.NewSimpleClientset(
		serviceDependencyService("ns-target", "dplt-api", portName, 80, targetPort),
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dplt-api-abc",
				Namespace: "ns-target",
				Labels:    map[string]string{discoveryv1.LabelServiceName: "dplt-api"},
			},
			Ports: []discoveryv1.EndpointPort{{Name: &portName, Port: &targetPort}},
			Endpoints: []discoveryv1.Endpoint{{
				Addresses:  []string{"10.42.0.10"},
				Conditions: discoveryv1.EndpointConditions{Ready: &notReady},
			}},
		},
	))

	diagnostic, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-api",
		PortNumber:      80,
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	if diagnostic.Endpoints.ReadyEndpointCount != 0 {
		t.Fatalf("endpoints = %#v", diagnostic.Endpoints)
	}
	assertDependencyCheckStatus(t, diagnostic, ServiceDependencyCheckEndpointReady, ServiceDependencyCheckFailed)
}

func TestCheckServiceDependencyDefaultsTargetPortAndSummarizesExpressions(t *testing.T) {
	client := NewClientForInterface(kubefake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "dplt-api", Namespace: "ns-target"},
			Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{
				Name: "http",
				Port: 8080,
			}}},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "target-policy", Namespace: "ns-target"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "tier",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"api"},
				}}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		},
	))

	diagnostic, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-api",
		PortNumber:      8080,
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	if diagnostic.Port.Port == nil || diagnostic.Port.Port.TargetPort != "8080" || diagnostic.Port.Port.Protocol != "TCP" {
		t.Fatalf("port = %#v", diagnostic.Port.Port)
	}
	if len(diagnostic.NetworkPolicies.TargetIngress) != 1 || diagnostic.NetworkPolicies.TargetIngress[0].PodSelector != "tier in (api)" {
		t.Fatalf("target ingress = %#v", diagnostic.NetworkPolicies.TargetIngress)
	}
}

func TestCheckServiceDependencyUsesReadOnlyNonSecretKubernetesAPIs(t *testing.T) {
	clientset := kubefake.NewSimpleClientset(serviceDependencyService("ns-target", "dplt-api", "http", 80, 8080))
	clientset.PrependReactor("*", "secrets", func(action ktesting.Action) (bool, runtime.Object, error) {
		t.Fatalf("unexpected Secret API action: %#v", action)
		return true, nil, nil
	})
	client := NewClientForInterface(clientset)

	_, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		SourceNamespace: "ns-source",
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-api",
		PortName:        "http",
	})
	if err != nil {
		t.Fatalf("CheckServiceDependency returned error: %v", err)
	}
	for _, action := range clientset.Actions() {
		if action.GetVerb() != "get" && action.GetVerb() != "list" {
			t.Fatalf("unexpected mutating Kubernetes API action: %s %s", action.GetVerb(), action.GetResource().Resource)
		}
		if action.GetResource().Resource == "secrets" {
			t.Fatalf("unexpected Secret API action: %#v", action)
		}
	}
}

func TestCheckServiceDependencyValidatesInput(t *testing.T) {
	client := NewClientForInterface(kubefake.NewSimpleClientset())
	_, err := client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		TargetNamespace: "Invalid_Namespace",
		ServiceName:     "dplt-api",
		PortNumber:      80,
	})
	if err == nil {
		t.Fatal("expected invalid target namespace to fail")
	}
	_, err = client.CheckServiceDependency(context.Background(), ServiceDependencyCheckOptions{
		TargetNamespace: "ns-target",
		ServiceName:     "dplt-api",
	})
	if err == nil {
		t.Fatal("expected missing port selector to fail")
	}
}

func serviceDependencyService(namespace, name, portName string, servicePort, targetPort int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.10",
			Ports: []corev1.ServicePort{{
				Name:       portName,
				Port:       servicePort,
				TargetPort: intstr.FromInt32(targetPort),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
}

func assertDependencyCheckStatus(t *testing.T, diagnostic ServiceDependencyDiagnostic, code string, expected ServiceDependencyCheckStatus) {
	t.Helper()
	for _, check := range diagnostic.Checks {
		if check.Code != code {
			continue
		}
		if check.Status != expected {
			t.Fatalf("check %s status = %s, want %s", code, check.Status, expected)
		}
		return
	}
	t.Fatalf("check %s not found in %#v", code, diagnostic.Checks)
}
