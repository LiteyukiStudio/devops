package kubernetes

import (
	"context"
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GatewayIngressSpec struct {
	Name          string
	Namespace     string
	ProjectID     string
	RouteID       string
	Host          string
	Path          string
	ServiceName   string
	ServicePort   int32
	TLSSecretName string
}

func (c *Client) ApplyGatewayIngress(ctx context.Context, spec GatewayIngressSpec) error {
	if err := validateGatewayIngressSpec(spec); err != nil {
		return err
	}
	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "liteyuki-devops",
				"liteyuki.devops/project-id":   spec.ProjectID,
				"liteyuki.devops/route-id":     spec.RouteID,
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: spec.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     normalizedIngressPath(spec.Path),
					PathType: &pathType,
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: spec.ServiceName,
						Port: networkingv1.ServiceBackendPort{Number: spec.ServicePort},
					}},
				}}}},
			}},
		},
	}
	if strings.TrimSpace(spec.TLSSecretName) != "" {
		ingress.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{spec.Host}, SecretName: spec.TLSSecretName}}
	}

	existing, err := c.client.NetworkingV1().Ingresses(spec.Namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.NetworkingV1().Ingresses(spec.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = ingress.Labels
	existing.Spec = ingress.Spec
	_, err = c.client.NetworkingV1().Ingresses(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func validateGatewayIngressSpec(spec GatewayIngressSpec) error {
	if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.Namespace) == "" {
		return fmt.Errorf("gateway ingress name and namespace are required")
	}
	if strings.TrimSpace(spec.Host) == "" {
		return fmt.Errorf("gateway host is required")
	}
	if strings.TrimSpace(spec.ServiceName) == "" {
		return fmt.Errorf("gateway service name is required")
	}
	if spec.ServicePort <= 0 || spec.ServicePort > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535")
	}
	return nil
}

func normalizedIngressPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") {
		return "/" + value
	}
	return value
}
