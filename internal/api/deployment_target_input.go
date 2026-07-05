package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultBuildCPURequest     = "2"
	defaultBuildMemoryRequest  = "4Gi"
	defaultBuildTimeoutSeconds = 1800
	minBuildTimeoutSeconds     = 60
	maxBuildTimeoutSeconds     = 24 * 60 * 60
)

func (h *Handlers) deploymentTargetFromInput(ctx *gin.Context, user model.User, app model.Application, input deploymentTargetInput, targetID string, existingSecretFiles map[string]string, existingRuntimeConfigRefs string) (model.DeploymentTarget, bool) {
	sourceType := normalizeDeploymentSourceType(input.SourceType)
	repositoryBindingID := strings.TrimSpace(input.RepositoryBindingID)
	if sourceType == "repository" {
		if repositoryBindingID == "" {
			writeError(ctx, http.StatusBadRequest, "代码仓库不能为空")
			return model.DeploymentTarget{}, false
		}
		var binding model.RepositoryBinding
		if err := h.db.First(&binding, "id = ? and project_id = ? and application_id = ?", repositoryBindingID, app.ProjectID, app.ID).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "代码仓库绑定不存在")
			return model.DeploymentTarget{}, false
		}
	}
	targetRepository, targetTag := splitTargetImageRef(input.TargetImageRef)
	if targetRepository == "" {
		targetRepository = strings.Trim(strings.TrimSpace(input.TargetRepository), "/")
		targetTag = strings.TrimSpace(input.TargetTag)
	}
	stage := normalizeStage(input.Stage)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = stage
	}
	buildHooksEnabled := true
	if input.BuildHooksEnabled != nil {
		buildHooksEnabled = *input.BuildHooksEnabled
	}
	dataCapacity, ok := normalizeDataCapacity(ctx, input.DataCapacity, input.DataRetentionEnabled)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	dataMountPath, ok := normalizeDataMountPath(ctx, input.DataMountPath, input.DataRetentionEnabled)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	dataVolumes, ok := normalizeDataVolumes(ctx, input.DataVolumes, input.DataRetentionEnabled, dataMountPath, dataCapacity)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	if len(dataVolumes) > 0 {
		dataMountPath = dataVolumes[0].MountPath
		dataCapacity = dataVolumes[0].Capacity
	}
	servicePorts, ok := normalizeDeploymentServicePorts(ctx, input.ServicePorts, input.ServicePort)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	servicePort := servicePorts[0].Port
	replicas := input.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	runtimeCPURequest, ok := normalizeBuildResourceQuantity(ctx, input.CPURequest, "1", "运行 CPU")
	if !ok {
		return model.DeploymentTarget{}, false
	}
	runtimeMemoryRequest, ok := normalizeBuildResourceQuantity(ctx, input.MemoryRequest, "1Gi", "运行内存")
	if !ok {
		return model.DeploymentTarget{}, false
	}
	runtimeCPULimit, ok := normalizeOptionalResourceQuantity(ctx, input.CPULimit, "运行 CPU 上限")
	if !ok {
		return model.DeploymentTarget{}, false
	}
	runtimeMemoryLimit, ok := normalizeOptionalResourceQuantity(ctx, input.MemoryLimit, "运行内存上限")
	if !ok {
		return model.DeploymentTarget{}, false
	}
	kubernetesAdvanced, ok := normalizeDeploymentKubernetesAdvanced(ctx, input)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	autoScaling, ok := normalizeDeploymentAutoScaling(ctx, input, replicas)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	buildCPURequest, ok := normalizeBuildResourceQuantity(ctx, input.BuildCPURequest, defaultBuildCPURequest, "构建 CPU")
	if !ok {
		return model.DeploymentTarget{}, false
	}
	buildMemoryRequest, ok := normalizeBuildResourceQuantity(ctx, input.BuildMemoryRequest, defaultBuildMemoryRequest, "构建内存")
	if !ok {
		return model.DeploymentTarget{}, false
	}
	buildTimeoutSeconds, ok := normalizeBuildTimeoutSeconds(ctx, input.BuildTimeoutSeconds)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	clusterID := strings.TrimSpace(input.ClusterID)
	targetRegistryID := strings.TrimSpace(input.TargetRegistryID)
	if _, ok := h.runtimeClusterForProjectUse(ctx, user, app.ProjectID, clusterID); !ok {
		return model.DeploymentTarget{}, false
	}
	targetRepository, targetTag, ok = h.applyRegistryCredentialImageTemplate(ctx, user, app, sourceType, targetRegistryID, targetRepository, targetTag, model.DeploymentTarget{
		ID:    targetID,
		Name:  name,
		Stage: stage,
	})
	if !ok {
		return model.DeploymentTarget{}, false
	}
	runtimeConfigRefs, ok := h.runtimeConfigRefsFromInput(ctx, app.ProjectID, input, existingRuntimeConfigRefs)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	runtimeConfigSetIDs := model.DeploymentRuntimeConfigLiveSetIDs(runtimeConfigRefs)
	configFiles, ok := normalizeRuntimeConfigFilesInput(ctx, input.ConfigFiles)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	secretFiles, ok := h.runtimeSecretFilesFromInput(ctx, user, targetID, input.SecretFiles, existingSecretFiles)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	secretFilesContent, err := json.Marshal(secretFiles)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return model.DeploymentTarget{}, false
	}
	for _, volume := range dataVolumes {
		if runtimeDataPathConflicts(volume.MountPath, configFiles, string(secretFilesContent)) {
			writeError(ctx, http.StatusBadRequest, "运行数据目录不能与配置文件或密钥文件挂载路径重叠")
			return model.DeploymentTarget{}, false
		}
	}
	return model.DeploymentTarget{
		ID:                           targetID,
		ProjectID:                    app.ProjectID,
		ApplicationID:                app.ID,
		EnvironmentID:                strings.TrimSpace(input.EnvironmentID),
		Name:                         name,
		Stage:                        stage,
		ClusterID:                    clusterID,
		Namespace:                    strings.TrimSpace(input.Namespace),
		WorkloadType:                 normalizeWorkloadType(input.WorkloadType),
		Replicas:                     replicas,
		CPURequest:                   runtimeCPURequest,
		MemoryRequest:                runtimeMemoryRequest,
		CPULimit:                     runtimeCPULimit,
		MemoryLimit:                  runtimeMemoryLimit,
		ImagePullPolicy:              kubernetesAdvanced.ImagePullPolicy,
		ContainerCommand:             kubernetesAdvanced.ContainerCommand,
		ContainerArgs:                kubernetesAdvanced.ContainerArgs,
		Lifecycle:                    kubernetesAdvanced.Lifecycle,
		InitContainers:               kubernetesAdvanced.InitContainers,
		SidecarContainers:            kubernetesAdvanced.SidecarContainers,
		ReadinessProbe:               kubernetesAdvanced.ReadinessProbe,
		LivenessProbe:                kubernetesAdvanced.LivenessProbe,
		StartupProbe:                 kubernetesAdvanced.StartupProbe,
		RunAsUser:                    kubernetesAdvanced.RunAsUser,
		RunAsGroup:                   kubernetesAdvanced.RunAsGroup,
		FSGroup:                      kubernetesAdvanced.FSGroup,
		FSGroupChangePolicy:          kubernetesAdvanced.FSGroupChangePolicy,
		ReadOnlyRootFilesystem:       kubernetesAdvanced.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation:     kubernetesAdvanced.AllowPrivilegeEscalation,
		CapabilityAdd:                kubernetesAdvanced.CapabilityAdd,
		CapabilityDrop:               kubernetesAdvanced.CapabilityDrop,
		NodeSelector:                 kubernetesAdvanced.NodeSelector,
		Tolerations:                  kubernetesAdvanced.Tolerations,
		Affinity:                     kubernetesAdvanced.Affinity,
		TopologySpreadConstraints:    kubernetesAdvanced.TopologySpreadConstraints,
		PriorityClassName:            kubernetesAdvanced.PriorityClassName,
		ServiceType:                  kubernetesAdvanced.ServiceType,
		ServiceAnnotations:           kubernetesAdvanced.ServiceAnnotations,
		ServiceExternalTrafficPolicy: kubernetesAdvanced.ServiceExternalTrafficPolicy,
		ServiceSessionAffinity:       kubernetesAdvanced.ServiceSessionAffinity,
		AutoScalingEnabled:           autoScaling.Enabled,
		AutoScalingMinReplicas:       autoScaling.MinReplicas,
		AutoScalingMaxReplicas:       autoScaling.MaxReplicas,
		AutoScalingCPUPercent:        autoScaling.CPUPercent,
		AutoScalingMemoryPercent:     autoScaling.MemoryPercent,
		AutoScalingBehavior:          autoScaling.Behavior,
		ServicePort:                  servicePort,
		ServicePorts:                 model.EncodeDeploymentServicePorts(servicePorts, servicePort),
		SourceType:                   sourceType,
		RepositoryBindingID:          repositoryBindingID,
		DockerfilePath:               fallback(strings.TrimSpace(input.DockerfilePath), "Dockerfile"),
		BuildContext:                 fallback(strings.TrimSpace(input.BuildContext), "."),
		BuildDirectory:               strings.TrimSpace(input.BuildDirectory),
		BuildEnvironmentID:           strings.TrimSpace(input.BuildEnvironmentID),
		BuildCPURequest:              buildCPURequest,
		BuildMemoryRequest:           buildMemoryRequest,
		BuildTimeoutSeconds:          buildTimeoutSeconds,
		TargetRegistryID:             targetRegistryID,
		TargetRepository:             targetRepository,
		TargetTag:                    fallback(targetTag, "latest"),
		ImageRef:                     strings.TrimSpace(input.ImageRef),
		BuildLabels:                  strings.Join(normalizeBuildSelectorList(strings.Split(input.BuildLabels, ",")), ","),
		BuildVariableSetIDs:          encodeBuildVariableSetIDs(input.BuildVariableSetIDs),
		BuildHooksEnabled:            buildHooksEnabled,
		AutoDeploy:                   input.AutoDeploy,
		BranchPattern:                strings.TrimSpace(input.BranchPattern),
		TagPattern:                   strings.TrimSpace(input.TagPattern),
		ConcurrencyPolicy:            normalizeBuildConcurrencyPolicy(input.ConcurrencyPolicy),
		RuntimeConfigSetIDs:          encodeBuildVariableSetIDs(runtimeConfigSetIDs),
		RuntimeConfigRefs:            model.EncodeDeploymentRuntimeConfigRefs(runtimeConfigRefs),
		EnvVars:                      strings.TrimSpace(input.EnvVars),
		ConfigRefs:                   strings.TrimSpace(input.ConfigRefs),
		SecretRefs:                   normalizeSecretRefsInput(input.SecretRefs),
		ConfigFiles:                  configFiles,
		SecretFiles:                  string(secretFilesContent),
		DataRetentionEnabled:         input.DataRetentionEnabled,
		DataCapacity:                 dataCapacity,
		DataMountPath:                dataMountPath,
		DataVolumes:                  encodeDataVolumes(dataVolumes),
		DataStorageClassName:         kubernetesAdvanced.DataStorageClassName,
		DataAccessMode:               kubernetesAdvanced.DataAccessMode,
		DataVolumeMode:               kubernetesAdvanced.DataVolumeMode,
		RequireApproval:              input.RequireApproval,
		Enabled:                      input.Enabled,
		CreatedBy:                    user.ID,
	}, true
}

func (h *Handlers) runtimeConfigRefsFromInput(ctx *gin.Context, projectID string, input deploymentTargetInput, existingRaw string) ([]model.DeploymentRuntimeConfigRef, bool) {
	refs := runtimeConfigRefInputs(input)
	if len(refs) == 0 {
		return nil, true
	}
	existingSnapshots := map[string]*model.DeploymentRuntimeConfigSnapshot{}
	for _, ref := range model.DecodeDeploymentRuntimeConfigRefs(existingRaw) {
		if ref.Mode == model.RuntimeConfigRefModeSnapshot && ref.Snapshot != nil {
			snapshot := *ref.Snapshot
			existingSnapshots[ref.SetID] = &snapshot
		}
	}
	setIDs := make([]string, 0, len(refs))
	for _, ref := range refs {
		mode := model.RuntimeConfigRefMode(ref.Mode)
		if mode == model.RuntimeConfigRefModeSnapshot && existingSnapshots[ref.SetID] != nil {
			continue
		}
		setIDs = append(setIDs, ref.SetID)
	}
	var sets []model.ProjectRuntimeConfigSet
	if len(setIDs) > 0 {
		if err := h.db.Where("project_id = ? and id in ?", projectID, setIDs).Find(&sets).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return nil, false
		}
	}
	setsByID := make(map[string]model.ProjectRuntimeConfigSet, len(sets))
	for _, set := range sets {
		setsByID[set.ID] = set
	}
	if len(setsByID) != len(setIDs) {
		writeError(ctx, http.StatusBadRequest, "运行配置集不存在或不属于当前项目空间")
		return nil, false
	}
	now := time.Now()
	output := make([]model.DeploymentRuntimeConfigRef, 0, len(refs))
	for _, ref := range refs {
		mode := model.RuntimeConfigRefMode(ref.Mode)
		next := model.DeploymentRuntimeConfigRef{SetID: ref.SetID, Mode: mode}
		if mode == model.RuntimeConfigRefModeSnapshot {
			if snapshot := existingSnapshots[ref.SetID]; snapshot != nil {
				next.Snapshot = snapshot
			} else {
				snapshot := model.ProjectRuntimeConfigSetSnapshot(setsByID[ref.SetID], now)
				next.Snapshot = &snapshot
			}
		}
		output = append(output, next)
	}
	return output, true
}

func runtimeConfigRefInputs(input deploymentTargetInput) []deploymentRuntimeConfigRefInput {
	refs := make([]deploymentRuntimeConfigRefInput, 0)
	if len(input.RuntimeConfigRefs) > 0 {
		seen := map[string]bool{}
		for _, ref := range input.RuntimeConfigRefs {
			setID := strings.TrimSpace(ref.SetID)
			if setID == "" || seen[setID] {
				continue
			}
			seen[setID] = true
			refs = append(refs, deploymentRuntimeConfigRefInput{
				SetID: setID,
				Mode:  model.RuntimeConfigRefMode(ref.Mode),
			})
		}
		return refs
	}
	for _, setID := range normalizeStringList(input.RuntimeConfigSetIDs) {
		refs = append(refs, deploymentRuntimeConfigRefInput{SetID: setID, Mode: model.RuntimeConfigRefModeLive})
	}
	return refs
}

func normalizeDeploymentSourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image":
		return "image"
	default:
		return "repository"
	}
}

func normalizeBuildTimeoutSeconds(ctx *gin.Context, value int) (int, bool) {
	normalized := normalizeBuildTimeoutSecondsValue(value)
	if normalized < minBuildTimeoutSeconds || normalized > maxBuildTimeoutSeconds {
		writeError(ctx, http.StatusBadRequest, "构建超时时间必须在 1 分钟到 24 小时之间")
		return 0, false
	}
	return normalized, true
}

func normalizeBuildTimeoutSecondsValue(value int) int {
	if value <= 0 {
		return defaultBuildTimeoutSeconds
	}
	return value
}

func (h *Handlers) applyRegistryCredentialImageTemplate(ctx *gin.Context, user model.User, app model.Application, sourceType string, registryID string, repository string, tag string, target model.DeploymentTarget) (string, string, bool) {
	if sourceType != "repository" || strings.TrimSpace(registryID) == "" {
		return repository, tag, true
	}
	var project model.Project
	if err := h.db.First(&project, "id = ?", app.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "项目空间不存在")
		return repository, tag, false
	}
	var registry model.ArtifactRegistry
	if err := h.db.First(&registry, "id = ?", registryID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "目标镜像站不存在")
		return repository, tag, false
	}
	credential, ok := h.registryPushCredentialFor(user, registry)
	if !ok {
		return repository, tag, true
	}
	if strings.TrimSpace(repository) == "" || isDefaultImageRepository(registry, project, app, repository) {
		repository, _ = splitTargetImageRef(buildTargetImageRepositoryForCredential(registry, credential, project, app, target))
		repository = repositoryWithoutRegistryHost(registry, repository)
	}
	if strings.TrimSpace(tag) == "" || (strings.TrimSpace(tag) == "latest" && strings.TrimSpace(credential.TagTemplate) != "") {
		tag = buildStaticTargetImageTagForCredential(registry, credential, project, app, target)
	}
	return strings.Trim(strings.TrimSpace(repository), "/"), strings.TrimSpace(tag), true
}

func normalizeDeploymentServicePorts(ctx *gin.Context, input []model.DeploymentServicePort, fallbackPort int) ([]model.DeploymentServicePort, bool) {
	if len(input) == 0 {
		input = []model.DeploymentServicePort{{Name: "http", Port: fallbackInt(fallbackPort, 8080)}}
	}
	if len(input) > 16 {
		writeError(ctx, http.StatusBadRequest, "服务端口最多配置 16 个")
		return nil, false
	}
	seenNames := map[string]bool{}
	seenPorts := map[int]bool{}
	ports := make([]model.DeploymentServicePort, 0, len(input))
	for index, item := range input {
		port := item.Port
		if port <= 0 || port > 65535 {
			writeError(ctx, http.StatusBadRequest, "服务端口必须在 1 到 65535 之间")
			return nil, false
		}
		if seenPorts[port] {
			writeError(ctx, http.StatusBadRequest, "服务端口不能重复")
			return nil, false
		}
		name := normalizeDeploymentServicePortName(item.Name, port, index)
		if seenNames[name] {
			writeError(ctx, http.StatusBadRequest, "服务端口名称不能重复")
			return nil, false
		}
		seenPorts[port] = true
		seenNames[name] = true
		ports = append(ports, model.DeploymentServicePort{Name: name, Port: port, AppProtocol: normalizeAppProtocol(item.AppProtocol)})
	}
	return ports, true
}

func normalizeAppProtocol(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 253 {
		return value[:253]
	}
	return value
}

func normalizeDeploymentServicePortName(value string, port int, index int) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' {
			builder.WriteRune(char)
		} else if char == '_' || unicode.IsSpace(char) {
			builder.WriteRune('-')
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		if index == 0 {
			name = "http"
		} else {
			name = fmt.Sprintf("port-%d", port)
		}
	}
	if len(name) > 63 {
		name = strings.Trim(name[:63], "-")
	}
	return name
}

func normalizeBuildResourceQuantity(ctx *gin.Context, value string, fallbackValue string, label string) (string, bool) {
	normalized, err := normalizeBuildResourceQuantityValue(value, fallbackValue, label)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return "", false
	}
	return normalized, true
}

func normalizeBuildResourceQuantityValue(value string, fallbackValue string, label string) (string, error) {
	normalized := fallback(strings.TrimSpace(value), fallbackValue)
	quantity, err := resource.ParseQuantity(normalized)
	if err != nil || quantity.Sign() <= 0 {
		return "", fmt.Errorf("%s必须是有效的正数资源规格", label)
	}
	return normalized, nil
}

func normalizeOptionalResourceQuantity(ctx *gin.Context, value string, label string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	quantity, err := resource.ParseQuantity(normalized)
	if err != nil || quantity.Sign() <= 0 {
		writeError(ctx, http.StatusBadRequest, label+"必须是有效的正数资源规格")
		return "", false
	}
	return normalized, true
}

type deploymentKubernetesAdvancedInput struct {
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
	DataStorageClassName         string
	DataAccessMode               string
	DataVolumeMode               string
}

func normalizeDeploymentKubernetesAdvanced(ctx *gin.Context, input deploymentTargetInput) (deploymentKubernetesAdvancedInput, bool) {
	lifecycle, ok := normalizeLifecycleJSON(ctx, input.Lifecycle)
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	initContainers, ok := normalizeAuxContainersJSON(ctx, input.InitContainers, "初始化容器")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	sidecarContainers, ok := normalizeAuxContainersJSON(ctx, input.SidecarContainers, "Sidecar 容器")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	readinessProbe, ok := normalizeProbeJSON(ctx, input.ReadinessProbe, "就绪探针")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	livenessProbe, ok := normalizeProbeJSON(ctx, input.LivenessProbe, "存活探针")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	startupProbe, ok := normalizeProbeJSON(ctx, input.StartupProbe, "启动探针")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	runAsUser, ok := normalizeOptionalNonNegativeInteger(ctx, input.RunAsUser, "运行用户 UID")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	runAsGroup, ok := normalizeOptionalNonNegativeInteger(ctx, input.RunAsGroup, "运行用户组 GID")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	fsGroup, ok := normalizeOptionalNonNegativeInteger(ctx, input.FSGroup, "文件系统组 GID")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	tolerations, ok := normalizeTolerationsJSON(ctx, input.Tolerations)
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	affinity, ok := normalizeAffinityJSON(ctx, input.Affinity)
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	topologySpreadConstraints, ok := normalizeTopologySpreadConstraintsJSON(ctx, input.TopologySpreadConstraints)
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	nodeSelector, ok := normalizeMapJSONOrLines(ctx, input.NodeSelector, "节点选择器")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	serviceAnnotations, ok := normalizeMapJSONOrLines(ctx, input.ServiceAnnotations, "Service 注解")
	if !ok {
		return deploymentKubernetesAdvancedInput{}, false
	}
	return deploymentKubernetesAdvancedInput{
		ImagePullPolicy:              normalizeImagePullPolicyValue(input.ImagePullPolicy),
		ContainerCommand:             normalizeStringArrayText(input.ContainerCommand),
		ContainerArgs:                normalizeStringArrayText(input.ContainerArgs),
		Lifecycle:                    lifecycle,
		InitContainers:               initContainers,
		SidecarContainers:            sidecarContainers,
		ReadinessProbe:               readinessProbe,
		LivenessProbe:                livenessProbe,
		StartupProbe:                 startupProbe,
		RunAsUser:                    runAsUser,
		RunAsGroup:                   runAsGroup,
		FSGroup:                      fsGroup,
		FSGroupChangePolicy:          normalizeFSGroupChangePolicy(input.FSGroupChangePolicy),
		ReadOnlyRootFilesystem:       input.ReadOnlyRootFilesystem,
		AllowPrivilegeEscalation:     normalizeTriStateBool(input.AllowPrivilegeEscalation),
		CapabilityAdd:                normalizeStringArrayText(input.CapabilityAdd),
		CapabilityDrop:               normalizeStringArrayText(input.CapabilityDrop),
		NodeSelector:                 nodeSelector,
		Tolerations:                  tolerations,
		Affinity:                     affinity,
		TopologySpreadConstraints:    topologySpreadConstraints,
		PriorityClassName:            strings.TrimSpace(input.PriorityClassName),
		ServiceType:                  normalizeServiceType(input.ServiceType),
		ServiceAnnotations:           serviceAnnotations,
		ServiceExternalTrafficPolicy: normalizeServiceExternalTrafficPolicy(input.ServiceExternalTrafficPolicy),
		ServiceSessionAffinity:       normalizeServiceSessionAffinity(input.ServiceSessionAffinity),
		DataStorageClassName:         strings.TrimSpace(input.DataStorageClassName),
		DataAccessMode:               normalizePersistentVolumeAccessMode(input.DataAccessMode),
		DataVolumeMode:               normalizePersistentVolumeMode(input.DataVolumeMode),
	}, true
}

type deploymentAutoScalingInput struct {
	Enabled       bool
	MinReplicas   int
	MaxReplicas   int
	CPUPercent    int
	MemoryPercent int
	Behavior      string
}

func normalizeDeploymentAutoScaling(ctx *gin.Context, input deploymentTargetInput, replicas int) (deploymentAutoScalingInput, bool) {
	if !input.AutoScalingEnabled {
		return deploymentAutoScalingInput{MinReplicas: 1, MaxReplicas: fallbackInt(replicas, 1)}, true
	}
	minReplicas := input.AutoScalingMinReplicas
	if minReplicas <= 0 {
		minReplicas = fallbackInt(replicas, 1)
	}
	maxReplicas := input.AutoScalingMaxReplicas
	if maxReplicas <= 0 {
		maxReplicas = minReplicas
	}
	if maxReplicas < minReplicas {
		writeError(ctx, http.StatusBadRequest, "自动伸缩最大副本数不能小于最小副本数")
		return deploymentAutoScalingInput{}, false
	}
	cpuPercent := input.AutoScalingCPUPercent
	memoryPercent := input.AutoScalingMemoryPercent
	if cpuPercent < 0 || cpuPercent > 1000 || memoryPercent < 0 || memoryPercent > 1000 {
		writeError(ctx, http.StatusBadRequest, "自动伸缩目标利用率必须在 1 到 1000 之间")
		return deploymentAutoScalingInput{}, false
	}
	if cpuPercent == 0 && memoryPercent == 0 {
		writeError(ctx, http.StatusBadRequest, "启用自动伸缩后至少需要配置 CPU 或内存目标利用率")
		return deploymentAutoScalingInput{}, false
	}
	behavior, ok := normalizeHPABehaviorJSON(ctx, input.AutoScalingBehavior)
	if !ok {
		return deploymentAutoScalingInput{}, false
	}
	return deploymentAutoScalingInput{
		Enabled:       true,
		MinReplicas:   minReplicas,
		MaxReplicas:   maxReplicas,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		Behavior:      behavior,
	}, true
}

func normalizeWorkloadType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "statefulset", "stateful-set":
		return "StatefulSet"
	default:
		return "Deployment"
	}
}

func normalizeImagePullPolicyValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "always":
		return "Always"
	case "never":
		return "Never"
	case "ifnotpresent", "if-not-present":
		return "IfNotPresent"
	default:
		return ""
	}
}

func normalizeFSGroupChangePolicy(value string) string {
	switch strings.TrimSpace(value) {
	case string(corev1.FSGroupChangeOnRootMismatch):
		return string(corev1.FSGroupChangeOnRootMismatch)
	case string(corev1.FSGroupChangeAlways):
		return string(corev1.FSGroupChangeAlways)
	default:
		return ""
	}
}

func normalizeTriStateBool(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return "true"
	case "false":
		return "false"
	default:
		return ""
	}
}

func normalizeServiceType(value string) string {
	switch strings.TrimSpace(value) {
	case string(corev1.ServiceTypeNodePort):
		return string(corev1.ServiceTypeNodePort)
	case string(corev1.ServiceTypeLoadBalancer):
		return string(corev1.ServiceTypeLoadBalancer)
	case string(corev1.ServiceTypeClusterIP):
		return string(corev1.ServiceTypeClusterIP)
	default:
		return ""
	}
}

func normalizeServiceExternalTrafficPolicy(value string) string {
	switch strings.TrimSpace(value) {
	case string(corev1.ServiceExternalTrafficPolicyLocal):
		return string(corev1.ServiceExternalTrafficPolicyLocal)
	case string(corev1.ServiceExternalTrafficPolicyCluster):
		return string(corev1.ServiceExternalTrafficPolicyCluster)
	default:
		return ""
	}
}

func normalizeServiceSessionAffinity(value string) string {
	switch strings.TrimSpace(value) {
	case string(corev1.ServiceAffinityClientIP):
		return string(corev1.ServiceAffinityClientIP)
	case string(corev1.ServiceAffinityNone):
		return string(corev1.ServiceAffinityNone)
	default:
		return ""
	}
}

func normalizePersistentVolumeAccessMode(value string) string {
	switch strings.TrimSpace(value) {
	case string(corev1.ReadWriteMany):
		return string(corev1.ReadWriteMany)
	case string(corev1.ReadOnlyMany):
		return string(corev1.ReadOnlyMany)
	case string(corev1.ReadWriteOnce):
		return string(corev1.ReadWriteOnce)
	default:
		return ""
	}
}

func normalizePersistentVolumeMode(value string) string {
	switch strings.TrimSpace(value) {
	case string(corev1.PersistentVolumeBlock):
		return string(corev1.PersistentVolumeBlock)
	case string(corev1.PersistentVolumeFilesystem):
		return string(corev1.PersistentVolumeFilesystem)
	default:
		return ""
	}
}

func normalizeOptionalNonNegativeInteger(ctx *gin.Context, value string, label string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	parsed, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil || parsed < 0 {
		writeError(ctx, http.StatusBadRequest, label+"必须是非负整数")
		return "", false
	}
	return strconv.FormatInt(parsed, 10), true
}

func normalizeProbeJSON(ctx *gin.Context, value string, label string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var probe corev1.Probe
	if err := json.Unmarshal([]byte(normalized), &probe); err != nil {
		writeError(ctx, http.StatusBadRequest, label+"必须是合法的 Kubernetes Probe JSON")
		return "", false
	}
	return normalized, true
}

func normalizeLifecycleJSON(ctx *gin.Context, value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var lifecycle corev1.Lifecycle
	if err := json.Unmarshal([]byte(normalized), &lifecycle); err != nil {
		writeError(ctx, http.StatusBadRequest, "生命周期钩子必须是合法的 Kubernetes Lifecycle JSON")
		return "", false
	}
	return normalized, true
}

func normalizeHPABehaviorJSON(ctx *gin.Context, value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var behavior autoscalingv2.HorizontalPodAutoscalerBehavior
	if err := json.Unmarshal([]byte(normalized), &behavior); err != nil {
		writeError(ctx, http.StatusBadRequest, "HPA 行为必须是合法的 Kubernetes HorizontalPodAutoscalerBehavior JSON")
		return "", false
	}
	return normalized, true
}

func normalizeAuxContainersJSON(ctx *gin.Context, value string, label string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var containers []corev1.Container
	if err := json.Unmarshal([]byte(normalized), &containers); err != nil {
		writeError(ctx, http.StatusBadRequest, label+"必须是合法的 Kubernetes Container JSON 数组")
		return "", false
	}
	if len(containers) > 8 {
		writeError(ctx, http.StatusBadRequest, label+"最多配置 8 个")
		return "", false
	}
	for _, container := range containers {
		if strings.TrimSpace(container.Name) == "" || strings.TrimSpace(container.Image) == "" {
			writeError(ctx, http.StatusBadRequest, label+"必须填写 name 和 image")
			return "", false
		}
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			writeError(ctx, http.StatusBadRequest, label+"不允许启用 privileged")
			return "", false
		}
	}
	return normalized, true
}

func normalizeTolerationsJSON(ctx *gin.Context, value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var tolerations []corev1.Toleration
	if err := json.Unmarshal([]byte(normalized), &tolerations); err != nil {
		writeError(ctx, http.StatusBadRequest, "Tolerations 必须是合法的 Kubernetes Toleration JSON 数组")
		return "", false
	}
	return normalized, true
}

func normalizeAffinityJSON(ctx *gin.Context, value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var affinity corev1.Affinity
	if err := json.Unmarshal([]byte(normalized), &affinity); err != nil {
		writeError(ctx, http.StatusBadRequest, "Affinity 必须是合法的 Kubernetes Affinity JSON")
		return "", false
	}
	return normalized, true
}

func normalizeTopologySpreadConstraintsJSON(ctx *gin.Context, value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	var constraints []corev1.TopologySpreadConstraint
	if err := json.Unmarshal([]byte(normalized), &constraints); err != nil {
		writeError(ctx, http.StatusBadRequest, "拓扑分布约束必须是合法的 Kubernetes TopologySpreadConstraint JSON 数组")
		return "", false
	}
	return normalized, true
}

func normalizeMapJSONOrLines(ctx *gin.Context, value string, label string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", true
	}
	if strings.HasPrefix(normalized, "{") {
		var values map[string]string
		if err := json.Unmarshal([]byte(normalized), &values); err != nil {
			writeError(ctx, http.StatusBadRequest, label+"必须是合法的 JSON 对象或 KEY=VALUE 多行文本")
			return "", false
		}
		return normalized, true
	}
	for _, line := range strings.Split(normalized, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			writeError(ctx, http.StatusBadRequest, label+"必须是合法的 JSON 对象或 KEY=VALUE 多行文本")
			return "", false
		}
	}
	return normalized, true
}

func normalizeStringArrayText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeSecretRefsInput(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "{}" {
		return ""
	}
	return normalized
}
