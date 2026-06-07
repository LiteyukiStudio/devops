package kubernetes

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ApplicationResourcesSpec struct {
	Name                  string
	Namespace             string
	ProjectID             string
	ApplicationID         string
	EnvironmentID         string
	Image                 string
	Replicas              int32
	ServicePort           int32
	CPURequest            string
	MemoryRequest         string
	RolloutTimeoutSeconds int32
	ConfigData            map[string]string
	SecretData            map[string]string
}

func (c *Client) ApplyApplicationResources(ctx context.Context, spec ApplicationResourcesSpec) error {
	if err := validateApplicationResourcesSpec(spec); err != nil {
		return err
	}
	labels := appLabels(spec)
	if err := c.applyConfigMap(ctx, spec, labels); err != nil {
		return err
	}
	if err := c.applySecret(ctx, spec, labels); err != nil {
		return err
	}
	if err := c.applyDeployment(ctx, spec, labels); err != nil {
		return err
	}
	return c.applyService(ctx, spec, labels)
}

func validateApplicationResourcesSpec(spec ApplicationResourcesSpec) error {
	if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.Namespace) == "" {
		return fmt.Errorf("application resource name and namespace are required")
	}
	if strings.TrimSpace(spec.Image) == "" {
		return fmt.Errorf("release image is required")
	}
	if spec.ServicePort <= 0 || spec.ServicePort > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535")
	}
	if _, err := resourceRequirements(spec); err != nil {
		return err
	}
	return nil
}

func (c *Client) applyConfigMap(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	item := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: spec.Name + "-config", Namespace: spec.Namespace, Labels: labels}, Data: spec.ConfigData}
	existing, err := c.client.CoreV1().ConfigMaps(spec.Namespace).Get(ctx, item.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.CoreV1().ConfigMaps(spec.Namespace).Create(ctx, item, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = item.Labels
	existing.Data = item.Data
	_, err = c.client.CoreV1().ConfigMaps(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) applySecret(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	data := make(map[string][]byte, len(spec.SecretData))
	for key, value := range spec.SecretData {
		data[key] = []byte(value)
	}
	item := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: spec.Name + "-secret", Namespace: spec.Namespace, Labels: labels}, Type: corev1.SecretTypeOpaque, Data: data}
	existing, err := c.client.CoreV1().Secrets(spec.Namespace).Get(ctx, item.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.CoreV1().Secrets(spec.Namespace).Create(ctx, item, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = item.Labels
	existing.Type = item.Type
	existing.Data = item.Data
	_, err = c.client.CoreV1().Secrets(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) applyDeployment(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	replicas := spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	progressDeadlineSeconds := spec.RolloutTimeoutSeconds
	if progressDeadlineSeconds <= 0 {
		progressDeadlineSeconds = 600
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas:                &replicas,
			Selector:                &metav1.LabelSelector{MatchLabels: labels},
			ProgressDeadlineSeconds: &progressDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  "app",
					Image: spec.Image,
					Ports: []corev1.ContainerPort{{ContainerPort: spec.ServicePort}},
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: spec.Name + "-config"}}},
						{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: spec.Name + "-secret"}}},
					},
					Resources: mustResourceRequirements(spec),
				}}},
			},
		},
	}
	existing, err := c.client.AppsV1().Deployments(spec.Namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.AppsV1().Deployments(spec.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = labels
	existing.Spec = deployment.Spec
	_, err = c.client.AppsV1().Deployments(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) applyService(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       spec.ServicePort,
				TargetPort: intstrFromInt32(spec.ServicePort),
			}},
		},
	}
	existing, err := c.client.CoreV1().Services(spec.Namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.CoreV1().Services(spec.Namespace).Create(ctx, service, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = labels
	existing.Spec.Selector = service.Spec.Selector
	existing.Spec.Ports = service.Spec.Ports
	_, err = c.client.CoreV1().Services(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func appLabels(spec ApplicationResourcesSpec) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "liteyuki-devops",
		"app.kubernetes.io/name":       spec.Name,
		"liteyuki.devops/project-id":   spec.ProjectID,
		"liteyuki.devops/app-id":       spec.ApplicationID,
		"liteyuki.devops/env-id":       spec.EnvironmentID,
	}
}

func resourceRequirements(spec ApplicationResourcesSpec) (corev1.ResourceRequirements, error) {
	requests := corev1.ResourceList{}
	if spec.CPURequest != "" {
		quantity, err := resource.ParseQuantity(spec.CPURequest)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid cpu request: %w", err)
		}
		requests[corev1.ResourceCPU] = quantity
	}
	if spec.MemoryRequest != "" {
		quantity, err := resource.ParseQuantity(spec.MemoryRequest)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid memory request: %w", err)
		}
		requests[corev1.ResourceMemory] = quantity
	}
	return corev1.ResourceRequirements{Requests: requests}, nil
}

func mustResourceRequirements(spec ApplicationResourcesSpec) corev1.ResourceRequirements {
	requirements, err := resourceRequirements(spec)
	if err != nil {
		panic(err)
	}
	return requirements
}

func intstrFromInt32(value int32) intstr.IntOrString {
	return intstr.FromInt(int(value))
}
