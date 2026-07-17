package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
)

type ApplicationTopologyOptions struct {
	ClusterID          string
	ClusterName        string
	Namespace          string
	ProjectID          string
	ApplicationID      string
	DeploymentTargetID string
}

type ApplicationTopologyNode struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	Status             string `json:"status"`
	Summary            string `json:"summary"`
	ClusterID          string `json:"clusterId"`
	ClusterName        string `json:"clusterName"`
	DeploymentTargetID string `json:"deploymentTargetId"`
}

type ApplicationTopologyEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type ApplicationTopologySnapshot struct {
	Nodes []ApplicationTopologyNode `json:"nodes"`
	Edges []ApplicationTopologyEdge `json:"edges"`
}

type applicationWorkload struct {
	kind     string
	name     string
	nodeID   string
	template corev1.PodTemplateSpec
}

func (c *Client) BuildApplicationTopology(ctx context.Context, options ApplicationTopologyOptions) (ApplicationTopologySnapshot, error) {
	resourceOptions := ResourceListOptions{
		Namespace:          options.Namespace,
		ProjectID:          options.ProjectID,
		ApplicationID:      options.ApplicationID,
		DeploymentTargetID: options.DeploymentTargetID,
	}
	runtimeSelector := managedRuntimeResourceSelector(resourceOptions)
	managedSelector := managedResourceSelector(resourceOptions)

	deployments, err := c.client.AppsV1().Deployments(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: runtimeSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	statefulSets, err := c.client.AppsV1().StatefulSets(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: runtimeSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	replicaSets, err := c.client.AppsV1().ReplicaSets(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: runtimeSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	pods, err := c.client.CoreV1().Pods(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: runtimeSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	services, err := c.client.CoreV1().Services(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: managedSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	configMaps, err := c.client.CoreV1().ConfigMaps(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: managedSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	secrets, err := c.client.CoreV1().Secrets(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: managedSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	claims, err := c.client.CoreV1().PersistentVolumeClaims(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: managedSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}
	hpas, err := c.client.AutoscalingV2().HorizontalPodAutoscalers(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: runtimeSelector})
	if err != nil {
		return ApplicationTopologySnapshot{}, err
	}

	nodes := map[string]ApplicationTopologyNode{}
	edges := map[string]ApplicationTopologyEdge{}
	workloads := make([]applicationWorkload, 0, len(deployments.Items)+len(statefulSets.Items))

	addSnapshot := func(snapshot ResourceSnapshot) string {
		node := topologyNode(snapshot, options)
		nodes[node.ID] = node
		return node.ID
	}
	for i := range deployments.Items {
		item := &deployments.Items[i]
		workloads = append(workloads, applicationWorkload{"Deployment", item.Name, addSnapshot(deploymentSnapshot(*item)), item.Spec.Template})
	}
	for i := range statefulSets.Items {
		item := &statefulSets.Items[i]
		workloads = append(workloads, applicationWorkload{"StatefulSet", item.Name, addSnapshot(statefulSetSnapshot(*item)), item.Spec.Template})
	}
	for i := range services.Items {
		addSnapshot(serviceSnapshot(services.Items[i]))
	}
	for i := range configMaps.Items {
		addSnapshot(snapshotFromMeta("ConfigMap", configMaps.Items[i].ObjectMeta, "", "ready", fmt.Sprintf("%d keys", len(configMaps.Items[i].Data))))
	}
	for i := range secrets.Items {
		// Secret values and key names are deliberately excluded from topology output.
		addSnapshot(snapshotFromMeta("Secret", secrets.Items[i].ObjectMeta, "", "ready", ""))
	}
	for i := range claims.Items {
		addSnapshot(snapshotFromMeta("PersistentVolumeClaim", claims.Items[i].ObjectMeta, "", claims.Items[i].Status.Phase, pvcSummary(claims.Items[i])))
	}
	for i := range hpas.Items {
		addSnapshot(hpaSnapshot(hpas.Items[i]))
	}

	replicaSetOwners := deploymentOwners(replicaSets.Items)
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Labels[HookRunIDLabel] != "" || ownerKind(pod.OwnerReferences) == "Job" {
			continue
		}
		podID := addSnapshot(podSnapshot(*pod))
		if workloadID := podWorkloadNodeID(*pod, replicaSetOwners, options); workloadID != "" {
			addTopologyEdge(edges, workloadID, podID, "owns")
		}
	}

	for i := range services.Items {
		service := &services.Items[i]
		if len(service.Spec.Selector) == 0 {
			continue
		}
		serviceID := topologyResourceID(options.ClusterID, "Service", service.Namespace, service.Name)
		selector := klabels.SelectorFromSet(service.Spec.Selector)
		matchedWorkload := false
		for _, workload := range workloads {
			if selector.Matches(klabels.Set(workload.template.Labels)) {
				addTopologyEdge(edges, serviceID, workload.nodeID, "selects")
				matchedWorkload = true
			}
		}
		if matchedWorkload {
			continue
		}
		for i := range pods.Items {
			if selector.Matches(klabels.Set(pods.Items[i].Labels)) {
				podID := topologyResourceID(options.ClusterID, "Pod", pods.Items[i].Namespace, pods.Items[i].Name)
				if _, exists := nodes[podID]; exists {
					addTopologyEdge(edges, serviceID, podID, "selects")
				}
			}
		}
	}

	for i := range hpas.Items {
		hpa := &hpas.Items[i]
		hpaID := topologyResourceID(options.ClusterID, "HorizontalPodAutoscaler", hpa.Namespace, hpa.Name)
		targetID := topologyResourceID(options.ClusterID, hpa.Spec.ScaleTargetRef.Kind, hpa.Namespace, hpa.Spec.ScaleTargetRef.Name)
		if _, exists := nodes[targetID]; exists {
			addTopologyEdge(edges, hpaID, targetID, "scales")
		}
	}
	for _, workload := range workloads {
		addPodTemplateDependencies(nodes, edges, workload, options)
	}

	if err := c.addGatewayTopology(ctx, options, managedSelector, nodes, edges); err != nil {
		return ApplicationTopologySnapshot{}, err
	}

	return sortedApplicationTopology(nodes, edges), nil
}

func topologyNode(snapshot ResourceSnapshot, options ApplicationTopologyOptions) ApplicationTopologyNode {
	return ApplicationTopologyNode{
		ID:                 topologyResourceID(options.ClusterID, snapshot.Kind, snapshot.Namespace, snapshot.Name),
		Kind:               snapshot.Kind,
		Name:               snapshot.Name,
		Namespace:          snapshot.Namespace,
		Status:             snapshot.Status,
		Summary:            snapshot.Summary,
		ClusterID:          options.ClusterID,
		ClusterName:        options.ClusterName,
		DeploymentTargetID: snapshot.DeploymentTargetID,
	}
}

func topologyResourceID(clusterID, kind, namespace, name string) string {
	return strings.Join([]string{clusterID, strings.ToLower(kind), namespace, name}, "/")
}

func addTopologyEdge(edges map[string]ApplicationTopologyEdge, source, target, edgeType string) {
	if source == "" || target == "" || source == target {
		return
	}
	id := source + "->" + target + ":" + edgeType
	edges[id] = ApplicationTopologyEdge{ID: id, Source: source, Target: target, Type: edgeType}
}

func deploymentOwners(replicaSets []appsv1.ReplicaSet) map[string]string {
	owners := make(map[string]string, len(replicaSets))
	for i := range replicaSets {
		for _, owner := range replicaSets[i].OwnerReferences {
			if owner.Kind == "Deployment" {
				owners[replicaSets[i].Name] = owner.Name
				break
			}
		}
	}
	return owners
}

func ownerKind(references []metav1.OwnerReference) string {
	if len(references) == 0 {
		return ""
	}
	return references[0].Kind
}

func podWorkloadNodeID(pod corev1.Pod, replicaSetOwners map[string]string, options ApplicationTopologyOptions) string {
	for _, owner := range pod.OwnerReferences {
		switch owner.Kind {
		case "StatefulSet":
			return topologyResourceID(options.ClusterID, owner.Kind, pod.Namespace, owner.Name)
		case "ReplicaSet":
			if deploymentName := replicaSetOwners[owner.Name]; deploymentName != "" {
				return topologyResourceID(options.ClusterID, "Deployment", pod.Namespace, deploymentName)
			}
		}
	}
	return ""
}

func addPodTemplateDependencies(nodes map[string]ApplicationTopologyNode, edges map[string]ApplicationTopologyEdge, workload applicationWorkload, options ApplicationTopologyOptions) {
	configMaps := map[string]struct{}{}
	secrets := map[string]struct{}{}
	claims := map[string]struct{}{}
	containers := append([]corev1.Container{}, workload.template.Spec.InitContainers...)
	containers = append(containers, workload.template.Spec.Containers...)
	for _, container := range containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				configMaps[envFrom.ConfigMapRef.Name] = struct{}{}
			}
			if envFrom.SecretRef != nil {
				secrets[envFrom.SecretRef.Name] = struct{}{}
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom == nil {
				continue
			}
			if env.ValueFrom.ConfigMapKeyRef != nil {
				configMaps[env.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
			}
			if env.ValueFrom.SecretKeyRef != nil {
				secrets[env.ValueFrom.SecretKeyRef.Name] = struct{}{}
			}
		}
	}
	for _, volume := range workload.template.Spec.Volumes {
		if volume.ConfigMap != nil {
			configMaps[volume.ConfigMap.Name] = struct{}{}
		}
		if volume.Secret != nil {
			secrets[volume.Secret.SecretName] = struct{}{}
		}
		if volume.PersistentVolumeClaim != nil {
			claims[volume.PersistentVolumeClaim.ClaimName] = struct{}{}
		}
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.ConfigMap != nil {
					configMaps[source.ConfigMap.Name] = struct{}{}
				}
				if source.Secret != nil {
					secrets[source.Secret.Name] = struct{}{}
				}
			}
		}
	}
	for _, pullSecret := range workload.template.Spec.ImagePullSecrets {
		secrets[pullSecret.Name] = struct{}{}
	}
	addDependencies := func(kind string, names map[string]struct{}, edgeType string) {
		for name := range names {
			dependencyID := topologyResourceID(options.ClusterID, kind, options.Namespace, name)
			if _, exists := nodes[dependencyID]; exists {
				addTopologyEdge(edges, dependencyID, workload.nodeID, edgeType)
			}
		}
	}
	addDependencies("ConfigMap", configMaps, "configures")
	addDependencies("Secret", secrets, "configures")
	addDependencies("PersistentVolumeClaim", claims, "mounts")
}

func (c *Client) addGatewayTopology(ctx context.Context, options ApplicationTopologyOptions, selector string, nodes map[string]ApplicationTopologyNode, edges map[string]ApplicationTopologyEdge) error {
	if c.dynamic == nil {
		return nil
	}
	routes, err := c.dynamic.Resource(httpRouteGVR).Namespace(options.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for i := range routes.Items {
		route := &routes.Items[i]
		routeNode := topologyNode(httpRouteSnapshot(route), options)
		nodes[routeNode.ID] = routeNode
		parentRefs, _, _ := unstructured.NestedSlice(route.Object, "spec", "parentRefs")
		for _, rawParent := range parentRefs {
			parent, ok := rawParent.(map[string]any)
			if !ok || stringField(parent, "kind", "Gateway") != "Gateway" {
				continue
			}
			name := stringField(parent, "name", "")
			namespace := stringField(parent, "namespace", route.GetNamespace())
			if name == "" {
				continue
			}
			gateway, getErr := c.dynamic.Resource(gatewayGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(getErr) {
				continue
			}
			if getErr != nil {
				return getErr
			}
			gatewayNode := topologyNode(gatewaySnapshot(gateway), options)
			nodes[gatewayNode.ID] = gatewayNode
			addTopologyEdge(edges, gatewayNode.ID, routeNode.ID, "accepts")
		}
		rules, _, _ := unstructured.NestedSlice(route.Object, "spec", "rules")
		for _, rawRule := range rules {
			rule, ok := rawRule.(map[string]any)
			if !ok {
				continue
			}
			backendRefs, _ := rule["backendRefs"].([]any)
			for _, rawBackend := range backendRefs {
				backend, ok := rawBackend.(map[string]any)
				if !ok || stringField(backend, "kind", "Service") != "Service" {
					continue
				}
				name := stringField(backend, "name", "")
				namespace := stringField(backend, "namespace", route.GetNamespace())
				serviceID := topologyResourceID(options.ClusterID, "Service", namespace, name)
				if _, exists := nodes[serviceID]; exists {
					addTopologyEdge(edges, routeNode.ID, serviceID, "routes")
				}
			}
		}
	}
	return nil
}

func stringField(values map[string]any, key, fallback string) string {
	value, ok := values[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func sortedApplicationTopology(nodes map[string]ApplicationTopologyNode, edges map[string]ApplicationTopologyEdge) ApplicationTopologySnapshot {
	snapshot := ApplicationTopologySnapshot{
		Nodes: make([]ApplicationTopologyNode, 0, len(nodes)),
		Edges: make([]ApplicationTopologyEdge, 0, len(edges)),
	}
	for _, node := range nodes {
		snapshot.Nodes = append(snapshot.Nodes, node)
	}
	for _, edge := range edges {
		snapshot.Edges = append(snapshot.Edges, edge)
	}
	sort.Slice(snapshot.Nodes, func(i, j int) bool { return snapshot.Nodes[i].ID < snapshot.Nodes[j].ID })
	sort.Slice(snapshot.Edges, func(i, j int) bool { return snapshot.Edges[i].ID < snapshot.Edges[j].ID })
	return snapshot
}
