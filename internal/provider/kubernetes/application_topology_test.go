package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildApplicationTopologyResolvesRuntimeRelations(t *testing.T) {
	labels := map[string]string{
		ManagedByLabel:          ManagedByValue,
		ProjectIDLabel:          "project-1",
		ApplicationIDLabel:      "app-1",
		DeploymentTargetIDLabel: "target-1",
	}
	podLabels := cloneStringMap(labels)
	podLabels["app"] = "demo"
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "project-ns", Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "demo"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "demo",
						EnvFrom: []corev1.EnvFromSource{
							{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "demo-config"}}},
							{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "demo-secret"}}},
						},
					}},
					Volumes: []corev1.Volume{{
						Name:         "data",
						VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "demo-data"}},
					}},
				},
			},
		},
	}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "demo-rs",
			Namespace:       "project-ns",
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "demo"}},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "demo-pod",
			Namespace:       "project-ns",
			Labels:          podLabels,
			OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "demo-rs"}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "project-ns", Labels: labels},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "demo"}},
	}
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "project-ns", Labels: labels},
		Spec:       autoscalingv2.HorizontalPodAutoscalerSpec{ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{Kind: "Deployment", Name: "demo"}, MaxReplicas: 3},
	}
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "demo-config", Namespace: "project-ns", Labels: labels}, Data: map[string]string{"PUBLIC_SETTING": "value"}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "demo-secret", Namespace: "project-ns", Labels: labels}, Data: map[string][]byte{"TOKEN": []byte("must-not-leak")}}
	claim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "demo-data", Namespace: "project-ns", Labels: labels}}

	route := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "HTTPRoute",
		"metadata":   map[string]any{"name": "demo-route", "namespace": "project-ns", "labels": stringMapAny(labels)},
		"spec": map[string]any{
			"parentRefs": []any{map[string]any{"name": "luna-gateway", "namespace": "gateway-system"}},
			"rules":      []any{map[string]any{"backendRefs": []any{map[string]any{"name": "demo"}}}},
		},
	}}
	gateway := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata":   map[string]any{"name": "luna-gateway", "namespace": "gateway-system"},
	}}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{httpRouteGVR: "HTTPRouteList", gatewayGVR: "GatewayList"},
	)
	if _, err := dynamicClient.Resource(httpRouteGVR).Namespace("project-ns").Create(context.Background(), route, metav1.CreateOptions{}); err != nil {
		t.Fatalf("fake HTTPRoute create failed: %v", err)
	}
	if _, err := dynamicClient.Resource(gatewayGVR).Namespace("gateway-system").Create(context.Background(), gateway, metav1.CreateOptions{}); err != nil {
		t.Fatalf("fake Gateway create failed: %v", err)
	}
	client := NewClientForInterfaces(fake.NewSimpleClientset(deployment, replicaSet, pod, service, hpa, configMap, secret, claim), dynamicClient)

	snapshot, err := client.BuildApplicationTopology(context.Background(), ApplicationTopologyOptions{
		ClusterID: "cluster-1", Namespace: "project-ns", ProjectID: "project-1", ApplicationID: "app-1", DeploymentTargetID: "target-1",
	})
	if err != nil {
		t.Fatalf("BuildApplicationTopology returned error: %v", err)
	}

	assertTopologyEdge(t, snapshot, "Deployment", "Pod", "owns")
	assertTopologyEdge(t, snapshot, "Service", "Deployment", "selects")
	assertTopologyEdge(t, snapshot, "HorizontalPodAutoscaler", "Deployment", "scales")
	assertTopologyEdge(t, snapshot, "ConfigMap", "Deployment", "configures")
	assertTopologyEdge(t, snapshot, "Secret", "Deployment", "configures")
	assertTopologyEdge(t, snapshot, "PersistentVolumeClaim", "Deployment", "mounts")
	assertTopologyEdge(t, snapshot, "Gateway", "HTTPRoute", "accepts")
	assertTopologyEdge(t, snapshot, "HTTPRoute", "Service", "routes")
	for _, node := range snapshot.Nodes {
		if node.Kind == "Secret" && node.Summary != "" {
			t.Fatalf("secret topology node exposed a summary: %q", node.Summary)
		}
	}
}

func assertTopologyEdge(t *testing.T, snapshot ApplicationTopologySnapshot, sourceKind, targetKind, edgeType string) {
	t.Helper()
	kinds := make(map[string]string, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		kinds[node.ID] = node.Kind
	}
	for _, edge := range snapshot.Edges {
		if edge.Type == edgeType && kinds[edge.Source] == sourceKind && kinds[edge.Target] == targetKind {
			return
		}
	}
	t.Fatalf("missing topology edge %s -> %s (%s)", sourceKind, targetKind, edgeType)
}

func stringMapAny(source map[string]string) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
