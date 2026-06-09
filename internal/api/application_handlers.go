package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) ListApplications(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}

	var applications []model.Application
	query := h.db.Where("project_id = ?", ctx.Param("projectId"))
	query = applySearch(ctx, query, "name", "slug")
	if err := query.Order("created_at desc").Find(&applications).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, applications)
}

func (h *Handlers) CreateApplication(ctx *gin.Context) {
	_, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}

	var input applicationInput
	if !bindJSON(ctx, &input) {
		return
	}
	input.Slug = strings.TrimSpace(input.Slug)
	if len(input.Slug) > applicationSlugMaxLength {
		writeError(ctx, http.StatusBadRequest, fmt.Sprintf("应用标识最多 %d 个字符", applicationSlugMaxLength))
		return
	}
	if !h.ensureApplicationSlugAvailable(ctx, ctx.Param("projectId"), input.Slug, "") {
		return
	}
	app := model.Application{
		ID:          id.New("app"),
		ProjectID:   ctx.Param("projectId"),
		Slug:        input.Slug,
		Name:        input.Name,
		Icon:        normalizeApplicationIcon(input.Icon),
		ServicePort: fallbackInt(input.ServicePort, 8080),
	}

	if err := h.db.Create(&app).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, app)
}

func (h *Handlers) GetApplication(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}

	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	ctx.JSON(http.StatusOK, app)
}

func (h *Handlers) UpdateApplication(ctx *gin.Context) {
	_, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}

	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}

	var input applicationInput
	if !bindJSON(ctx, &input) {
		return
	}
	input.Slug = strings.TrimSpace(input.Slug)
	if len(input.Slug) > applicationSlugMaxLength {
		writeError(ctx, http.StatusBadRequest, fmt.Sprintf("应用标识最多 %d 个字符", applicationSlugMaxLength))
		return
	}
	if !h.ensureApplicationSlugAvailable(ctx, ctx.Param("projectId"), input.Slug, app.ID) {
		return
	}
	app.Slug = input.Slug
	app.Name = input.Name
	app.Icon = normalizeApplicationIcon(input.Icon)
	app.ServicePort = fallbackInt(input.ServicePort, 8080)

	if err := h.db.Save(&app).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, app)
}

func (h *Handlers) ListApplicationModules(ctx *gin.Context) {
	_, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer", "viewer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var configs []model.ApplicationModule
	if err := h.db.Where("project_id = ? and application_id = ?", app.ProjectID, app.ID).Order("created_at asc").Find(&configs).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.attachApplicationModuleHookBindings(configs); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, configs)
}

func (h *Handlers) CreateApplicationModule(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var input applicationModuleInput
	if !bindJSON(ctx, &input) {
		return
	}
	input.Enabled = true
	config, ok := h.applicationModuleFromInput(ctx, user, app, input, id.New("mod"))
	if !ok {
		return
	}
	if err := h.saveApplicationModule(config, input.BuildHookBindings); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	config, err := h.applicationModuleWithHookBindings(config)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, config)
}

func (h *Handlers) UpdateApplicationModule(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var existing model.ApplicationModule
	if err := h.db.First(&existing, "id = ? and project_id = ? and application_id = ?", ctx.Param("moduleId"), app.ProjectID, app.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "module not found")
		return
	}
	var input applicationModuleInput
	if !bindJSON(ctx, &input) {
		return
	}
	config, ok := h.applicationModuleFromInput(ctx, user, app, input, existing.ID)
	if !ok {
		return
	}
	config.CreatedBy = existing.CreatedBy
	config.CreatedAt = existing.CreatedAt
	if err := h.saveApplicationModule(config, input.BuildHookBindings); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	config, err := h.applicationModuleWithHookBindings(config)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, config)
}

func (h *Handlers) DeleteApplicationModule(ctx *gin.Context) {
	_, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var config model.ApplicationModule
	if err := h.db.First(&config, "id = ? and project_id = ? and application_id = ?", ctx.Param("moduleId"), app.ProjectID, app.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "module not found")
		return
	}
	if h.applicationModuleInUse(config.ID) {
		writeError(ctx, http.StatusBadRequest, "模块配置已被构建记录或部署配置引用，不能删除")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("module_id = ?", config.ID).Delete(&model.ApplicationModuleHookBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&config).Error
	}); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) DeleteApplication(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}

	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	if err := h.db.Delete(&app).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(user.ID, "application.delete", app.ID, true, app.Name)
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) findApplication(ctx *gin.Context) (model.Application, bool) {
	var app model.Application
	err := h.db.First(
		&app,
		"id = ? and project_id = ?",
		ctx.Param("applicationId"),
		ctx.Param("projectId"),
	).Error
	if err != nil {
		writeError(ctx, http.StatusNotFound, "application not found")
		return app, false
	}
	return app, true
}

func (h *Handlers) ensureApplicationSlugAvailable(ctx *gin.Context, projectID string, slug string, excludeApplicationID string) bool {
	if slug == "" {
		writeError(ctx, http.StatusBadRequest, "应用标识不能为空")
		return false
	}
	query := h.db.Model(&model.Application{}).Where("project_id = ? and slug = ?", projectID, slug)
	if strings.TrimSpace(excludeApplicationID) != "" {
		query = query.Where("id <> ?", excludeApplicationID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return false
	}
	if count > 0 {
		writeError(ctx, http.StatusBadRequest, "该项目空间内应用标识已存在")
		return false
	}
	return true
}

func (h *Handlers) applicationModuleFromInput(ctx *gin.Context, user model.User, app model.Application, input applicationModuleInput, configID string) (model.ApplicationModule, bool) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		writeError(ctx, http.StatusBadRequest, "模块配置名称不能为空")
		return model.ApplicationModule{}, false
	}
	slug := dnsSafeSegment(input.Slug)
	if slug == "" {
		slug = dnsSafeSegment(name)
	}
	if slug == "" {
		writeError(ctx, http.StatusBadRequest, "模块配置标识不能为空")
		return model.ApplicationModule{}, false
	}
	if len(slug) > deploymentSlugMaxLength {
		writeError(ctx, http.StatusBadRequest, fmt.Sprintf("模块配置标识最多 %d 个字符", deploymentSlugMaxLength))
		return model.ApplicationModule{}, false
	}
	repositoryBindingID := strings.TrimSpace(input.RepositoryBindingID)
	if repositoryBindingID != "" {
		var binding model.RepositoryBinding
		if err := h.db.First(&binding, "id = ? and project_id = ? and application_id = ?", repositoryBindingID, app.ProjectID, app.ID).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "代码仓库绑定不存在")
			return model.ApplicationModule{}, false
		}
	}
	if strings.TrimSpace(input.BuildProviderID) != "" {
		var provider model.BuildProvider
		if err := h.db.First(&provider, "id = ? and enabled = ?", input.BuildProviderID, true).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "构建提供方不可用")
			return model.ApplicationModule{}, false
		}
	}
	targetRepository, targetTag := splitTargetImageRef(input.TargetImageRef)
	if targetRepository == "" {
		targetRepository = strings.Trim(strings.TrimSpace(input.TargetRepository), "/")
		targetTag = strings.TrimSpace(input.TargetTag)
	}
	buildHooksEnabled := true
	if input.BuildHooksEnabled != nil {
		buildHooksEnabled = *input.BuildHooksEnabled
	}
	return model.ApplicationModule{
		ID:                  configID,
		ProjectID:           app.ProjectID,
		ApplicationID:       app.ID,
		Name:                name,
		Slug:                slug,
		RepositoryBindingID: repositoryBindingID,
		BuildProviderID:     strings.TrimSpace(input.BuildProviderID),
		DockerfilePath:      fallback(strings.TrimSpace(input.DockerfilePath), "Dockerfile"),
		BuildContext:        fallback(strings.TrimSpace(input.BuildContext), "."),
		BuildDirectory:      strings.TrimSpace(input.BuildDirectory),
		TargetRegistryID:    strings.TrimSpace(input.TargetRegistryID),
		TargetRepository:    targetRepository,
		TargetTag:           fallback(targetTag, "latest"),
		BuildLabels:         strings.Join(normalizeBuildSelectorList(strings.Split(input.BuildLabels, ",")), ","),
		BuildVariableSetIDs: encodeBuildVariableSetIDs(input.BuildVariableSetIDs),
		BuildHooksEnabled:   buildHooksEnabled,
		BranchPattern:       strings.TrimSpace(input.BranchPattern),
		TagPattern:          strings.TrimSpace(input.TagPattern),
		ConcurrencyPolicy:   normalizeBuildConcurrencyPolicy(input.ConcurrencyPolicy),
		Enabled:             input.Enabled,
		CreatedBy:           user.ID,
	}, true
}

func (h *Handlers) saveApplicationModule(config model.ApplicationModule, hookInputs []applicationModuleHookBindingInput) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.ApplicationModule{}).
			Where("project_id = ? and application_id = ? and slug = ? and id <> ?", config.ProjectID, config.ApplicationID, config.Slug, config.ID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errModuleSlugExists
		}
		if err := tx.Save(&config).Error; err != nil {
			return err
		}
		return h.replaceApplicationModuleHookBindings(tx, config, hookInputs)
	})
}

func (h *Handlers) attachApplicationModuleHookBindings(configs []model.ApplicationModule) error {
	if len(configs) == 0 {
		return nil
	}
	moduleIDs := make([]string, 0, len(configs))
	moduleIndex := make(map[string]int, len(configs))
	for index := range configs {
		moduleIDs = append(moduleIDs, configs[index].ID)
		moduleIndex[configs[index].ID] = index
	}
	var bindings []model.ApplicationModuleHookBinding
	if err := h.db.Where("module_id in ?", moduleIDs).Order("run_order asc, created_at asc").Find(&bindings).Error; err != nil {
		return err
	}
	for _, binding := range bindings {
		index, ok := moduleIndex[binding.ModuleID]
		if !ok {
			continue
		}
		configs[index].BuildHookBindings = append(configs[index].BuildHookBindings, binding)
	}
	return nil
}

func (h *Handlers) applicationModuleWithHookBindings(config model.ApplicationModule) (model.ApplicationModule, error) {
	configs := []model.ApplicationModule{config}
	if err := h.attachApplicationModuleHookBindings(configs); err != nil {
		return config, err
	}
	return configs[0], nil
}

func (h *Handlers) replaceApplicationModuleHookBindings(tx *gorm.DB, config model.ApplicationModule, inputs []applicationModuleHookBindingInput) error {
	if err := tx.Where("module_id = ?", config.ID).Delete(&model.ApplicationModuleHookBinding{}).Error; err != nil {
		return err
	}
	if len(inputs) == 0 {
		return nil
	}
	hookIDs := make([]string, 0, len(inputs))
	seen := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		hookID := strings.TrimSpace(input.HookConfigID)
		if hookID == "" || seen[hookID] {
			continue
		}
		seen[hookID] = true
		hookIDs = append(hookIDs, hookID)
	}
	if len(hookIDs) == 0 {
		return nil
	}
	var hooks []model.ProjectHookConfig
	if err := tx.Where("project_id = ? and id in ? and phase in ?", config.ProjectID, hookIDs, []string{hookPhasePreBuild, hookPhasePostBuild}).Find(&hooks).Error; err != nil {
		return err
	}
	validHookIDs := make(map[string]bool, len(hooks))
	for _, hook := range hooks {
		validHookIDs[hook.ID] = true
	}
	if len(validHookIDs) != len(hookIDs) {
		return errors.New("构建钩子不存在或不是构建阶段")
	}
	bindings := make([]model.ApplicationModuleHookBinding, 0, len(hookIDs))
	for index, hookID := range hookIDs {
		bindings = append(bindings, model.ApplicationModuleHookBinding{
			ID:            id.New("mhb"),
			ProjectID:     config.ProjectID,
			ApplicationID: config.ApplicationID,
			ModuleID:      config.ID,
			HookConfigID:  hookID,
			RunOrder:      index + 1,
		})
	}
	return tx.Create(&bindings).Error
}

func (h *Handlers) applicationModuleInUse(configID string) bool {
	var count int64
	if err := h.db.Model(&model.BuildRun{}).Where("module_id = ?", configID).Count(&count).Error; err != nil || count > 0 {
		return true
	}
	if err := h.db.Model(&model.DeploymentTarget{}).Where("module_id = ?", configID).Count(&count).Error; err != nil || count > 0 {
		return true
	}
	return false
}

var errModuleSlugExists = errors.New("模块配置标识已存在")

type applicationInput struct {
	Slug        string `json:"slug" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Icon        string `json:"icon"`
	ServicePort int    `json:"servicePort"`
}

type applicationModuleInput struct {
	Name                string                              `json:"name" binding:"required"`
	Slug                string                              `json:"slug"`
	RepositoryBindingID string                              `json:"repositoryBindingId"`
	BuildProviderID     string                              `json:"buildProviderId"`
	DockerfilePath      string                              `json:"dockerfilePath"`
	BuildContext        string                              `json:"buildContext"`
	BuildDirectory      string                              `json:"buildDirectory"`
	TargetRegistryID    string                              `json:"targetRegistryId"`
	TargetImageRef      string                              `json:"targetImageRef"`
	TargetRepository    string                              `json:"targetRepository"`
	TargetTag           string                              `json:"targetTag"`
	BuildLabels         string                              `json:"buildLabels"`
	BuildVariableSetIDs []string                            `json:"buildVariableSetIds"`
	BuildHooksEnabled   *bool                               `json:"buildHooksEnabled"`
	BuildHookBindings   []applicationModuleHookBindingInput `json:"buildHookBindings"`
	BranchPattern       string                              `json:"branchPattern"`
	TagPattern          string                              `json:"tagPattern"`
	ConcurrencyPolicy   string                              `json:"concurrencyPolicy"`
	Enabled             bool                                `json:"enabled"`
}

type applicationModuleHookBindingInput struct {
	HookConfigID string `json:"hookConfigId"`
	RunOrder     int    `json:"runOrder"`
}

func normalizeBuildConcurrencyPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "parallel":
		return "parallel"
	default:
		return "queue"
	}
}

func normalizeApplicationIcon(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "box"
	}
	for _, icon := range applicationIconNames {
		if normalized == icon {
			return normalized
		}
	}
	return "box"
}

var applicationIconNames = []string{
	"box",
	"app-window",
	"layout-dashboard",
	"server",
	"database",
	"cpu",
	"cloud",
	"globe",
	"network",
	"shield",
	"lock-keyhole",
	"key-round",
	"shopping-cart",
	"credit-card",
	"chart-line",
	"bar-chart-3",
	"message-square",
	"mail",
	"bell",
	"calendar",
	"file-text",
	"folder-kanban",
	"git-branch",
	"terminal",
	"workflow",
	"package",
	"container",
	"rocket",
	"zap",
	"bot",
	"users",
	"settings",
}
