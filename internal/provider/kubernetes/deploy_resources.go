package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	DeploymentTargetID    string
	ReleaseID             string
	Image                 string
	Replicas              int32
	ServicePort           int32
	CPURequest            string
	MemoryRequest         string
	RolloutTimeoutSeconds int32
	ConfigData            map[string]string
	SecretData            map[string]string
}

type HookJobSpec struct {
	Name               string
	Namespace          string
	ProjectID          string
	ApplicationID      string
	ModuleID           string
	BuildRunID         string
	EnvironmentID      string
	DeploymentTargetID string
	ReleaseID          string
	HookRunID          string
	Phase              string
	Image              string
	GitBranch          string
	GitTag             string
	GitRefName         string
	GitRefType         string
	GitRef             string
	GitSHA             string
	GitShortSHA        string
	Shell              string
	Script             string
	TimeoutSeconds     int32
	ConfigMapName      string
	SecretName         string
}

type HookJobResult struct {
	Succeeded bool
	ExitCode  int32
	Message   string
	Logs      string
}

func (c *Client) ApplyApplicationResources(ctx context.Context, spec ApplicationResourcesSpec) error {
	if err := validateApplicationResourcesSpec(spec); err != nil {
		return err
	}
	objectLabels := appObjectLabels(spec)
	selectorLabels := appSelectorLabels(spec)
	if err := c.applyApplicationRuntimeConfig(ctx, spec, objectLabels); err != nil {
		return err
	}
	if err := c.applyDeployment(ctx, spec, objectLabels, selectorLabels); err != nil {
		return err
	}
	return c.applyService(ctx, spec, objectLabels, selectorLabels)
}

func (c *Client) ApplyApplicationRuntimeConfig(ctx context.Context, spec ApplicationResourcesSpec) error {
	if err := validateApplicationResourcesSpec(spec); err != nil {
		return err
	}
	return c.applyApplicationRuntimeConfig(ctx, spec, appObjectLabels(spec))
}

func (c *Client) applyApplicationRuntimeConfig(ctx context.Context, spec ApplicationResourcesSpec, objectLabels map[string]string) error {
	if err := c.applyConfigMap(ctx, spec, objectLabels); err != nil {
		return err
	}
	if err := c.applySecret(ctx, spec, objectLabels); err != nil {
		return err
	}
	return nil
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

func (c *Client) applyDeployment(ctx context.Context, spec ApplicationResourcesSpec, objectLabels map[string]string, selectorLabels map[string]string) error {
	replicas := spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	progressDeadlineSeconds := spec.RolloutTimeoutSeconds
	if progressDeadlineSeconds <= 0 {
		progressDeadlineSeconds = 600
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: objectLabels},
		Spec: appsv1.DeploymentSpec{
			Replicas:                &replicas,
			Selector:                &metav1.LabelSelector{MatchLabels: selectorLabels},
			ProgressDeadlineSeconds: &progressDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: selectorLabels},
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
	existing.Labels = objectLabels
	existing.Spec = deployment.Spec
	_, err = c.client.AppsV1().Deployments(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) applyService(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string, selectorLabels map[string]string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
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
	existing.Spec.Selector = selectorLabels
	existing.Spec.Ports = service.Spec.Ports
	_, err = c.client.CoreV1().Services(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) RunHookJob(ctx context.Context, spec HookJobSpec) (HookJobResult, error) {
	if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.Namespace) == "" || strings.TrimSpace(spec.Image) == "" {
		return HookJobResult{}, fmt.Errorf("hook job name, namespace and image are required")
	}
	labels := baseManagedLabels(spec.Name)
	setLabel(labels, ProjectIDLabel, spec.ProjectID)
	setLabel(labels, ApplicationIDLabel, spec.ApplicationID)
	setLabel(labels, EnvironmentIDLabel, spec.EnvironmentID)
	setLabel(labels, DeploymentTargetIDLabel, spec.DeploymentTargetID)
	setLabel(labels, ReleaseIDLabel, spec.ReleaseID)
	setLabel(labels, HookRunIDLabel, spec.HookRunID)
	setLabel(labels, HookPhaseLabel, spec.Phase)
	scriptMapName := spec.Name + "-script"
	scriptMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: scriptMapName, Namespace: spec.Namespace, Labels: labels},
		Data:       map[string]string{"run.sh": spec.Script},
	}
	if err := c.applyHookScriptConfigMap(ctx, scriptMap); err != nil {
		return HookJobResult{}, err
	}
	shell := strings.TrimSpace(spec.Shell)
	if shell != "bash" {
		shell = "sh"
	}
	timeout := spec.TimeoutSeconds
	if timeout <= 0 {
		timeout = 300
	}
	backoffLimit := int32(0)
	mode := int32(0o755)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoffLimit,
			ActiveDeadlineSeconds: int64Ptr(int64(timeout)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: boolPtr(false),
					RestartPolicy:                corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "hook",
						Image:   spec.Image,
						Command: []string{shell, "/liteyuki-hooks/run.sh"},
						EnvFrom: []corev1.EnvFromSource{
							{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: spec.ConfigMapName}}},
							{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: spec.SecretName}}},
						},
						Env: []corev1.EnvVar{
							{Name: "LITEYUKI_PROJECT_ID", Value: spec.ProjectID},
							{Name: "LITEYUKI_APPLICATION_ID", Value: spec.ApplicationID},
							{Name: "LITEYUKI_MODULE_ID", Value: spec.ModuleID},
							{Name: "LITEYUKI_BUILD_RUN_ID", Value: spec.BuildRunID},
							{Name: "LITEYUKI_ENVIRONMENT_ID", Value: spec.EnvironmentID},
							{Name: "LITEYUKI_DEPLOYMENT_TARGET_ID", Value: spec.DeploymentTargetID},
							{Name: "LITEYUKI_RELEASE_ID", Value: spec.ReleaseID},
							{Name: "LITEYUKI_HOOK_RUN_ID", Value: spec.HookRunID},
							{Name: "LITEYUKI_HOOK_PHASE", Value: spec.Phase},
							{Name: "LITEYUKI_IMAGE_REF", Value: spec.Image},
							{Name: "LITEYUKI_GIT_BRANCH", Value: spec.GitBranch},
							{Name: "LITEYUKI_GIT_TAG", Value: spec.GitTag},
							{Name: "LITEYUKI_GIT_REF_NAME", Value: spec.GitRefName},
							{Name: "LITEYUKI_GIT_REF_TYPE", Value: spec.GitRefType},
							{Name: "LITEYUKI_GIT_REF", Value: spec.GitRef},
							{Name: "LITEYUKI_GIT_SHA", Value: spec.GitSHA},
							{Name: "LITEYUKI_GIT_SHORT_SHA", Value: spec.GitShortSHA},
						},
						VolumeMounts: []corev1.VolumeMount{{Name: "hook-script", MountPath: "/liteyuki-hooks", ReadOnly: true}},
					}},
					Volumes: []corev1.Volume{{Name: "hook-script", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: scriptMapName},
						DefaultMode:          &mode,
					}}}},
				},
			},
		},
	}
	_ = c.client.BatchV1().Jobs(spec.Namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if _, err := c.client.BatchV1().Jobs(spec.Namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
		return HookJobResult{}, err
	}
	return c.waitForHookJob(ctx, spec.Namespace, spec.Name, time.Duration(timeout)*time.Second)
}

func (c *Client) applyHookScriptConfigMap(ctx context.Context, item *corev1.ConfigMap) error {
	existing, err := c.client.CoreV1().ConfigMaps(item.Namespace).Get(ctx, item.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.CoreV1().ConfigMaps(item.Namespace).Create(ctx, item, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = item.Labels
	existing.Data = item.Data
	_, err = c.client.CoreV1().ConfigMaps(item.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) waitForHookJob(ctx context.Context, namespace string, name string, timeout time.Duration) (HookJobResult, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		job, err := c.client.BatchV1().Jobs(namespace).Get(waitCtx, name, metav1.GetOptions{})
		if err != nil {
			return HookJobResult{}, err
		}
		if job.Status.Succeeded > 0 {
			logs := c.hookJobLogs(waitCtx, namespace, name)
			return HookJobResult{Succeeded: true, Message: "hook job succeeded", Logs: logs}, nil
		}
		if job.Status.Failed > 0 {
			logs := c.hookJobLogs(waitCtx, namespace, name)
			return HookJobResult{Succeeded: false, ExitCode: 1, Message: "hook job failed", Logs: logs}, nil
		}
		select {
		case <-waitCtx.Done():
			logs := c.hookJobLogs(ctx, namespace, name)
			return HookJobResult{Succeeded: false, ExitCode: 124, Message: fmt.Sprintf("hook job timed out after %s", timeout), Logs: logs}, nil
		case <-ticker.C:
		}
	}
}

func (c *Client) hookJobLogs(ctx context.Context, namespace string, jobName string) string {
	pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}
	req := c.client.CoreV1().Pods(namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{})
	data, err := req.Do(ctx).Raw()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func appSelectorLabels(spec ApplicationResourcesSpec) map[string]string {
	labels := baseManagedLabels(spec.Name)
	setLabel(labels, ProjectIDLabel, spec.ProjectID)
	setLabel(labels, ApplicationIDLabel, spec.ApplicationID)
	setLabel(labels, EnvironmentIDLabel, spec.EnvironmentID)
	setLabel(labels, DeploymentTargetIDLabel, spec.DeploymentTargetID)
	setLabel(labels, legacyApplicationIDLabel, spec.ApplicationID)
	setLabel(labels, legacyEnvironmentIDLabel, spec.EnvironmentID)
	return labels
}

func appObjectLabels(spec ApplicationResourcesSpec) map[string]string {
	labels := appSelectorLabels(spec)
	setLabel(labels, ReleaseIDLabel, spec.ReleaseID)
	return labels
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

func int64Ptr(value int64) *int64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
