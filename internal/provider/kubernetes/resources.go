package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceListOptions struct {
	Kind          string
	Namespace     string
	ProjectID     string
	ApplicationID string
	EnvironmentID string
}

type ResourceSnapshot struct {
	ID            string            `json:"id"`
	Kind          string            `json:"kind"`
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	Status        string            `json:"status"`
	Summary       string            `json:"summary"`
	ProjectID     string            `json:"projectId"`
	ApplicationID string            `json:"applicationId"`
	EnvironmentID string            `json:"environmentId"`
	ReleaseID     string            `json:"releaseId"`
	RouteID       string            `json:"routeId"`
	Labels        map[string]string `json:"labels"`
	CreatedAt     time.Time         `json:"createdAt"`
}

func (c *Client) ListManagedResources(ctx context.Context, options ResourceListOptions) ([]ResourceSnapshot, error) {
	switch normalizeResourceKind(options.Kind) {
	case "namespaces":
		return c.listManagedNamespaces(ctx, options)
	case "workloads":
		return c.listManagedWorkloads(ctx, options)
	case "services":
		return c.listManagedServicesAndIngresses(ctx, options)
	case "configs":
		return c.listManagedConfigs(ctx, options)
	case "storage":
		return c.listManagedStorage(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", options.Kind)
	}
}

func (c *Client) listManagedNamespaces(ctx context.Context, options ResourceListOptions) ([]ResourceSnapshot, error) {
	list, err := c.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: managedResourceSelector(options)})
	if err != nil {
		return nil, err
	}
	items := make([]ResourceSnapshot, 0, len(list.Items))
	for _, item := range list.Items {
		if !matchesResourceOptions(item.Labels, options) {
			continue
		}
		items = append(items, snapshotFromMeta("Namespace", item.ObjectMeta, "", item.Status.Phase, ""))
	}
	return items, nil
}

func (c *Client) listManagedWorkloads(ctx context.Context, options ResourceListOptions) ([]ResourceSnapshot, error) {
	selector := managedResourceSelector(options)
	deployments, err := c.client.AppsV1().Deployments(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	pods, err := c.client.CoreV1().Pods(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	items := make([]ResourceSnapshot, 0, len(deployments.Items)+len(pods.Items))
	for _, item := range deployments.Items {
		items = append(items, deploymentSnapshot(item))
	}
	for _, item := range pods.Items {
		items = append(items, podSnapshot(item))
	}
	return items, nil
}

func (c *Client) listManagedServicesAndIngresses(ctx context.Context, options ResourceListOptions) ([]ResourceSnapshot, error) {
	selector := managedResourceSelector(options)
	services, err := c.client.CoreV1().Services(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	ingresses, err := c.client.NetworkingV1().Ingresses(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	items := make([]ResourceSnapshot, 0, len(services.Items)+len(ingresses.Items))
	for _, item := range services.Items {
		items = append(items, serviceSnapshot(item))
	}
	for _, item := range ingresses.Items {
		items = append(items, ingressSnapshot(item))
	}
	return items, nil
}

func (c *Client) listManagedConfigs(ctx context.Context, options ResourceListOptions) ([]ResourceSnapshot, error) {
	selector := managedResourceSelector(options)
	configMaps, err := c.client.CoreV1().ConfigMaps(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	secrets, err := c.client.CoreV1().Secrets(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	items := make([]ResourceSnapshot, 0, len(configMaps.Items)+len(secrets.Items))
	for _, item := range configMaps.Items {
		items = append(items, snapshotFromMeta("ConfigMap", item.ObjectMeta, "", fmt.Sprintf("%d keys", len(item.Data)), ""))
	}
	for _, item := range secrets.Items {
		items = append(items, snapshotFromMeta("Secret", item.ObjectMeta, "", string(item.Type), "data hidden"))
	}
	return items, nil
}

func (c *Client) listManagedStorage(ctx context.Context, options ResourceListOptions) ([]ResourceSnapshot, error) {
	claims, err := c.client.CoreV1().PersistentVolumeClaims(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: managedResourceSelector(options)})
	if err != nil {
		return nil, err
	}
	items := make([]ResourceSnapshot, 0, len(claims.Items))
	for _, item := range claims.Items {
		items = append(items, snapshotFromMeta("PersistentVolumeClaim", item.ObjectMeta, "", item.Status.Phase, pvcSummary(item)))
	}
	return items, nil
}

func deploymentSnapshot(item appsv1.Deployment) ResourceSnapshot {
	desired := int32(0)
	if item.Spec.Replicas != nil {
		desired = *item.Spec.Replicas
	}
	status := "progressing"
	if item.Status.ReadyReplicas >= desired && item.Status.AvailableReplicas >= desired {
		status = "ready"
	}
	return snapshotFromMeta("Deployment", item.ObjectMeta, "", status, fmt.Sprintf("ready %d/%d", item.Status.ReadyReplicas, desired))
}

func podSnapshot(item corev1.Pod) ResourceSnapshot {
	ready := 0
	for _, condition := range item.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			ready = 1
			break
		}
	}
	return snapshotFromMeta("Pod", item.ObjectMeta, "", item.Status.Phase, fmt.Sprintf("ready %d/1", ready))
}

func serviceSnapshot(item corev1.Service) ResourceSnapshot {
	ports := make([]string, 0, len(item.Spec.Ports))
	for _, port := range item.Spec.Ports {
		ports = append(ports, strconv.Itoa(int(port.Port)))
	}
	return snapshotFromMeta("Service", item.ObjectMeta, "", string(item.Spec.Type), strings.Join(ports, ", "))
}

func ingressSnapshot(item networkingv1.Ingress) ResourceSnapshot {
	hosts := make([]string, 0, len(item.Spec.Rules))
	for _, rule := range item.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	return snapshotFromMeta("Ingress", item.ObjectMeta, "", "active", strings.Join(hosts, ", "))
}

func pvcSummary(item corev1.PersistentVolumeClaim) string {
	storage := item.Status.Capacity.Storage()
	if storage == nil || storage.IsZero() {
		if item.Spec.StorageClassName != nil {
			return *item.Spec.StorageClassName
		}
		return ""
	}
	return storage.String()
}

func snapshotFromMeta(kind string, meta metav1.ObjectMeta, namespace string, status any, summary string) ResourceSnapshot {
	labels := cloneLabels(meta.Labels)
	ns := namespace
	if ns == "" {
		ns = meta.Namespace
	}
	return ResourceSnapshot{
		ID:            resourceID(kind, ns, meta.Name),
		Kind:          kind,
		Name:          meta.Name,
		Namespace:     ns,
		Status:        fmt.Sprint(status),
		Summary:       summary,
		ProjectID:     labels[ProjectIDLabel],
		ApplicationID: firstNonEmpty(labels[ApplicationIDLabel], labels[legacyApplicationIDLabel]),
		EnvironmentID: firstNonEmpty(labels[EnvironmentIDLabel], labels[legacyEnvironmentIDLabel]),
		ReleaseID:     labels[ReleaseIDLabel],
		RouteID:       firstNonEmpty(labels[GatewayRouteIDLabel], labels[legacyGatewayRouteIDLabel]),
		Labels:        labels,
		CreatedAt:     meta.CreationTimestamp.Time,
	}
}

func managedResourceSelector(options ResourceListOptions) string {
	parts := []string{ManagedByLabel + "=" + ManagedByValue}
	if options.ProjectID != "" {
		parts = append(parts, ProjectIDLabel+"="+options.ProjectID)
	}
	if options.ApplicationID != "" {
		parts = append(parts, ApplicationIDLabel+"="+options.ApplicationID)
	}
	if options.EnvironmentID != "" {
		parts = append(parts, EnvironmentIDLabel+"="+options.EnvironmentID)
	}
	return strings.Join(parts, ",")
}

func matchesResourceOptions(labels map[string]string, options ResourceListOptions) bool {
	if options.ProjectID != "" && labels[ProjectIDLabel] != options.ProjectID {
		return false
	}
	if options.ApplicationID != "" && firstNonEmpty(labels[ApplicationIDLabel], labels[legacyApplicationIDLabel]) != options.ApplicationID {
		return false
	}
	if options.EnvironmentID != "" && firstNonEmpty(labels[EnvironmentIDLabel], labels[legacyEnvironmentIDLabel]) != options.EnvironmentID {
		return false
	}
	return true
}

func normalizeResourceKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "namespace", "namespaces":
		return "namespaces"
	case "workload", "workloads":
		return "workloads"
	case "service", "services", "ingress", "ingresses":
		return "services"
	case "config", "configs", "secret", "secrets":
		return "configs"
	case "storage", "pvc", "pvcs":
		return "storage"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func resourceID(kind string, namespace string, name string) string {
	if namespace == "" {
		return kind + "/" + name
	}
	return kind + "/" + namespace + "/" + name
}

func cloneLabels(labels map[string]string) map[string]string {
	result := make(map[string]string, len(labels))
	for key, value := range labels {
		result[key] = value
	}
	return result
}
