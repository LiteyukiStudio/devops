package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) ListArtifactRegistries(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	projectID := strings.TrimSpace(ctx.Query("projectId"))
	if projectID != "" {
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return
		}
	}

	var registries []model.ArtifactRegistry
	query := h.db.Order("is_default desc, created_at desc")
	if user.Role != "platform_admin" {
		projectIDs := h.projectIDsForUser(user.ID)
		conditions := []string{"scope = 'global'", "(scope = 'user' and owner_ref = ?)"}
		args := []any{user.ID}
		if projectID != "" {
			conditions = append(conditions, "(scope = 'project' and owner_ref = ?)")
			args = append(args, projectID)
		} else if len(projectIDs) > 0 {
			conditions = append(conditions, "(scope = 'project' and owner_ref in ?)")
			args = append(args, projectIDs)
		}
		query = query.Where(strings.Join(conditions, " or "), args...)
	} else if projectID != "" {
		query = query.Where("scope <> 'project' or owner_ref = ?", projectID)
	}

	if err := query.Find(&registries).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, registryResponses(registries))
}

func (h *Handlers) CreateArtifactRegistry(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var input artifactRegistryInput
	if !bindJSON(ctx, &input) {
		return
	}

	registry, ok := h.registryFromInput(ctx, user, input, "")
	if !ok {
		return
	}
	registry.ID = id.New("reg")
	registry.CreatedBy = user.ID

	if err := h.saveRegistryWithDefault(registry); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, registryResponse(registry))
}

func (h *Handlers) UpdateArtifactRegistry(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var existing model.ArtifactRegistry
	if err := h.db.First(&existing, "id = ?", ctx.Param("registryId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "artifact registry not found")
		return
	}
	if !h.canManageRegistry(ctx, user, existing) {
		return
	}

	var input artifactRegistryInput
	if !bindJSON(ctx, &input) {
		return
	}

	next, ok := h.registryFromInput(ctx, user, input, existing.ID)
	if !ok {
		return
	}
	existing.Name = next.Name
	existing.Provider = next.Provider
	existing.Endpoint = next.Endpoint
	existing.Namespace = next.Namespace
	existing.Scope = next.Scope
	existing.OwnerRef = next.OwnerRef
	existing.CredentialRef = next.CredentialRef
	existing.IsDefault = next.IsDefault
	existing.Capabilities = next.Capabilities

	if err := h.saveRegistryWithDefault(existing); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, registryResponse(existing))
}

func (h *Handlers) DeleteArtifactRegistry(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var registry model.ArtifactRegistry
	if err := h.db.First(&registry, "id = ?", ctx.Param("registryId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "artifact registry not found")
		return
	}
	if !h.canManageRegistry(ctx, user, registry) {
		return
	}
	if err := h.db.Delete(&registry).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) GetDefaultArtifactRegistry(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(ctx.Param("projectId"))
	if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
		return
	}

	registry, ok := h.defaultRegistryFor(user.ID, projectID)
	if !ok {
		writeError(ctx, http.StatusNotFound, "default artifact registry not found")
		return
	}
	ctx.JSON(http.StatusOK, registryResponse(registry))
}

func (h *Handlers) ListRegistryCredentials(ctx *gin.Context) {
	user, registry, ok := h.registryForCurrentUser(ctx)
	if !ok {
		return
	}
	if !h.canManageRegistry(ctx, user, registry) {
		return
	}

	var credentials []model.RegistryCredential
	if err := h.db.Where("registry_id = ?", registry.ID).Order("created_at desc").Find(&credentials).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, credentialResponses(credentials))
}

func (h *Handlers) CreateRegistryCredential(ctx *gin.Context) {
	user, registry, ok := h.registryForCurrentUser(ctx)
	if !ok {
		return
	}
	if !h.canManageRegistry(ctx, user, registry) {
		return
	}

	var input registryCredentialInput
	if !bindJSON(ctx, &input) {
		return
	}

	credential := model.RegistryCredential{
		ID:          id.New("regc"),
		RegistryID:  registry.ID,
		Name:        fallback(strings.TrimSpace(input.Name), "default"),
		Username:    strings.TrimSpace(input.Username),
		PasswordRef: storedSecretRef(input.Password),
		TokenRef:    storedSecretRef(input.Token),
		Scope:       normalizeCredentialScope(input.Scope),
		CreatedBy:   user.ID,
	}
	if credential.PasswordRef == "" && credential.TokenRef == "" {
		writeError(ctx, http.StatusBadRequest, "请填写 Registry 密码或 Token")
		return
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&credential).Error; err != nil {
			return err
		}
		if registry.CredentialRef == "" {
			return tx.Model(&registry).Update("credential_ref", credential.ID).Error
		}
		return nil
	}); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, credentialResponse(credential))
}

func (h *Handlers) DeleteRegistryCredential(ctx *gin.Context) {
	user, registry, ok := h.registryForCurrentUser(ctx)
	if !ok {
		return
	}
	if !h.canManageRegistry(ctx, user, registry) {
		return
	}

	var credential model.RegistryCredential
	if err := h.db.First(&credential, "id = ? and registry_id = ?", ctx.Param("credentialId"), registry.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "registry credential not found")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&credential).Error; err != nil {
			return err
		}
		if registry.CredentialRef == credential.ID {
			return tx.Model(&registry).Update("credential_ref", "").Error
		}
		return nil
	}); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) TestArtifactRegistry(ctx *gin.Context) {
	_, registry, ok := h.registryForCurrentUser(ctx)
	if !ok {
		return
	}

	result := h.pingRegistry(ctx.Request.Context(), registry)
	status := http.StatusOK
	if !result.Success {
		status = http.StatusBadRequest
	}
	ctx.JSON(status, result)
}

func (h *Handlers) ListContainerImages(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	query := h.db.Order("created_at desc")
	if projectID := strings.TrimSpace(ctx.Query("projectId")); projectID != "" {
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return
		}
		query = query.Where("project_id = ?", projectID)
	} else if user.Role != "platform_admin" {
		query = query.Where("created_by = ? or project_id in ?", user.ID, h.projectIDsForUser(user.ID))
	}

	var images []model.ContainerImage
	if err := query.Find(&images).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, images)
}

func (h *Handlers) CreateContainerImage(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var input containerImageInput
	if !bindJSON(ctx, &input) {
		return
	}
	registry, ok := h.findAccessibleRegistry(ctx, user, input.RegistryID)
	if !ok {
		return
	}
	if input.ProjectID != "" {
		if _, ok := h.findProjectForCurrentUserByID(ctx, input.ProjectID); !ok {
			return
		}
	}

	image := model.ContainerImage{
		ID:            id.New("img"),
		ProjectID:     strings.TrimSpace(input.ProjectID),
		ApplicationID: strings.TrimSpace(input.ApplicationID),
		RegistryID:    registry.ID,
		Repository:    strings.Trim(strings.TrimSpace(input.Repository), "/"),
		Tag:           fallback(strings.TrimSpace(input.Tag), "latest"),
		Digest:        strings.TrimSpace(input.Digest),
		SourceCommit:  strings.TrimSpace(input.SourceCommit),
		BuildRunID:    strings.TrimSpace(input.BuildRunID),
		SourceType:    normalizeImageSourceType(input.SourceType),
		ScanStatus:    normalizeScanStatus(input.ScanStatus),
		CreatedBy:     user.ID,
	}
	if image.Repository == "" {
		writeError(ctx, http.StatusBadRequest, "请输入镜像仓库路径")
		return
	}
	image.ImageRef = imageReference(registry, image.Repository, image.Tag, image.Digest)

	if err := h.db.Create(&image).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, image)
}

func (h *Handlers) registryFromInput(ctx *gin.Context, user model.User, input artifactRegistryInput, registryID string) (model.ArtifactRegistry, bool) {
	scope := normalizeRegistryScope(input.Scope)
	ownerRef := strings.TrimSpace(input.OwnerRef)
	switch scope {
	case "global":
		if user.Role != "platform_admin" {
			writeError(ctx, http.StatusForbidden, "只有平台管理员可以维护全局镜像站")
			return model.ArtifactRegistry{}, false
		}
		ownerRef = ""
	case "project":
		if ownerRef == "" {
			writeError(ctx, http.StatusBadRequest, "项目镜像站需要选择项目")
			return model.ArtifactRegistry{}, false
		}
		if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, ownerRef, "owner", "admin", "developer"); !ok {
			return model.ArtifactRegistry{}, false
		}
	case "user":
		ownerRef = user.ID
	}

	endpoint := strings.TrimRight(strings.TrimSpace(input.Endpoint), "/")
	if _, err := parseRegistryEndpoint(endpoint); err != nil {
		writeError(ctx, http.StatusBadRequest, "请输入有效镜像站地址")
		return model.ArtifactRegistry{}, false
	}

	registry := model.ArtifactRegistry{
		ID:            registryID,
		Name:          strings.TrimSpace(input.Name),
		Provider:      normalizeRegistryProvider(input.Provider),
		Endpoint:      endpoint,
		Namespace:     strings.Trim(strings.TrimSpace(input.Namespace), "/"),
		Scope:         scope,
		OwnerRef:      ownerRef,
		CredentialRef: strings.TrimSpace(input.CredentialRef),
		IsDefault:     input.IsDefault,
		Capabilities:  strings.Join(normalizeList(input.Capabilities, false), ","),
		CreatedBy:     user.ID,
	}
	if registry.Name == "" {
		writeError(ctx, http.StatusBadRequest, "请输入镜像站名称")
		return model.ArtifactRegistry{}, false
	}
	return registry, true
}

func (h *Handlers) saveRegistryWithDefault(registry model.ArtifactRegistry) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if registry.IsDefault {
			if err := tx.Model(&model.ArtifactRegistry{}).
				Where("scope = ? and owner_ref = ? and id <> ?", registry.Scope, registry.OwnerRef, registry.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&registry).Error
	})
}

func (h *Handlers) registryForCurrentUser(ctx *gin.Context) (model.User, model.ArtifactRegistry, bool) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return model.User{}, model.ArtifactRegistry{}, false
	}
	registry, ok := h.findAccessibleRegistry(ctx, user, ctx.Param("registryId"))
	return user, registry, ok
}

func (h *Handlers) findAccessibleRegistry(ctx *gin.Context, user model.User, registryID string) (model.ArtifactRegistry, bool) {
	var registry model.ArtifactRegistry
	if err := h.db.First(&registry, "id = ?", strings.TrimSpace(registryID)).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "artifact registry not found")
		return registry, false
	}
	if !h.canUseRegistry(ctx, user, registry) {
		return registry, false
	}
	return registry, true
}

func (h *Handlers) canUseRegistry(ctx *gin.Context, user model.User, registry model.ArtifactRegistry) bool {
	switch registry.Scope {
	case "global":
		return true
	case "user":
		if registry.OwnerRef == user.ID || user.Role == "platform_admin" {
			return true
		}
	case "project":
		if user.Role == "platform_admin" {
			return true
		}
		if h.userHasProject(ctx, user.ID, registry.OwnerRef) {
			return true
		}
	}
	writeError(ctx, http.StatusForbidden, "无权访问该镜像站")
	return false
}

func (h *Handlers) canManageRegistry(ctx *gin.Context, user model.User, registry model.ArtifactRegistry) bool {
	switch registry.Scope {
	case "global":
		if user.Role == "platform_admin" {
			return true
		}
	case "user":
		if registry.OwnerRef == user.ID || user.Role == "platform_admin" {
			return true
		}
	case "project":
		if user.Role == "platform_admin" {
			return true
		}
		if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, registry.OwnerRef, "owner", "admin", "developer"); ok {
			return true
		}
		return false
	}
	writeError(ctx, http.StatusForbidden, "无权维护该镜像站")
	return false
}

func (h *Handlers) defaultRegistryFor(userID, projectID string) (model.ArtifactRegistry, bool) {
	candidates := []struct {
		scope string
		owner string
	}{
		{scope: "project", owner: projectID},
		{scope: "user", owner: userID},
		{scope: "global", owner: ""},
	}
	for _, candidate := range candidates {
		var registry model.ArtifactRegistry
		err := h.db.First(&registry, "scope = ? and owner_ref = ? and is_default = ?", candidate.scope, candidate.owner, true).Error
		if err == nil {
			return registry, true
		}
	}
	return model.ArtifactRegistry{}, false
}

func (h *Handlers) projectIDsForUser(userID string) []string {
	var projectIDs []string
	_ = h.db.Model(&model.ProjectMember{}).Where("user_id = ?", userID).Pluck("project_id", &projectIDs).Error
	return projectIDs
}

func (h *Handlers) userHasProject(ctx *gin.Context, userID, projectID string) bool {
	var count int64
	if err := h.db.Model(&model.ProjectMember{}).Where("user_id = ? and project_id = ?", userID, projectID).Count(&count).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return false
	}
	return count > 0
}

func (h *Handlers) findProjectForCurrentUserByID(ctx *gin.Context, projectID string) (model.Project, bool) {
	original := ctx.Param("projectId")
	ctx.Params = append(ctx.Params, gin.Param{Key: "projectId", Value: projectID})
	project, ok := h.findProjectForCurrentUser(ctx)
	ctx.Params = replaceParam(ctx.Params, "projectId", original)
	return project, ok
}

func (h *Handlers) findProjectForCurrentUserWithRolesByID(ctx *gin.Context, projectID string, roles ...string) (model.Project, bool) {
	original := ctx.Param("projectId")
	ctx.Params = append(ctx.Params, gin.Param{Key: "projectId", Value: projectID})
	project, ok := h.findProjectForCurrentUserWithRoles(ctx, roles...)
	ctx.Params = replaceParam(ctx.Params, "projectId", original)
	return project, ok
}

func replaceParam(params gin.Params, key, value string) gin.Params {
	result := gin.Params{}
	for _, param := range params {
		if param.Key == key {
			continue
		}
		result = append(result, param)
	}
	if value != "" {
		result = append(result, gin.Param{Key: key, Value: value})
	}
	return result
}

func (h *Handlers) pingRegistry(parent context.Context, registry model.ArtifactRegistry) registryTestResult {
	endpoint, err := registryPingURL(registry.Endpoint)
	if err != nil {
		return registryTestResult{Success: false, Message: err.Error()}
	}
	if err := validateRegistryPingTarget(endpoint, h.mode); err != nil {
		return registryTestResult{Success: false, Message: err.Error(), Endpoint: endpoint.String()}
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return registryTestResult{Success: false, Message: err.Error(), Endpoint: endpoint.String()}
	}
	if credential, ok := h.registryCredentialFor(registry); ok {
		secret := resolveStoredSecretRef(credential.TokenRef)
		if secret == "" {
			secret = resolveStoredSecretRef(credential.PasswordRef)
		}
		if credential.Username != "" && secret != "" {
			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(credential.Username+":"+secret)))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return registryTestResult{Success: false, Message: err.Error(), Endpoint: endpoint.String()}
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 400
	if resp.StatusCode == http.StatusUnauthorized {
		return registryTestResult{Success: false, StatusCode: resp.StatusCode, Message: "镜像站可访问，但凭据未通过认证", Endpoint: endpoint.String()}
	}
	return registryTestResult{Success: success, StatusCode: resp.StatusCode, Message: registryTestMessage(success, resp.StatusCode), Endpoint: endpoint.String()}
}

func (h *Handlers) registryCredentialFor(registry model.ArtifactRegistry) (model.RegistryCredential, bool) {
	var credential model.RegistryCredential
	if registry.CredentialRef != "" {
		if err := h.db.First(&credential, "id = ? and registry_id = ?", registry.CredentialRef, registry.ID).Error; err == nil {
			return credential, true
		}
	}
	if err := h.db.Where("registry_id = ?", registry.ID).Order("created_at desc").First(&credential).Error; err == nil {
		return credential, true
	}
	return credential, false
}

func registryResponses(registries []model.ArtifactRegistry) []artifactRegistryOutput {
	result := make([]artifactRegistryOutput, 0, len(registries))
	for _, registry := range registries {
		result = append(result, registryResponse(registry))
	}
	return result
}

func registryResponse(registry model.ArtifactRegistry) artifactRegistryOutput {
	return artifactRegistryOutput{
		ID:            registry.ID,
		Name:          registry.Name,
		Provider:      registry.Provider,
		Endpoint:      registry.Endpoint,
		Namespace:     registry.Namespace,
		Scope:         registry.Scope,
		OwnerRef:      registry.OwnerRef,
		CredentialRef: registry.CredentialRef,
		IsDefault:     registry.IsDefault,
		Capabilities:  jsonList(splitCSV(registry.Capabilities)),
		CreatedBy:     registry.CreatedBy,
		CreatedAt:     registry.CreatedAt,
	}
}

func credentialResponses(credentials []model.RegistryCredential) []registryCredentialOutput {
	result := make([]registryCredentialOutput, 0, len(credentials))
	for _, credential := range credentials {
		result = append(result, credentialResponse(credential))
	}
	return result
}

func credentialResponse(credential model.RegistryCredential) registryCredentialOutput {
	return registryCredentialOutput{
		ID:          credential.ID,
		RegistryID:  credential.RegistryID,
		Name:        credential.Name,
		Username:    credential.Username,
		Scope:       credential.Scope,
		PasswordSet: secretRefHasValue(credential.PasswordRef),
		TokenSet:    secretRefHasValue(credential.TokenRef),
		CreatedAt:   credential.CreatedAt,
	}
}

func normalizeRegistryProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dockerhub", "gitea-registry":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "harbor"
	}
}

func normalizeRegistryScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "project", "user":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "global"
	}
}

func normalizeCredentialScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "push", "pull":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "push-pull"
	}
}

func normalizeImageSourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "build":
		return "build"
	default:
		return "manual-image"
	}
}

func normalizeScanStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending", "scanning", "passed", "failed":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}

func parseRegistryEndpoint(endpoint string) (*url.URL, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("invalid registry endpoint")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("registry endpoint must use http or https")
	}
	return parsed, nil
}

func registryPingURL(endpoint string) (*url.URL, error) {
	parsed, err := parseRegistryEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v2/"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

func validateRegistryPingTarget(endpoint *url.URL, mode string) error {
	host := endpoint.Hostname()
	port := endpoint.Port()
	if port == "" {
		if endpoint.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if mode == "development" && (host == "localhost" || host == "127.0.0.1" || host == "::1") {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve registry host: %w", err)
	}
	for _, ip := range ips {
		if isBlockedRegistryIP(ip) && port != "443" {
			return errors.New("private registry endpoints are only allowed on tcp/443")
		}
		if isMetadataIP(ip) {
			return errors.New("metadata endpoints are not allowed")
		}
	}
	return nil
}

func isBlockedRegistryIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func isMetadataIP(ip net.IP) bool {
	return ip.Equal(net.ParseIP("169.254.169.254")) || ip.Equal(net.ParseIP("100.100.100.200"))
}

func registryTestMessage(success bool, statusCode int) string {
	if success {
		return "镜像站连接成功"
	}
	return fmt.Sprintf("镜像站返回 HTTP %d", statusCode)
}

func imageReference(registry model.ArtifactRegistry, repository, tag, digest string) string {
	host := strings.TrimPrefix(strings.TrimPrefix(strings.TrimRight(registry.Endpoint, "/"), "https://"), "http://")
	parts := []string{host}
	if registry.Namespace != "" {
		parts = append(parts, registry.Namespace)
	}
	parts = append(parts, strings.Trim(repository, "/"))
	base := strings.Join(parts, "/")
	if digest != "" {
		return base + "@" + digest
	}
	return base + ":" + fallback(tag, "latest")
}

type artifactRegistryInput struct {
	Name          string   `json:"name" binding:"required"`
	Provider      string   `json:"provider"`
	Endpoint      string   `json:"endpoint" binding:"required"`
	Namespace     string   `json:"namespace"`
	Scope         string   `json:"scope"`
	OwnerRef      string   `json:"ownerRef"`
	CredentialRef string   `json:"credentialRef"`
	IsDefault     bool     `json:"isDefault"`
	Capabilities  []string `json:"capabilities"`
}

type artifactRegistryOutput struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Provider      string    `json:"provider"`
	Endpoint      string    `json:"endpoint"`
	Namespace     string    `json:"namespace"`
	Scope         string    `json:"scope"`
	OwnerRef      string    `json:"ownerRef"`
	CredentialRef string    `json:"credentialRef"`
	IsDefault     bool      `json:"isDefault"`
	Capabilities  []string  `json:"capabilities"`
	CreatedBy     string    `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
}

type registryCredentialInput struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Scope    string `json:"scope"`
}

type registryCredentialOutput struct {
	ID          string    `json:"id"`
	RegistryID  string    `json:"registryId"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	Scope       string    `json:"scope"`
	PasswordSet bool      `json:"passwordSet"`
	TokenSet    bool      `json:"tokenSet"`
	CreatedAt   time.Time `json:"createdAt"`
}

type containerImageInput struct {
	ProjectID     string `json:"projectId"`
	ApplicationID string `json:"applicationId"`
	RegistryID    string `json:"registryId" binding:"required"`
	Repository    string `json:"repository" binding:"required"`
	Tag           string `json:"tag"`
	Digest        string `json:"digest"`
	SourceCommit  string `json:"sourceCommit"`
	BuildRunID    string `json:"buildRunId"`
	SourceType    string `json:"sourceType"`
	ScanStatus    string `json:"scanStatus"`
}

type registryTestResult struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Endpoint   string `json:"endpoint"`
}
