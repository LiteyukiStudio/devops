package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type ApplicationResourcesSpec struct {
	Name                         string
	Namespace                    string
	WorkloadType                 string
	ProjectID                    string
	ApplicationID                string
	EnvironmentID                string
	DeploymentTargetID           string
	ReleaseID                    string
	BuildRunID                   string
	ImageDigest                  string
	Image                        string
	Replicas                     int32
	ServicePort                  int32
	ServicePorts                 []ApplicationServicePort
	CPURequest                   string
	MemoryRequest                string
	CPULimit                     string
	MemoryLimit                  string
	ImagePullPolicy              string
	ContainerCommand             string
	ContainerArgs                string
	Lifecycle                    string
	InitContainers               string
	SidecarContainers            string
	ReadinessProbe               string
	LivenessProbe                string
	StartupProbe                 string
	RunAsUser                    string
	RunAsGroup                   string
	FSGroup                      string
	FSGroupChangePolicy          string
	ReadOnlyRootFilesystem       bool
	AllowPrivilegeEscalation     string
	CapabilityAdd                string
	CapabilityDrop               string
	NodeSelector                 string
	Tolerations                  string
	Affinity                     string
	TopologySpreadConstraints    string
	PriorityClassName            string
	ServiceType                  string
	ServiceAnnotations           string
	ServiceExternalTrafficPolicy string
	ServiceSessionAffinity       string
	AutoScalingEnabled           bool
	AutoScalingMinReplicas       int32
	AutoScalingMaxReplicas       int32
	AutoScalingCPUPercent        int32
	AutoScalingMemoryPercent     int32
	AutoScalingBehavior          string
	RolloutTimeoutSeconds        int32
	ConfigData                   map[string]string
	SecretData                   map[string]string
	ConfigFiles                  []ApplicationConfigFile
	SecretFiles                  []ApplicationConfigFile
	DataRetentionEnabled         bool
	DataCapacity                 string
	DataMountPath                string
	DataVolumes                  []ApplicationDataVolume
	DataStorageClassName         string
	DataAccessMode               string
	DataVolumeMode               string
	ForceImagePull               bool
}

type ApplicationServicePort struct {
	Name        string
	Port        int32
	AppProtocol string
}

type ApplicationConfigFile struct {
	Path    string
	Key     string
	Content string
}

type ApplicationDataVolume struct {
	Name              string
	MountPath         string
	Capacity          string
	SourceType        string
	ExistingClaimName string
	EmptyDirMedium    string
	EmptyDirSizeLimit string
}

type HookJobSpec struct {
	Name               string
	Namespace          string
	ProjectID          string
	ApplicationID      string
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

const (
	hookJobSuccessTTLSeconds int32 = 300
	hookJobFailureTTLSeconds int32 = 86400
)

type DataExportSpec struct {
	Name      string
	Namespace string
	PVCName   string
	MountPath string
	Volumes   []DataExportVolume
}

type DataExportVolume struct {
	Name    string
	PVCName string
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
	if spec.DataRetentionEnabled {
		for _, volume := range persistentDataVolumes(spec) {
			if err := validateApplicationDataVolume(volume); err != nil {
				return err
			}
			if !dataVolumeNeedsPVC(volume) {
				continue
			}
			if err := c.applyPersistentDataVolume(ctx, spec, volume, objectLabels); err != nil {
				return err
			}
		}
	}
	effectiveSelectorLabels, err := c.applyApplicationWorkload(ctx, spec, objectLabels, selectorLabels)
	if err != nil {
		return err
	}
	if err := c.applyApplicationAutoScaling(ctx, spec, objectLabels); err != nil {
		return err
	}
	return c.applyService(ctx, spec, objectLabels, effectiveSelectorLabels)
}

func validateApplicationDataVolume(volume ApplicationDataVolume) error {
	switch dataVolumeSourceType(volume) {
	case "existingClaim":
		if strings.TrimSpace(volume.ExistingClaimName) == "" {
			return fmt.Errorf("existing claim data volume %s requires claim name", persistentDataVolumeName(volume))
		}
	case "emptyDir":
		if sizeLimit := strings.TrimSpace(volume.EmptyDirSizeLimit); sizeLimit != "" {
			quantity, err := resource.ParseQuantity(sizeLimit)
			if err != nil || quantity.Sign() <= 0 {
				return fmt.Errorf("emptyDir size limit must be a positive resource quantity")
			}
		}
	}
	return nil
}

func (c *Client) ApplyApplicationRuntimeConfig(ctx context.Context, spec ApplicationResourcesSpec) error {
	if err := validateApplicationResourcesSpec(spec); err != nil {
		return err
	}
	objectLabels := appObjectLabels(spec)
	if err := c.applyApplicationRuntimeConfig(ctx, spec, objectLabels); err != nil {
		return err
	}
	if spec.DataRetentionEnabled {
		for _, volume := range persistentDataVolumes(spec) {
			if !dataVolumeNeedsPVC(volume) {
				continue
			}
			if err := c.applyPersistentDataVolume(ctx, spec, volume, objectLabels); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) ApplyPersistentDataVolume(ctx context.Context, spec ApplicationResourcesSpec) error {
	if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.Namespace) == "" {
		return fmt.Errorf("application resource name and namespace are required")
	}
	for _, volume := range persistentDataVolumes(spec) {
		if !dataVolumeNeedsPVC(volume) {
			continue
		}
		if _, err := persistentDataCapacity(volume); err != nil {
			return err
		}
		if err := c.applyPersistentDataVolume(ctx, spec, volume, appObjectLabels(spec)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) applyApplicationRuntimeConfig(ctx context.Context, spec ApplicationResourcesSpec, objectLabels map[string]string) error {
	if err := c.applyConfigMap(ctx, spec, objectLabels); err != nil {
		return err
	}
	if err := c.applySecret(ctx, spec, objectLabels); err != nil {
		return err
	}
	if err := c.applyConfigFilesConfigMap(ctx, spec, objectLabels); err != nil {
		return err
	}
	if err := c.applySecretFilesSecret(ctx, spec, objectLabels); err != nil {
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
	for _, port := range applicationServicePorts(spec) {
		if port.Port <= 0 || port.Port > 65535 {
			return fmt.Errorf("service port must be between 1 and 65535")
		}
	}
	if _, err := resourceRequirements(spec); err != nil {
		return err
	}
	if _, err := applicationPodSecurityContext(spec); err != nil {
		return err
	}
	if _, err := applicationContainerSecurityContext(spec); err != nil {
		return err
	}
	if _, err := applicationNodeSelector(spec); err != nil {
		return err
	}
	if _, err := applicationTolerations(spec); err != nil {
		return err
	}
	if _, err := applicationAffinity(spec); err != nil {
		return err
	}
	if _, err := applicationTopologySpreadConstraints(spec); err != nil {
		return err
	}
	if _, err := applicationProbe(spec.ReadinessProbe, "readiness probe"); err != nil {
		return err
	}
	if _, err := applicationProbe(spec.LivenessProbe, "liveness probe"); err != nil {
		return err
	}
	if _, err := applicationProbe(spec.StartupProbe, "startup probe"); err != nil {
		return err
	}
	if _, err := applicationLifecycle(spec); err != nil {
		return err
	}
	if _, err := applicationAuxContainers(spec.InitContainers, "init containers", spec, nil); err != nil {
		return err
	}
	if _, err := applicationAuxContainers(spec.SidecarContainers, "sidecar containers", spec, nil); err != nil {
		return err
	}
	if _, err := applicationStringList(spec.ContainerCommand, "container command"); err != nil {
		return err
	}
	if _, err := applicationStringList(spec.ContainerArgs, "container args"); err != nil {
		return err
	}
	if _, err := applicationStringList(spec.CapabilityAdd, "capability add"); err != nil {
		return err
	}
	if _, err := applicationStringList(spec.CapabilityDrop, "capability drop"); err != nil {
		return err
	}
	if _, err := applicationServiceAnnotations(spec); err != nil {
		return err
	}
	if err := validateApplicationAutoScaling(spec); err != nil {
		return err
	}
	if _, err := applicationAutoScalingBehavior(spec); err != nil {
		return err
	}
	if spec.DataRetentionEnabled {
		for _, volume := range persistentDataVolumes(spec) {
			if !dataVolumeNeedsPVC(volume) {
				continue
			}
			if _, err := persistentDataCapacity(volume); err != nil {
				return err
			}
		}
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

func (c *Client) applyConfigFilesConfigMap(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	if len(spec.ConfigFiles) == 0 {
		return c.deleteConfigMapIfExists(ctx, spec.Namespace, spec.Name+"-config-files")
	}
	data := make(map[string]string, len(spec.ConfigFiles))
	for _, file := range spec.ConfigFiles {
		data[file.Key] = file.Content
	}
	item := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: spec.Name + "-config-files", Namespace: spec.Namespace, Labels: labels}, Data: data}
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

func (c *Client) applySecretFilesSecret(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	if len(spec.SecretFiles) == 0 {
		return c.deleteSecretIfExists(ctx, spec.Namespace, spec.Name+"-secret-files")
	}
	data := make(map[string][]byte, len(spec.SecretFiles))
	for _, file := range spec.SecretFiles {
		data[file.Key] = []byte(file.Content)
	}
	item := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: spec.Name + "-secret-files", Namespace: spec.Namespace, Labels: labels}, Type: corev1.SecretTypeOpaque, Data: data}
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

func (c *Client) deleteConfigMapIfExists(ctx context.Context, namespace string, name string) error {
	err := c.client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) deleteSecretIfExists(ctx context.Context, namespace string, name string) error {
	err := c.client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) applyPersistentDataVolume(ctx context.Context, spec ApplicationResourcesSpec, volume ApplicationDataVolume, labels map[string]string) error {
	capacity, err := persistentDataCapacity(volume)
	if err != nil {
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: persistentDataPVCName(spec, volume), Namespace: spec.Namespace, Labels: labels},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{persistentDataAccessMode(spec)},
			Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: capacity}},
		},
	}
	if mode := persistentDataVolumeMode(spec); mode != "" {
		pvc.Spec.VolumeMode = &mode
	}
	if storageClassName := strings.TrimSpace(spec.DataStorageClassName); storageClassName != "" {
		pvc.Spec.StorageClassName = &storageClassName
	}
	existing, err := c.client.CoreV1().PersistentVolumeClaims(spec.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.CoreV1().PersistentVolumeClaims(spec.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = pvc.Labels
	existing.Spec.Resources.Requests[corev1.ResourceStorage] = capacity
	_, err = c.client.CoreV1().PersistentVolumeClaims(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) applyApplicationWorkload(ctx context.Context, spec ApplicationResourcesSpec, objectLabels map[string]string, selectorLabels map[string]string) (map[string]string, error) {
	switch applicationWorkloadType(spec) {
	case "StatefulSet":
		effectiveSelectorLabels, err := c.applyStatefulSet(ctx, spec, objectLabels, selectorLabels)
		if err != nil {
			return nil, err
		}
		return effectiveSelectorLabels, c.deleteStaleApplicationDeployment(ctx, spec.Namespace, spec.Name)
	default:
		effectiveSelectorLabels, err := c.applyDeployment(ctx, spec, objectLabels, selectorLabels)
		if err != nil {
			return nil, err
		}
		return effectiveSelectorLabels, c.deleteStaleApplicationStatefulSet(ctx, spec.Namespace, spec.Name)
	}
}

func (c *Client) applyDeployment(ctx context.Context, spec ApplicationResourcesSpec, objectLabels map[string]string, selectorLabels map[string]string) (map[string]string, error) {
	replicas := spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	progressDeadlineSeconds := spec.RolloutTimeoutSeconds
	if progressDeadlineSeconds <= 0 {
		progressDeadlineSeconds = 600
	}
	template := applicationPodTemplate(spec, objectLabels, selectorLabels)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: objectLabels},
		Spec: appsv1.DeploymentSpec{
			Replicas:                &replicas,
			Selector:                &metav1.LabelSelector{MatchLabels: selectorLabels},
			ProgressDeadlineSeconds: &progressDeadlineSeconds,
			Template:                template,
		},
	}
	existing, err := c.client.AppsV1().Deployments(spec.Namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.AppsV1().Deployments(spec.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
		return selectorLabels, err
	}
	if err != nil {
		return nil, err
	}
	effectiveSelectorLabels := deploymentSelectorLabels(existing, selectorLabels)
	existing.Labels = objectLabels
	existing.Spec = deployment.Spec
	existing.Spec.Selector = &metav1.LabelSelector{MatchLabels: effectiveSelectorLabels}
	existing.Spec.Template.Labels = appPodTemplateLabels(objectLabels, effectiveSelectorLabels)
	_, err = c.client.AppsV1().Deployments(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return effectiveSelectorLabels, err
}

func (c *Client) applyStatefulSet(ctx context.Context, spec ApplicationResourcesSpec, objectLabels map[string]string, selectorLabels map[string]string) (map[string]string, error) {
	replicas := spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: objectLabels},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: spec.Name,
			Selector:    &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template:    applicationPodTemplate(spec, objectLabels, selectorLabels),
		},
	}
	existing, err := c.client.AppsV1().StatefulSets(spec.Namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.AppsV1().StatefulSets(spec.Namespace).Create(ctx, statefulSet, metav1.CreateOptions{})
		return selectorLabels, err
	}
	if err != nil {
		return nil, err
	}
	effectiveSelectorLabels := statefulSetSelectorLabels(existing, selectorLabels)
	existing.Labels = objectLabels
	existing.Spec = statefulSet.Spec
	existing.Spec.Selector = &metav1.LabelSelector{MatchLabels: effectiveSelectorLabels}
	existing.Spec.Template.Labels = appPodTemplateLabels(objectLabels, effectiveSelectorLabels)
	_, err = c.client.AppsV1().StatefulSets(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return effectiveSelectorLabels, err
}

func applicationPodTemplate(spec ApplicationResourcesSpec, objectLabels map[string]string, selectorLabels map[string]string) corev1.PodTemplateSpec {
	container := corev1.Container{
		Name:            "app",
		Image:           spec.Image,
		ImagePullPolicy: applicationImagePullPolicy(spec),
		Command:         mustApplicationStringList(spec.ContainerCommand),
		Args:            mustApplicationStringList(spec.ContainerArgs),
		Ports:           containerPorts(spec),
		EnvFrom:         applicationEnvFrom(spec),
		Resources:       mustResourceRequirements(spec),
		SecurityContext: mustApplicationContainerSecurityContext(spec),
		Lifecycle:       mustApplicationLifecycle(spec),
		ReadinessProbe:  mustApplicationProbe(spec.ReadinessProbe, "readiness probe"),
		LivenessProbe:   mustApplicationProbe(spec.LivenessProbe, "liveness probe"),
		StartupProbe:    mustApplicationProbe(spec.StartupProbe, "startup probe"),
	}
	volumes := []corev1.Volume{}
	if len(spec.ConfigFiles) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "config-files",
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: spec.Name + "-config-files"},
				Items:                configFileKeyPaths(spec.ConfigFiles),
			}},
		})
		for _, file := range spec.ConfigFiles {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: "config-files", MountPath: file.Path, SubPath: file.Key, ReadOnly: true})
		}
	}
	if len(spec.SecretFiles) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "secret-files",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName: spec.Name + "-secret-files",
				Items:      configFileKeyPaths(spec.SecretFiles),
			}},
		})
		for _, file := range spec.SecretFiles {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: "secret-files", MountPath: file.Path, SubPath: file.Key, ReadOnly: true})
		}
	}
	if spec.DataRetentionEnabled {
		for _, dataVolume := range persistentDataVolumes(spec) {
			volumeName := persistentDataVolumeName(dataVolume)
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: dataVolume.MountPath})
			volumes = append(volumes, applicationDataVolumeSource(spec, dataVolume, volumeName))
		}
	}
	availableVolumeNames := map[string]bool{}
	for _, volume := range volumes {
		availableVolumeNames[volume.Name] = true
	}
	initContainers := mustApplicationAuxContainers(spec.InitContainers, "init containers", spec, availableVolumeNames)
	sidecarContainers := mustApplicationAuxContainers(spec.SidecarContainers, "sidecar containers", spec, availableVolumeNames)
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: appPodTemplateLabels(objectLabels, selectorLabels), Annotations: appPodTemplateAnnotations(spec)},
		Spec: corev1.PodSpec{
			InitContainers:               initContainers,
			Containers:                   append([]corev1.Container{container}, sidecarContainers...),
			Volumes:                      volumes,
			SecurityContext:              mustApplicationPodSecurityContext(spec),
			NodeSelector:                 mustApplicationNodeSelector(spec),
			Tolerations:                  mustApplicationTolerations(spec),
			Affinity:                     mustApplicationAffinity(spec),
			TopologySpreadConstraints:    mustApplicationTopologySpreadConstraints(spec),
			PriorityClassName:            strings.TrimSpace(spec.PriorityClassName),
			AutomountServiceAccountToken: nil,
		},
	}
}

func configFileKeyPaths(files []ApplicationConfigFile) []corev1.KeyToPath {
	items := make([]corev1.KeyToPath, 0, len(files))
	for _, file := range files {
		items = append(items, corev1.KeyToPath{Key: file.Key, Path: file.Key})
	}
	return items
}

func (c *Client) StreamDataArchive(ctx context.Context, spec DataExportSpec, output io.Writer) error {
	if c.restConfig == nil {
		return fmt.Errorf("kubernetes rest config is required")
	}
	exportVolumes := spec.Volumes
	if len(exportVolumes) == 0 && strings.TrimSpace(spec.PVCName) != "" {
		exportVolumes = []DataExportVolume{{Name: "data", PVCName: spec.PVCName}}
	}
	if strings.TrimSpace(spec.Name) == "" || strings.TrimSpace(spec.Namespace) == "" || len(exportVolumes) == 0 {
		return fmt.Errorf("data export name, namespace and pvc name are required")
	}
	for _, volume := range exportVolumes {
		if strings.TrimSpace(volume.Name) == "" || strings.TrimSpace(volume.PVCName) == "" {
			return fmt.Errorf("data export volume name and pvc name are required")
		}
	}
	podName := dnsLabel(firstNonEmpty(spec.Name, "data-export"))
	singleVolume := len(exportVolumes) == 1
	mountPath := firstNonEmpty(spec.MountPath, "/data")
	volumeMounts := make([]corev1.VolumeMount, 0, len(exportVolumes))
	volumes := make([]corev1.Volume, 0, len(exportVolumes))
	for _, volume := range exportVolumes {
		name := persistentDataVolumeName(ApplicationDataVolume{Name: volume.Name})
		targetMountPath := mountPath
		if !singleVolume {
			targetMountPath = "/mnt/" + name
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: name, MountPath: targetMountPath, ReadOnly: true})
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volume.PVCName,
				ReadOnly:  true,
			}},
		})
	}
	tarRoot := mountPath
	if !singleVolume {
		tarRoot = "/mnt"
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: spec.Namespace, Labels: baseManagedLabels(podName)},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: boolPtr(false),
			RestartPolicy:                corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:         "export",
				Image:        "busybox:1.36",
				Command:      []string{"sh", "-c", "sleep 300"},
				VolumeMounts: volumeMounts,
			}},
			Volumes: volumes,
		},
	}
	_ = c.client.CoreV1().Pods(spec.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if _, err := c.client.CoreV1().Pods(spec.Namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		return err
	}
	defer c.client.CoreV1().Pods(spec.Namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err := c.waitForPodRunning(ctx, spec.Namespace, podName); err != nil {
		return err
	}
	req := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(spec.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "export",
			Command:   []string{"tar", "czf", "-", "-C", tarRoot, "."},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(c.restConfig, "POST", req.URL())
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: output, Stderr: &stderr}); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func (c *Client) waitForPodRunning(ctx context.Context, namespace string, name string) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := c.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return false, fmt.Errorf("export pod finished before streaming: %s", pod.Status.Phase)
		}
		return pod.Status.Phase == corev1.PodRunning, nil
	})
}

func applicationServicePorts(spec ApplicationResourcesSpec) []ApplicationServicePort {
	ports := make([]ApplicationServicePort, 0, len(spec.ServicePorts))
	seen := map[int32]bool{}
	seenNames := map[string]int{}
	for _, item := range spec.ServicePorts {
		if item.Port <= 0 || item.Port > 65535 || seen[item.Port] {
			continue
		}
		seen[item.Port] = true
		name := dnsLabel(firstNonEmpty(strings.TrimSpace(item.Name), fmt.Sprintf("port-%d", item.Port)))
		if name == "" {
			name = fmt.Sprintf("port-%d", item.Port)
		}
		if count := seenNames[name]; count > 0 {
			suffix := fmt.Sprintf("-%d", count+1)
			if len(name)+len(suffix) > 63 {
				name = strings.Trim(name[:63-len(suffix)], "-")
			}
			name = fmt.Sprintf("%s%s", name, suffix)
		}
		seenNames[name]++
		ports = append(ports, ApplicationServicePort{Name: name, Port: item.Port, AppProtocol: strings.TrimSpace(item.AppProtocol)})
	}
	if len(ports) == 0 {
		port := spec.ServicePort
		if port <= 0 {
			port = 8080
		}
		ports = append(ports, ApplicationServicePort{Name: "http", Port: port})
	}
	return ports
}

func containerPorts(spec ApplicationResourcesSpec) []corev1.ContainerPort {
	ports := applicationServicePorts(spec)
	result := make([]corev1.ContainerPort, 0, len(ports))
	for _, item := range ports {
		result = append(result, corev1.ContainerPort{Name: item.Name, ContainerPort: item.Port})
	}
	return result
}

func servicePorts(spec ApplicationResourcesSpec) []corev1.ServicePort {
	ports := applicationServicePorts(spec)
	result := make([]corev1.ServicePort, 0, len(ports))
	for _, item := range ports {
		result = append(result, corev1.ServicePort{
			Name:        item.Name,
			Port:        item.Port,
			TargetPort:  intstrFromInt32(item.Port),
			AppProtocol: stringPtrOrNil(item.AppProtocol),
		})
	}
	return result
}

func (c *Client) applyService(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string, selectorLabels map[string]string) error {
	annotations := mustApplicationServiceAnnotations(spec)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels, Annotations: annotations},
		Spec: corev1.ServiceSpec{
			Type:                  applicationServiceType(spec),
			Selector:              selectorLabels,
			Ports:                 servicePorts(spec),
			ExternalTrafficPolicy: applicationServiceExternalTrafficPolicy(spec),
			SessionAffinity:       applicationServiceSessionAffinity(spec),
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
	existing.Annotations = annotations
	existing.Spec.Type = service.Spec.Type
	existing.Spec.Selector = selectorLabels
	existing.Spec.Ports = service.Spec.Ports
	existing.Spec.ExternalTrafficPolicy = service.Spec.ExternalTrafficPolicy
	existing.Spec.SessionAffinity = service.Spec.SessionAffinity
	_, err = c.client.CoreV1().Services(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) applyApplicationAutoScaling(ctx context.Context, spec ApplicationResourcesSpec, labels map[string]string) error {
	if !spec.AutoScalingEnabled {
		err := c.client.AutoscalingV2().HorizontalPodAutoscalers(spec.Namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	behavior, err := applicationAutoScalingBehavior(spec)
	if err != nil {
		return err
	}
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: spec.Namespace, Labels: labels},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       applicationWorkloadType(spec),
				Name:       spec.Name,
			},
			MinReplicas: int32Ptr(autoScalingMinReplicas(spec)),
			MaxReplicas: autoScalingMaxReplicas(spec),
			Metrics:     autoScalingMetrics(spec),
			Behavior:    behavior,
		},
	}
	existing, err := c.client.AutoscalingV2().HorizontalPodAutoscalers(spec.Namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.client.AutoscalingV2().HorizontalPodAutoscalers(spec.Namespace).Create(ctx, hpa, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	existing.Labels = labels
	existing.Spec = hpa.Spec
	_, err = c.client.AutoscalingV2().HorizontalPodAutoscalers(spec.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
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
			BackoffLimit:            &backoffLimit,
			ActiveDeadlineSeconds:   int64Ptr(int64(timeout)),
			TTLSecondsAfterFinished: int32Ptr(hookJobFailureTTLSeconds),
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
	createdJob, err := c.client.BatchV1().Jobs(spec.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return HookJobResult{}, err
	}
	_ = c.attachHookScriptOwner(ctx, spec.Namespace, scriptMapName, createdJob)
	result, err := c.waitForHookJob(ctx, spec.Namespace, spec.Name, time.Duration(timeout)*time.Second)
	if err != nil {
		return result, err
	}
	ttlSeconds := hookJobFailureTTLSeconds
	if result.Succeeded {
		ttlSeconds = hookJobSuccessTTLSeconds
	}
	_ = c.setHookJobTTL(ctx, spec.Namespace, spec.Name, ttlSeconds)
	return result, nil
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

func (c *Client) attachHookScriptOwner(ctx context.Context, namespace string, name string, owner *batchv1.Job) error {
	item, err := c.client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	item.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(owner, batchv1.SchemeGroupVersion.WithKind("Job"))}
	_, err = c.client.CoreV1().ConfigMaps(namespace).Update(ctx, item, metav1.UpdateOptions{})
	return err
}

func (c *Client) setHookJobTTL(ctx context.Context, namespace string, name string, ttlSeconds int32) error {
	job, err := c.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	job.Spec.TTLSecondsAfterFinished = int32Ptr(ttlSeconds)
	_, err = c.client.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
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
	setLabel(labels, DeploymentTargetIDLabel, spec.DeploymentTargetID)
	return labels
}

func appObjectLabels(spec ApplicationResourcesSpec) map[string]string {
	labels := appSelectorLabels(spec)
	setLabel(labels, ProjectIDLabel, spec.ProjectID)
	setLabel(labels, ApplicationIDLabel, spec.ApplicationID)
	setLabel(labels, EnvironmentIDLabel, spec.EnvironmentID)
	setLabel(labels, DeploymentTargetIDLabel, spec.DeploymentTargetID)
	setLabel(labels, ReleaseIDLabel, spec.ReleaseID)
	return labels
}

func appPodTemplateAnnotations(spec ApplicationResourcesSpec) map[string]string {
	annotations := map[string]string{}
	setLabel(annotations, ReleaseIDLabel, spec.ReleaseID)
	setLabel(annotations, BuildRunIDLabel, spec.BuildRunID)
	setLabel(annotations, ImageDigestLabel, spec.ImageDigest)
	return annotations
}

func appPodTemplateLabels(objectLabels map[string]string, selectorLabels map[string]string) map[string]string {
	labels := cloneStringMap(objectLabels)
	for key, value := range selectorLabels {
		labels[key] = value
	}
	return labels
}

func deploymentSelectorLabels(existing *appsv1.Deployment, fallback map[string]string) map[string]string {
	if existing != nil && existing.Spec.Selector != nil && len(existing.Spec.Selector.MatchLabels) > 0 {
		return cloneStringMap(existing.Spec.Selector.MatchLabels)
	}
	return cloneStringMap(fallback)
}

func statefulSetSelectorLabels(existing *appsv1.StatefulSet, fallback map[string]string) map[string]string {
	if existing != nil && existing.Spec.Selector != nil && len(existing.Spec.Selector.MatchLabels) > 0 {
		return cloneStringMap(existing.Spec.Selector.MatchLabels)
	}
	return cloneStringMap(fallback)
}

func applicationWorkloadType(spec ApplicationResourcesSpec) string {
	switch strings.ToLower(strings.TrimSpace(spec.WorkloadType)) {
	case "statefulset", "stateful-set":
		return "StatefulSet"
	default:
		return "Deployment"
	}
}

func (c *Client) deleteStaleApplicationDeployment(ctx context.Context, namespace string, name string) error {
	err := c.client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) deleteStaleApplicationStatefulSet(ctx context.Context, namespace string, name string) error {
	err := c.client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func applicationEnvFrom(spec ApplicationResourcesSpec) []corev1.EnvFromSource {
	return []corev1.EnvFromSource{
		{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: spec.Name + "-config"}}},
		{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: spec.Name + "-secret"}}},
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func applicationImagePullPolicy(spec ApplicationResourcesSpec) corev1.PullPolicy {
	if spec.ForceImagePull {
		return corev1.PullAlways
	}
	switch strings.TrimSpace(spec.ImagePullPolicy) {
	case string(corev1.PullAlways):
		return corev1.PullAlways
	case string(corev1.PullNever):
		return corev1.PullNever
	case string(corev1.PullIfNotPresent):
		return corev1.PullIfNotPresent
	}
	return corev1.PullIfNotPresent
}

func applicationLifecycle(spec ApplicationResourcesSpec) (*corev1.Lifecycle, error) {
	raw := strings.TrimSpace(spec.Lifecycle)
	if raw == "" {
		return nil, nil
	}
	var lifecycle corev1.Lifecycle
	if err := json.Unmarshal([]byte(raw), &lifecycle); err != nil {
		return nil, fmt.Errorf("invalid lifecycle: %w", err)
	}
	return &lifecycle, nil
}

func mustApplicationLifecycle(spec ApplicationResourcesSpec) *corev1.Lifecycle {
	lifecycle, err := applicationLifecycle(spec)
	if err != nil {
		panic(err)
	}
	return lifecycle
}

func applicationAuxContainers(raw string, label string, spec ApplicationResourcesSpec, availableVolumeNames map[string]bool) ([]corev1.Container, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var input []corev1.Container
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", label, err)
	}
	output := make([]corev1.Container, 0, len(input))
	for _, item := range input {
		name := dnsLabel(item.Name)
		if name == "" || strings.TrimSpace(item.Image) == "" {
			return nil, fmt.Errorf("%s requires name and image", label)
		}
		securityContext, err := allowedAuxSecurityContext(item.SecurityContext, label)
		if err != nil {
			return nil, err
		}
		container := corev1.Container{
			Name:            name,
			Image:           strings.TrimSpace(item.Image),
			ImagePullPolicy: applicationImagePullPolicy(spec),
			Command:         compactStringList(item.Command),
			Args:            compactStringList(item.Args),
			Ports:           allowedAuxContainerPorts(item.Ports),
			Resources:       item.Resources,
			SecurityContext: securityContext,
			Env:             allowedAuxEnvVars(item.Env),
			EnvFrom:         applicationEnvFrom(spec),
			VolumeMounts:    allowedVolumeMounts(item.VolumeMounts, availableVolumeNames),
		}
		output = append(output, container)
	}
	return output, nil
}

func mustApplicationAuxContainers(raw string, label string, spec ApplicationResourcesSpec, availableVolumeNames map[string]bool) []corev1.Container {
	containers, err := applicationAuxContainers(raw, label, spec, availableVolumeNames)
	if err != nil {
		panic(err)
	}
	return containers
}

func allowedAuxContainerPorts(input []corev1.ContainerPort) []corev1.ContainerPort {
	if len(input) == 0 {
		return nil
	}
	output := make([]corev1.ContainerPort, 0, len(input))
	for _, item := range input {
		if item.ContainerPort <= 0 || item.ContainerPort > 65535 {
			continue
		}
		output = append(output, corev1.ContainerPort{
			Name:          dnsLabel(item.Name),
			ContainerPort: item.ContainerPort,
			Protocol:      item.Protocol,
		})
	}
	return output
}

func allowedAuxEnvVars(input []corev1.EnvVar) []corev1.EnvVar {
	if len(input) == 0 {
		return nil
	}
	output := make([]corev1.EnvVar, 0, len(input))
	for _, item := range input {
		name := strings.TrimSpace(item.Name)
		if name == "" || item.ValueFrom != nil {
			continue
		}
		output = append(output, corev1.EnvVar{Name: name, Value: item.Value})
	}
	return output
}

func allowedAuxSecurityContext(input *corev1.SecurityContext, label string) (*corev1.SecurityContext, error) {
	if input == nil {
		return nil, nil
	}
	if input.Privileged != nil && *input.Privileged {
		return nil, fmt.Errorf("%s cannot enable privileged", label)
	}
	if input.AllowPrivilegeEscalation != nil && *input.AllowPrivilegeEscalation {
		return nil, fmt.Errorf("%s cannot enable privilege escalation", label)
	}
	if input.Capabilities != nil && len(input.Capabilities.Add) > 0 {
		return nil, fmt.Errorf("%s cannot add Linux capabilities", label)
	}
	context := &corev1.SecurityContext{}
	hasValue := false
	if input.RunAsUser != nil {
		context.RunAsUser = input.RunAsUser
		hasValue = true
	}
	if input.RunAsGroup != nil {
		context.RunAsGroup = input.RunAsGroup
		hasValue = true
	}
	if input.RunAsNonRoot != nil {
		context.RunAsNonRoot = input.RunAsNonRoot
		hasValue = true
	}
	if input.ReadOnlyRootFilesystem != nil {
		context.ReadOnlyRootFilesystem = input.ReadOnlyRootFilesystem
		hasValue = true
	}
	if input.AllowPrivilegeEscalation != nil {
		context.AllowPrivilegeEscalation = input.AllowPrivilegeEscalation
		hasValue = true
	}
	if input.Capabilities != nil && len(input.Capabilities.Drop) > 0 {
		context.Capabilities = &corev1.Capabilities{Drop: input.Capabilities.Drop}
		hasValue = true
	}
	if input.SeccompProfile != nil {
		context.SeccompProfile = input.SeccompProfile
		hasValue = true
	}
	if !hasValue {
		return nil, nil
	}
	return context, nil
}

func allowedVolumeMounts(input []corev1.VolumeMount, available map[string]bool) []corev1.VolumeMount {
	if len(input) == 0 || len(available) == 0 {
		return nil
	}
	output := make([]corev1.VolumeMount, 0, len(input))
	for _, item := range input {
		if available[item.Name] && strings.TrimSpace(item.MountPath) != "" {
			output = append(output, corev1.VolumeMount{
				Name:      item.Name,
				ReadOnly:  item.ReadOnly,
				MountPath: item.MountPath,
				SubPath:   item.SubPath,
			})
		}
	}
	return output
}

func validateApplicationAutoScaling(spec ApplicationResourcesSpec) error {
	if !spec.AutoScalingEnabled {
		return nil
	}
	if autoScalingMaxReplicas(spec) < autoScalingMinReplicas(spec) {
		return fmt.Errorf("hpa max replicas must be greater than or equal to min replicas")
	}
	if len(autoScalingMetrics(spec)) == 0 {
		return fmt.Errorf("hpa requires at least one cpu or memory metric")
	}
	return nil
}

func autoScalingMinReplicas(spec ApplicationResourcesSpec) int32 {
	if spec.AutoScalingMinReplicas > 0 {
		return spec.AutoScalingMinReplicas
	}
	if spec.Replicas > 0 {
		return spec.Replicas
	}
	return 1
}

func autoScalingMaxReplicas(spec ApplicationResourcesSpec) int32 {
	if spec.AutoScalingMaxReplicas > 0 {
		return spec.AutoScalingMaxReplicas
	}
	return autoScalingMinReplicas(spec)
}

func autoScalingMetrics(spec ApplicationResourcesSpec) []autoscalingv2.MetricSpec {
	metrics := []autoscalingv2.MetricSpec{}
	if spec.AutoScalingCPUPercent > 0 {
		metrics = append(metrics, autoScalingResourceMetric(corev1.ResourceCPU, spec.AutoScalingCPUPercent))
	}
	if spec.AutoScalingMemoryPercent > 0 {
		metrics = append(metrics, autoScalingResourceMetric(corev1.ResourceMemory, spec.AutoScalingMemoryPercent))
	}
	return metrics
}

func applicationAutoScalingBehavior(spec ApplicationResourcesSpec) (*autoscalingv2.HorizontalPodAutoscalerBehavior, error) {
	raw := strings.TrimSpace(spec.AutoScalingBehavior)
	if raw == "" {
		return nil, nil
	}
	var behavior autoscalingv2.HorizontalPodAutoscalerBehavior
	if err := json.Unmarshal([]byte(raw), &behavior); err != nil {
		return nil, fmt.Errorf("invalid hpa behavior: %w", err)
	}
	return &behavior, nil
}

func autoScalingResourceMetric(name corev1.ResourceName, utilization int32) autoscalingv2.MetricSpec {
	return autoscalingv2.MetricSpec{
		Type: autoscalingv2.ResourceMetricSourceType,
		Resource: &autoscalingv2.ResourceMetricSource{
			Name: name,
			Target: autoscalingv2.MetricTarget{
				Type:               autoscalingv2.UtilizationMetricType,
				AverageUtilization: int32Ptr(utilization),
			},
		},
	}
}

func applicationPodSecurityContext(spec ApplicationResourcesSpec) (*corev1.PodSecurityContext, error) {
	context := &corev1.PodSecurityContext{}
	hasValue := false
	if value, ok, err := optionalInt64(spec.RunAsUser); err != nil {
		return nil, fmt.Errorf("invalid runAsUser: %w", err)
	} else if ok {
		context.RunAsUser = &value
		hasValue = true
	}
	if value, ok, err := optionalInt64(spec.RunAsGroup); err != nil {
		return nil, fmt.Errorf("invalid runAsGroup: %w", err)
	} else if ok {
		context.RunAsGroup = &value
		hasValue = true
	}
	if value, ok, err := optionalInt64(spec.FSGroup); err != nil {
		return nil, fmt.Errorf("invalid fsGroup: %w", err)
	} else if ok {
		context.FSGroup = &value
		hasValue = true
	}
	if policy := strings.TrimSpace(spec.FSGroupChangePolicy); policy != "" {
		value := corev1.PodFSGroupChangePolicy(policy)
		context.FSGroupChangePolicy = &value
		hasValue = true
	}
	if !hasValue {
		return nil, nil
	}
	return context, nil
}

func mustApplicationPodSecurityContext(spec ApplicationResourcesSpec) *corev1.PodSecurityContext {
	context, err := applicationPodSecurityContext(spec)
	if err != nil {
		panic(err)
	}
	return context
}

func applicationContainerSecurityContext(spec ApplicationResourcesSpec) (*corev1.SecurityContext, error) {
	context := &corev1.SecurityContext{}
	hasValue := false
	if spec.ReadOnlyRootFilesystem {
		context.ReadOnlyRootFilesystem = boolPtr(true)
		hasValue = true
	}
	if value, ok, err := optionalBool(spec.AllowPrivilegeEscalation); err != nil {
		return nil, fmt.Errorf("invalid allowPrivilegeEscalation: %w", err)
	} else if ok {
		context.AllowPrivilegeEscalation = &value
		hasValue = true
	}
	add, err := applicationStringList(spec.CapabilityAdd, "capability add")
	if err != nil {
		return nil, err
	}
	drop, err := applicationStringList(spec.CapabilityDrop, "capability drop")
	if err != nil {
		return nil, err
	}
	if len(add) > 0 || len(drop) > 0 {
		context.Capabilities = &corev1.Capabilities{
			Add:  capabilityNames(add),
			Drop: capabilityNames(drop),
		}
		hasValue = true
	}
	if !hasValue {
		return nil, nil
	}
	return context, nil
}

func mustApplicationContainerSecurityContext(spec ApplicationResourcesSpec) *corev1.SecurityContext {
	context, err := applicationContainerSecurityContext(spec)
	if err != nil {
		panic(err)
	}
	return context
}

func capabilityNames(values []string) []corev1.Capability {
	output := make([]corev1.Capability, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			output = append(output, corev1.Capability(value))
		}
	}
	return output
}

func applicationProbe(raw string, label string) (*corev1.Probe, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var probe corev1.Probe
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", label, err)
	}
	return &probe, nil
}

func mustApplicationProbe(raw string, label string) *corev1.Probe {
	probe, err := applicationProbe(raw, label)
	if err != nil {
		panic(err)
	}
	return probe
}

func applicationNodeSelector(spec ApplicationResourcesSpec) (map[string]string, error) {
	return stringMapFromJSONOrLines(spec.NodeSelector, "node selector")
}

func mustApplicationNodeSelector(spec ApplicationResourcesSpec) map[string]string {
	values, err := applicationNodeSelector(spec)
	if err != nil {
		panic(err)
	}
	return values
}

func applicationTolerations(spec ApplicationResourcesSpec) ([]corev1.Toleration, error) {
	raw := strings.TrimSpace(spec.Tolerations)
	if raw == "" {
		return nil, nil
	}
	var values []corev1.Toleration
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("invalid tolerations: %w", err)
	}
	return values, nil
}

func mustApplicationTolerations(spec ApplicationResourcesSpec) []corev1.Toleration {
	values, err := applicationTolerations(spec)
	if err != nil {
		panic(err)
	}
	return values
}

func applicationAffinity(spec ApplicationResourcesSpec) (*corev1.Affinity, error) {
	raw := strings.TrimSpace(spec.Affinity)
	if raw == "" {
		return nil, nil
	}
	var value corev1.Affinity
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("invalid affinity: %w", err)
	}
	return &value, nil
}

func mustApplicationAffinity(spec ApplicationResourcesSpec) *corev1.Affinity {
	value, err := applicationAffinity(spec)
	if err != nil {
		panic(err)
	}
	return value
}

func applicationTopologySpreadConstraints(spec ApplicationResourcesSpec) ([]corev1.TopologySpreadConstraint, error) {
	raw := strings.TrimSpace(spec.TopologySpreadConstraints)
	if raw == "" {
		return nil, nil
	}
	var values []corev1.TopologySpreadConstraint
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("invalid topology spread constraints: %w", err)
	}
	return values, nil
}

func mustApplicationTopologySpreadConstraints(spec ApplicationResourcesSpec) []corev1.TopologySpreadConstraint {
	values, err := applicationTopologySpreadConstraints(spec)
	if err != nil {
		panic(err)
	}
	return values
}

func applicationServiceType(spec ApplicationResourcesSpec) corev1.ServiceType {
	switch strings.TrimSpace(spec.ServiceType) {
	case string(corev1.ServiceTypeNodePort):
		return corev1.ServiceTypeNodePort
	case string(corev1.ServiceTypeLoadBalancer):
		return corev1.ServiceTypeLoadBalancer
	default:
		return corev1.ServiceTypeClusterIP
	}
}

func applicationServiceExternalTrafficPolicy(spec ApplicationResourcesSpec) corev1.ServiceExternalTrafficPolicy {
	serviceType := applicationServiceType(spec)
	if serviceType != corev1.ServiceTypeNodePort && serviceType != corev1.ServiceTypeLoadBalancer {
		return ""
	}
	switch strings.TrimSpace(spec.ServiceExternalTrafficPolicy) {
	case string(corev1.ServiceExternalTrafficPolicyLocal):
		return corev1.ServiceExternalTrafficPolicyLocal
	case string(corev1.ServiceExternalTrafficPolicyCluster):
		return corev1.ServiceExternalTrafficPolicyCluster
	default:
		return ""
	}
}

func applicationServiceSessionAffinity(spec ApplicationResourcesSpec) corev1.ServiceAffinity {
	switch strings.TrimSpace(spec.ServiceSessionAffinity) {
	case string(corev1.ServiceAffinityClientIP):
		return corev1.ServiceAffinityClientIP
	default:
		return corev1.ServiceAffinityNone
	}
}

func applicationServiceAnnotations(spec ApplicationResourcesSpec) (map[string]string, error) {
	return stringMapFromJSONOrLines(spec.ServiceAnnotations, "service annotations")
}

func mustApplicationServiceAnnotations(spec ApplicationResourcesSpec) map[string]string {
	values, err := applicationServiceAnnotations(spec)
	if err != nil {
		panic(err)
	}
	return values
}

func persistentDataAccessMode(spec ApplicationResourcesSpec) corev1.PersistentVolumeAccessMode {
	switch strings.TrimSpace(spec.DataAccessMode) {
	case string(corev1.ReadWriteMany):
		return corev1.ReadWriteMany
	case string(corev1.ReadOnlyMany):
		return corev1.ReadOnlyMany
	default:
		return corev1.ReadWriteOnce
	}
}

func persistentDataVolumeMode(spec ApplicationResourcesSpec) corev1.PersistentVolumeMode {
	switch strings.TrimSpace(spec.DataVolumeMode) {
	case string(corev1.PersistentVolumeBlock):
		return corev1.PersistentVolumeBlock
	case string(corev1.PersistentVolumeFilesystem):
		return corev1.PersistentVolumeFilesystem
	default:
		return ""
	}
}

func persistentDataVolumes(spec ApplicationResourcesSpec) []ApplicationDataVolume {
	if len(spec.DataVolumes) > 0 {
		volumes := make([]ApplicationDataVolume, 0, len(spec.DataVolumes))
		for _, volume := range spec.DataVolumes {
			name := firstNonEmpty(volume.Name, "data")
			volumes = append(volumes, ApplicationDataVolume{
				Name:              name,
				MountPath:         firstNonEmpty(volume.MountPath, "/data"),
				Capacity:          firstNonEmpty(volume.Capacity, "1Gi"),
				SourceType:        dataVolumeSourceType(volume),
				ExistingClaimName: strings.TrimSpace(volume.ExistingClaimName),
				EmptyDirMedium:    strings.TrimSpace(volume.EmptyDirMedium),
				EmptyDirSizeLimit: strings.TrimSpace(volume.EmptyDirSizeLimit),
			})
		}
		return volumes
	}
	return []ApplicationDataVolume{{
		Name:       "data",
		MountPath:  persistentDataMountPath(spec),
		Capacity:   firstNonEmpty(spec.DataCapacity, "1Gi"),
		SourceType: "managed",
	}}
}

func dataVolumeSourceType(volume ApplicationDataVolume) string {
	switch strings.TrimSpace(volume.SourceType) {
	case "existingClaim":
		return "existingClaim"
	case "emptyDir":
		return "emptyDir"
	default:
		return "managed"
	}
}

func dataVolumeNeedsPVC(volume ApplicationDataVolume) bool {
	return dataVolumeSourceType(volume) == "managed"
}

func applicationDataVolumeSource(spec ApplicationResourcesSpec, volume ApplicationDataVolume, name string) corev1.Volume {
	switch dataVolumeSourceType(volume) {
	case "existingClaim":
		return corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: strings.TrimSpace(volume.ExistingClaimName),
			}},
		}
	case "emptyDir":
		emptyDir := &corev1.EmptyDirVolumeSource{}
		if medium := strings.TrimSpace(volume.EmptyDirMedium); medium != "" {
			emptyDir.Medium = corev1.StorageMedium(medium)
		}
		if sizeLimit := strings.TrimSpace(volume.EmptyDirSizeLimit); sizeLimit != "" {
			if quantity, err := resource.ParseQuantity(sizeLimit); err == nil {
				emptyDir.SizeLimit = &quantity
			}
		}
		return corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{EmptyDir: emptyDir}}
	default:
		return corev1.Volume{
			Name:         name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: persistentDataPVCName(spec, volume)}},
		}
	}
}

func persistentDataPVCName(spec ApplicationResourcesSpec, volume ApplicationDataVolume) string {
	name := persistentDataVolumeName(volume)
	if name == "data" {
		return spec.Name + "-data"
	}
	return dnsLabel(spec.Name + "-" + name + "-data")
}

func persistentDataMountPath(spec ApplicationResourcesSpec) string {
	return firstNonEmpty(spec.DataMountPath, "/data")
}

func persistentDataVolumeName(volume ApplicationDataVolume) string {
	return dnsLabel(firstNonEmpty(volume.Name, "data"))
}

func persistentDataCapacity(volume ApplicationDataVolume) (resource.Quantity, error) {
	value := firstNonEmpty(volume.Capacity, "1Gi")
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return resource.Quantity{}, fmt.Errorf("invalid data capacity: %w", err)
	}
	if quantity.Sign() <= 0 {
		return resource.Quantity{}, fmt.Errorf("data capacity must be greater than zero")
	}
	return quantity, nil
}

func resourceRequirements(spec ApplicationResourcesSpec) (corev1.ResourceRequirements, error) {
	requests := corev1.ResourceList{}
	limits := corev1.ResourceList{}
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
	if spec.CPULimit != "" {
		quantity, err := resource.ParseQuantity(spec.CPULimit)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid cpu limit: %w", err)
		}
		limits[corev1.ResourceCPU] = quantity
	}
	if spec.MemoryLimit != "" {
		quantity, err := resource.ParseQuantity(spec.MemoryLimit)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid memory limit: %w", err)
		}
		limits[corev1.ResourceMemory] = quantity
	}
	requirements := corev1.ResourceRequirements{Requests: requests}
	if len(limits) > 0 {
		requirements.Limits = limits
	}
	return requirements, nil
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

func int32Ptr(value int32) *int32 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt64(value string) (int64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, false, fmt.Errorf("must be a non-negative integer")
	}
	return parsed, true, nil
}

func optionalBool(value string) (bool, bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, false, nil
	case "true":
		return true, true, nil
	case "false":
		return false, true, nil
	default:
		return false, false, fmt.Errorf("must be true or false")
	}
}

func applicationStringList(raw string, label string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "[") {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", label, err)
		}
		return compactStringList(values), nil
	}
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	return compactStringList(values), nil
}

func mustApplicationStringList(raw string) []string {
	values, err := applicationStringList(raw, "string list")
	if err != nil {
		panic(err)
	}
	return values
}

func compactStringList(values []string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			output = append(output, value)
		}
	}
	return output
}

func stringMapFromJSONOrLines(raw string, label string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "{") {
		values := map[string]string{}
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", label, err)
		}
		return compactStringMap(values), nil
	}
	values := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid %s line %q", label, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid %s empty key", label)
		}
		values[key] = strings.TrimSpace(value)
	}
	return values, nil
}

func compactStringMap(values map[string]string) map[string]string {
	output := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key != "" {
			output[key] = strings.TrimSpace(value)
		}
	}
	if len(output) == 0 {
		return nil
	}
	return output
}
