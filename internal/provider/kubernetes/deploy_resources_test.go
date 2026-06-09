package kubernetes

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestApplyApplicationResourcesCreatesWorkloadResources(t *testing.T) {
	client := NewClientForInterface(fake.NewSimpleClientset())
	spec := ApplicationResourcesSpec{
		Name:                  "api-dev",
		Namespace:             "project-demo",
		ProjectID:             "prj_demo",
		ApplicationID:         "app_api",
		EnvironmentID:         "env_dev",
		DeploymentTargetID:    "dplt_backend",
		ReleaseID:             "rel_1",
		Image:                 "registry.example.com/acme/api:v1",
		Replicas:              2,
		ServicePort:           8080,
		CPURequest:            "100m",
		MemoryRequest:         "128Mi",
		RolloutTimeoutSeconds: 120,
		ConfigData:            map[string]string{"APP_ENV": "dev"},
		SecretData:            map[string]string{"TOKEN": "secret"},
	}

	if err := client.ApplyApplicationResources(context.Background(), spec); err != nil {
		t.Fatalf("ApplyApplicationResources returned error: %v", err)
	}

	deployment, err := client.client.AppsV1().Deployments(spec.Namespace).Get(context.Background(), spec.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if *deployment.Spec.Replicas != 2 {
		t.Fatalf("replicas = %d", *deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != spec.Image {
		t.Fatalf("image = %q", deployment.Spec.Template.Spec.Containers[0].Image)
	}
	if deployment.Spec.ProgressDeadlineSeconds == nil || *deployment.Spec.ProgressDeadlineSeconds != 120 {
		t.Fatalf("progress deadline = %#v", deployment.Spec.ProgressDeadlineSeconds)
	}
	assertManagedLabels(t, deployment.Labels, spec.Name, spec.ProjectID, spec.ApplicationID, spec.EnvironmentID, spec.DeploymentTargetID, spec.ReleaseID)
	assertSelectorLabels(t, deployment.Spec.Selector.MatchLabels, spec.Name, spec.ProjectID, spec.ApplicationID, spec.EnvironmentID, spec.DeploymentTargetID)
	assertSelectorLabels(t, deployment.Spec.Template.Labels, spec.Name, spec.ProjectID, spec.ApplicationID, spec.EnvironmentID, spec.DeploymentTargetID)

	service, err := client.client.CoreV1().Services(spec.Namespace).Get(context.Background(), spec.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get service: %v", err)
	}
	if service.Spec.Ports[0].Port != 8080 {
		t.Fatalf("service port = %d", service.Spec.Ports[0].Port)
	}
	assertManagedLabels(t, service.Labels, spec.Name, spec.ProjectID, spec.ApplicationID, spec.EnvironmentID, spec.DeploymentTargetID, spec.ReleaseID)

	configMap, err := client.client.CoreV1().ConfigMaps(spec.Namespace).Get(context.Background(), spec.Name+"-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get configmap: %v", err)
	}
	if configMap.Data["APP_ENV"] != "dev" {
		t.Fatalf("config data = %#v", configMap.Data)
	}
	assertManagedLabels(t, configMap.Labels, spec.Name, spec.ProjectID, spec.ApplicationID, spec.EnvironmentID, spec.DeploymentTargetID, spec.ReleaseID)

	secret, err := client.client.CoreV1().Secrets(spec.Namespace).Get(context.Background(), spec.Name+"-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if string(secret.Data["TOKEN"]) != "secret" {
		t.Fatalf("secret data = %#v", secret.Data)
	}
	assertManagedLabels(t, secret.Labels, spec.Name, spec.ProjectID, spec.ApplicationID, spec.EnvironmentID, spec.DeploymentTargetID, spec.ReleaseID)
}

func TestApplyApplicationResourcesKeepsDeploymentSelectorStableAcrossReleases(t *testing.T) {
	client := NewClientForInterface(fake.NewSimpleClientset())
	spec := ApplicationResourcesSpec{
		Name:               "api-backend-dev",
		Namespace:          "project-demo",
		ProjectID:          "prj_demo",
		ApplicationID:      "app_api",
		EnvironmentID:      "env_dev",
		DeploymentTargetID: "dplt_backend",
		ReleaseID:          "rel_1",
		Image:              "registry.example.com/acme/api:v1",
		ServicePort:        8080,
	}
	if err := client.ApplyApplicationResources(context.Background(), spec); err != nil {
		t.Fatalf("first apply returned error: %v", err)
	}

	spec.ReleaseID = "rel_2"
	spec.Image = "registry.example.com/acme/api:v2"
	if err := client.ApplyApplicationResources(context.Background(), spec); err != nil {
		t.Fatalf("second apply returned error: %v", err)
	}

	deployment, err := client.client.AppsV1().Deployments(spec.Namespace).Get(context.Background(), spec.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if deployment.Spec.Selector.MatchLabels[ReleaseIDLabel] != "" {
		t.Fatalf("selector should not include release label: %#v", deployment.Spec.Selector.MatchLabels)
	}
	if deployment.Labels[ReleaseIDLabel] != "rel_2" {
		t.Fatalf("deployment release label = %q", deployment.Labels[ReleaseIDLabel])
	}
}

func assertManagedLabels(t *testing.T, labels map[string]string, name string, projectID string, applicationID string, environmentID string, deploymentTargetID string, releaseID string) {
	t.Helper()
	expected := map[string]string{
		ManagedByLabel:           ManagedByValue,
		ApplicationNameKey:       name,
		ProjectIDLabel:           projectID,
		ApplicationIDLabel:       applicationID,
		EnvironmentIDLabel:       environmentID,
		DeploymentTargetIDLabel:  deploymentTargetID,
		ReleaseIDLabel:           releaseID,
		legacyApplicationIDLabel: applicationID,
		legacyEnvironmentIDLabel: environmentID,
	}
	for key, value := range expected {
		if labels[key] != value {
			t.Fatalf("label %s = %q, want %q in %#v", key, labels[key], value, labels)
		}
	}
}

func assertSelectorLabels(t *testing.T, labels map[string]string, name string, projectID string, applicationID string, environmentID string, deploymentTargetID string) {
	t.Helper()
	expected := map[string]string{
		ManagedByLabel:           ManagedByValue,
		ApplicationNameKey:       name,
		ProjectIDLabel:           projectID,
		ApplicationIDLabel:       applicationID,
		EnvironmentIDLabel:       environmentID,
		DeploymentTargetIDLabel:  deploymentTargetID,
		legacyApplicationIDLabel: applicationID,
		legacyEnvironmentIDLabel: environmentID,
	}
	for key, value := range expected {
		if labels[key] != value {
			t.Fatalf("selector label %s = %q, want %q in %#v", key, labels[key], value, labels)
		}
	}
	if labels[ReleaseIDLabel] != "" {
		t.Fatalf("selector labels must not include release id: %#v", labels)
	}
}
