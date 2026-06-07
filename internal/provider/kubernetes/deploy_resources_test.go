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

	service, err := client.client.CoreV1().Services(spec.Namespace).Get(context.Background(), spec.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get service: %v", err)
	}
	if service.Spec.Ports[0].Port != 8080 {
		t.Fatalf("service port = %d", service.Spec.Ports[0].Port)
	}

	configMap, err := client.client.CoreV1().ConfigMaps(spec.Namespace).Get(context.Background(), spec.Name+"-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get configmap: %v", err)
	}
	if configMap.Data["APP_ENV"] != "dev" {
		t.Fatalf("config data = %#v", configMap.Data)
	}

	secret, err := client.client.CoreV1().Secrets(spec.Namespace).Get(context.Background(), spec.Name+"-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if string(secret.Data["TOKEN"]) != "secret" {
		t.Fatalf("secret data = %#v", secret.Data)
	}
}
