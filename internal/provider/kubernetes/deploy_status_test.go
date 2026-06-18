package kubernetes

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetDeploymentSnapshotReadsSucceededDeployment(t *testing.T) {
	replicas := int32(2)
	client := NewClientForInterface(fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api-dev", Namespace: "project-demo", Generation: 3},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 3,
			UpdatedReplicas:    2,
			ReadyReplicas:      2,
			AvailableReplicas:  2,
			Conditions: []appsv1.DeploymentCondition{{
				Type:    appsv1.DeploymentAvailable,
				Status:  corev1.ConditionTrue,
				Message: "available",
			}},
		},
	}))

	snapshot, err := client.GetDeploymentSnapshot(context.Background(), "project-demo", "api-dev")
	if err != nil {
		t.Fatalf("GetDeploymentSnapshot returned error: %v", err)
	}
	if snapshot.Phase != DeploymentSucceeded || snapshot.Message != "available" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestGetDeploymentSnapshotReadsProgressDeadlineFailure(t *testing.T) {
	replicas := int32(2)
	client := NewClientForInterface(fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api-dev", Namespace: "project-demo", Generation: 3},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 3,
			UpdatedReplicas:    1,
			Conditions: []appsv1.DeploymentCondition{{
				Type:    appsv1.DeploymentProgressing,
				Status:  corev1.ConditionFalse,
				Reason:  "ProgressDeadlineExceeded",
				Message: "timed out",
			}},
		},
	}))

	snapshot, err := client.GetDeploymentSnapshot(context.Background(), "project-demo", "api-dev")
	if err != nil {
		t.Fatalf("GetDeploymentSnapshot returned error: %v", err)
	}
	if snapshot.Phase != DeploymentFailed || snapshot.Message != "timed out" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestRestartDeploymentUpdatesPodTemplateAnnotation(t *testing.T) {
	client := NewClientForInterface(fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api-dev", Namespace: "project-demo"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"existing": "keep"}},
			},
		},
	}))

	if err := client.RestartDeployment(context.Background(), "project-demo", "api-dev"); err != nil {
		t.Fatalf("RestartDeployment returned error: %v", err)
	}
	deployment, err := client.client.AppsV1().Deployments("project-demo").Get(context.Background(), "api-dev", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get deployment returned error: %v", err)
	}
	if deployment.Spec.Template.Annotations["existing"] != "keep" {
		t.Fatalf("existing annotation was not preserved: %#v", deployment.Spec.Template.Annotations)
	}
	if deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] == "" {
		t.Fatalf("restart annotation was not set: %#v", deployment.Spec.Template.Annotations)
	}
}
