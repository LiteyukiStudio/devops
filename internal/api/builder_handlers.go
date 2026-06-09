package api

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/config"
	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/variables"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *Handlers) BuilderHeartbeat(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	var input builderHeartbeatInput
	if !bindJSON(ctx, &input) {
		return
	}
	agentID := strings.TrimSpace(input.AgentID)
	if agentID == "" {
		writeError(ctx, http.StatusBadRequest, "builder agent id is required")
		return
	}
	now := time.Now()
	agent := model.BuilderAgent{
		ID:                 agentID,
		Name:               fallback(strings.TrimSpace(input.Name), agentID),
		Labels:             strings.Join(normalizeStringList(input.Labels), ","),
		Scopes:             strings.Join(normalizeStringList(input.Scopes), ","),
		Executor:           fallback(strings.TrimSpace(input.Executor), "docker"),
		Status:             "online",
		MaxConcurrency:     fallbackInt(input.MaxConcurrency, 1),
		CurrentConcurrency: input.CurrentConcurrency,
		LastHeartbeatAt:    &now,
	}
	if err := h.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "labels", "scopes", "executor", "status", "max_concurrency", "current_concurrency", "last_heartbeat_at", "updated_at",
		}),
	}).Create(&agent).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, agent)
}

func (h *Handlers) ClaimBuilderTask(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	var input builderClaimInput
	if !bindJSON(ctx, &input) {
		return
	}
	heartbeat := heartbeatFromClaimInput(input)
	agentID := strings.TrimSpace(heartbeat.AgentID)
	if agentID == "" {
		writeError(ctx, http.StatusBadRequest, "builder agent id is required")
		return
	}
	if err := h.recordBuilderHeartbeat(heartbeat); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	task, ok, err := h.claimBuildTask(heartbeat)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		ctx.Status(http.StatusNoContent)
		return
	}
	ctx.JSON(http.StatusOK, task)
}

func (h *Handlers) AppendBuilderTaskLogs(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	job, ok := h.builderRunningJobForAgentLease(ctx)
	if !ok {
		return
	}
	var input builderLogsInput
	if !bindJSON(ctx, &input) {
		return
	}
	if err := h.appendBuildLog(job, input.Content); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) ProgressBuilderTask(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	job, ok := h.builderRunningJobForAgentLease(ctx)
	if !ok {
		return
	}
	var input builderProgressInput
	if !bindJSON(ctx, &input) {
		return
	}
	progress := normalizeBuilderProgressKey(input.Key)
	if progress == "" {
		ctx.Status(http.StatusNoContent)
		return
	}
	if err := h.db.Model(&model.BuildJob{}).
		Where("id = ? and builder_id = ? and lease_token = ? and status = ?", job.ID, job.BuilderID, job.LeaseToken, "running").
		Update("message", progress).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) RenewBuilderTask(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	job, ok := h.builderRunningJobForAgentLease(ctx)
	if !ok {
		return
	}
	var input builderRenewInput
	if !bindJSON(ctx, &input) {
		return
	}
	now := time.Now()
	leaseUntil := now.Add(time.Duration(config.Load().BuilderTaskLeaseSeconds) * time.Second)
	updates := map[string]any{
		"lease_until":       &leaseUntil,
		"last_heartbeat_at": &now,
	}
	if strings.TrimSpace(input.ExecutorID) != "" {
		updates["executor_id"] = strings.TrimSpace(input.ExecutorID)
	}
	if strings.TrimSpace(input.ExecutorName) != "" {
		updates["executor_name"] = strings.TrimSpace(input.ExecutorName)
	}
	if err := h.db.Model(&model.BuildJob{}).
		Where("id = ? and builder_id = ? and lease_token = ? and status = ?", job.ID, job.BuilderID, job.LeaseToken, "running").
		Updates(updates).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"leaseUntil": leaseUntil})
}

var builderProgressKeys = map[string]bool{
	"claimed":              true,
	"clone_repository":     true,
	"load_dockerfile":      true,
	"pull_image_metadata":  true,
	"pull_base_image":      true,
	"upload_build_context": true,
	"run_command":          true,
	"export_image":         true,
	"push_image_layers":    true,
	"push_image_manifest":  true,
	"registry_auth":        true,
}

func normalizeBuilderProgressKey(progress string) string {
	progress = strings.TrimSpace(progress)
	if builderProgressKeys[progress] {
		return progress
	}
	return ""
}

func (h *Handlers) CompleteBuilderTask(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	job, ok := h.builderRunningJobForAgentLease(ctx)
	if !ok {
		return
	}
	var input builderCompleteInput
	if !bindJSON(ctx, &input) {
		return
	}
	finishedAt := time.Now()
	imageDigest := strings.TrimSpace(input.ImageDigest)
	var completedRun model.BuildRun
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var lockedJob model.BuildJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedJob, "id = ? and builder_id = ? and lease_token = ?", job.ID, job.BuilderID, job.LeaseToken).Error; err != nil {
			return err
		}
		var run model.BuildRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ? and project_id = ?", lockedJob.BuildRunID, lockedJob.ProjectID).Error; err != nil {
			return err
		}
		if lockedJob.Status != "running" || run.Status == "canceled" {
			return nil
		}
		imageRef := fallback(strings.TrimSpace(input.ImageRef), run.ImageRef)
		sourceCommit := fallback(strings.TrimSpace(input.SourceCommit), run.SourceCommit)
		sourceAuthorName := fallback(strings.TrimSpace(input.SourceAuthorName), run.SourceAuthorName)
		sourceAuthorEmail := fallback(strings.TrimSpace(input.SourceAuthorEmail), run.SourceAuthorEmail)
		run.Status = "succeeded"
		run.ImageRef = imageRef
		run.ImageDigest = imageDigest
		run.SourceCommit = sourceCommit
		run.SourceAuthorName = sourceAuthorName
		run.SourceAuthorEmail = sourceAuthorEmail
		run.FinishedAt = &finishedAt
		if err := tx.Model(&model.BuildJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":            "succeeded",
			"message":           fallback(strings.TrimSpace(input.Message), "builder task succeeded"),
			"lease_token":       "",
			"lease_until":       nil,
			"last_heartbeat_at": &finishedAt,
			"finished_at":       &finishedAt,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.BuildRun{}).Where("id = ?", run.ID).Updates(map[string]any{
			"status":              "succeeded",
			"image_ref":           imageRef,
			"image_digest":        imageDigest,
			"source_commit":       sourceCommit,
			"source_author_name":  sourceAuthorName,
			"source_author_email": sourceAuthorEmail,
			"finished_at":         &finishedAt,
		}).Error; err != nil {
			return err
		}
		if imageRef != "" {
			image := containerImageFromBuildRun(run, imageRef, imageDigest, sourceCommit)
			if image.ID != "" {
				if err := tx.Create(&image).Error; err != nil {
					return err
				}
			}
		}
		completedRun = run
		return nil
	}); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if completedRun.ID != "" {
		h.enqueueAutoDeploymentsForBuildRun(ctx.Request.Context(), completedRun)
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) FailBuilderTask(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	job, ok := h.builderRunningJobForAgentLease(ctx)
	if !ok {
		return
	}
	var input builderFailInput
	if !bindJSON(ctx, &input) {
		return
	}
	finishedAt := time.Now()
	message := fallback(strings.TrimSpace(input.Message), "builder task failed")
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var lockedJob model.BuildJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedJob, "id = ? and builder_id = ? and lease_token = ?", job.ID, job.BuilderID, job.LeaseToken).Error; err != nil {
			return err
		}
		var run model.BuildRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ? and project_id = ?", lockedJob.BuildRunID, lockedJob.ProjectID).Error; err != nil {
			return err
		}
		if lockedJob.Status != "running" || run.Status == "canceled" {
			return nil
		}
		if err := tx.Model(&model.BuildJob{}).Where("id = ?", lockedJob.ID).Updates(map[string]any{
			"status":            "failed",
			"message":           message,
			"lease_token":       "",
			"lease_until":       nil,
			"last_heartbeat_at": &finishedAt,
			"finished_at":       &finishedAt,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.BuildRun{}).Where("id = ?", run.ID).Updates(map[string]any{
			"status":      "failed",
			"finished_at": &finishedAt,
		}).Error
	}); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) GetBuilderTaskCancelled(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	job, ok := h.builderJobForAgent(ctx)
	if !ok {
		return
	}
	if !builderLeaseTokenMatches(ctx, job) {
		writeError(ctx, http.StatusConflict, "builder task lease is no longer active")
		return
	}
	var run model.BuildRun
	if err := h.db.First(&run, "id = ? and project_id = ?", job.BuildRunID, job.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build run not found")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"cancelled": job.Status == "canceled" || run.Status == "canceled"})
}

func (h *Handlers) AppendBuilderHookRunLogs(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	hookRun, ok := h.builderHookRunForAgent(ctx)
	if !ok {
		return
	}
	var input builderLogsInput
	if !bindJSON(ctx, &input) {
		return
	}
	if err := h.appendHookRunLog(hookRun, input.Content); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) CompleteBuilderHookRun(ctx *gin.Context) {
	if !h.authorizeBuilder(ctx) {
		return
	}
	hookRun, ok := h.builderHookRunForAgent(ctx)
	if !ok {
		return
	}
	var input builderHookCompleteInput
	if !bindJSON(ctx, &input) {
		return
	}
	status := "succeeded"
	if !input.Succeeded {
		status = "failed"
	}
	finishedAt := time.Now()
	if err := h.db.Model(&model.HookRun{}).Where("id = ?", hookRun.ID).Updates(map[string]any{
		"status":      status,
		"exit_code":   input.ExitCode,
		"message":     strings.TrimSpace(input.Message),
		"finished_at": &finishedAt,
	}).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) authorizeBuilder(ctx *gin.Context) bool {
	expected := strings.TrimSpace(h.builderToken)
	if expected == "" {
		writeError(ctx, http.StatusServiceUnavailable, "builder token is not configured")
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(ctx.GetHeader("Authorization"), "Bearer "))
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		writeError(ctx, http.StatusUnauthorized, "invalid builder token")
		return false
	}
	return true
}

func (h *Handlers) builderHookRunForAgent(ctx *gin.Context) (model.HookRun, bool) {
	runID := strings.TrimSpace(ctx.Param("runId"))
	if runID == "" {
		writeError(ctx, http.StatusBadRequest, "hook run id is required")
		return model.HookRun{}, false
	}
	var hookRun model.HookRun
	if err := h.db.First(&hookRun, "id = ?", runID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "hook run not found")
		return model.HookRun{}, false
	}
	var job model.BuildJob
	if err := h.db.First(&job, "id = ? and project_id = ? and status = ?", hookRun.BuildJobID, hookRun.ProjectID, "running").Error; err != nil {
		writeError(ctx, http.StatusConflict, "hook run build job is not running")
		return model.HookRun{}, false
	}
	leaseToken := strings.TrimSpace(ctx.Query("leaseToken"))
	if leaseToken == "" || leaseToken != job.LeaseToken {
		writeError(ctx, http.StatusConflict, "builder task lease is no longer active")
		return model.HookRun{}, false
	}
	return hookRun, true
}

func (h *Handlers) claimBuildTask(heartbeat builderHeartbeatInput) (builderTaskResponse, bool, error) {
	now := time.Now()
	leaseUntil := now.Add(time.Duration(config.Load().BuilderTaskLeaseSeconds) * time.Second)
	leaseToken := id.New("blse")
	var response builderTaskResponse
	agentID := strings.TrimSpace(heartbeat.AgentID)
	labels := strings.Join(normalizeStringList(heartbeat.Labels), ",")
	scopes := strings.Join(normalizeStringList(heartbeat.Scopes), ",")
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var jobs []model.BuildJob
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", "queued").
			Order("created_at asc").
			Limit(20).
			Find(&jobs).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, job := range jobs {
			var run model.BuildRun
			if err := tx.First(&run, "id = ? and project_id = ?", job.BuildRunID, job.ProjectID).Error; err != nil {
				return err
			}
			if job.Status == "canceled" || run.Status == "canceled" {
				continue
			}
			if !builderHasLabels(labels, normalizeBuildSelectorList(strings.Split(run.BuildLabels, ","))) {
				continue
			}
			if !builderAllowsRun(scopes, run.ProjectID, run.CreatedBy) {
				continue
			}
			if h.buildRunBlockedByConfigConcurrency(tx, run, job.ID, now) {
				continue
			}
			payload, err := h.builderPayloadForRun(tx, run, job)
			if err != nil {
				finishedAt := time.Now()
				if updateErr := tx.Model(&model.BuildJob{}).Where("id = ?", job.ID).Updates(map[string]any{
					"status":      "failed",
					"message":     err.Error(),
					"finished_at": &finishedAt,
				}).Error; updateErr != nil {
					return updateErr
				}
				return tx.Model(&model.BuildRun{}).Where("id = ?", run.ID).Updates(map[string]any{
					"status":      "failed",
					"finished_at": &finishedAt,
				}).Error
			}
			if err := tx.Model(&model.BuildJob{}).Where("id = ?", job.ID).Updates(map[string]any{
				"status":            "running",
				"builder_id":        agentID,
				"lease_token":       leaseToken,
				"lease_until":       &leaseUntil,
				"last_heartbeat_at": &now,
				"log_ref":           "builder:" + agentID + "/" + job.ID,
				"message":           "claimed",
				"attempts":          job.Attempts + 1,
				"started_at":        nullableStartTime(job.StartedAt, now),
				"finished_at":       nil,
			}).Error; err != nil {
				return err
			}
			runUpdates := map[string]any{"status": "running"}
			if run.StartedAt == nil {
				runUpdates["started_at"] = &now
			}
			if err := tx.Model(&model.BuildRun{}).Where("id = ?", run.ID).Updates(runUpdates).Error; err != nil {
				return err
			}
			response = payload
			response.LeaseToken = leaseToken
			response.LeaseUntil = leaseUntil
			return nil
		}
		return nil
	})
	if err != nil {
		return builderTaskResponse{}, false, err
	}
	return response, response.JobID != "", nil
}

func (h *Handlers) buildRunBlockedByConfigConcurrency(tx *gorm.DB, run model.BuildRun, jobID string, now time.Time) bool {
	if strings.TrimSpace(run.ModuleID) == "" {
		return false
	}
	var config model.ApplicationModule
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&config, "id = ? and project_id = ? and application_id = ?", run.ModuleID, run.ProjectID, run.ApplicationID).Error; err != nil {
		return false
	}
	if strings.TrimSpace(config.ConcurrencyPolicy) == "parallel" {
		return false
	}
	var count int64
	err := tx.Model(&model.BuildRun{}).
		Joins("join build_jobs on build_jobs.build_run_id = build_runs.id").
		Where("build_runs.project_id = ? and build_runs.application_id = ? and build_runs.module_id = ? and build_runs.status = ? and build_runs.id <> ? and build_jobs.id <> ? and build_jobs.status = ? and (build_jobs.lease_until is null or build_jobs.lease_until >= ?)",
			run.ProjectID,
			run.ApplicationID,
			run.ModuleID,
			"running",
			run.ID,
			jobID,
			"running",
			now,
		).
		Count(&count).Error
	return err == nil && count > 0
}

func (h *Handlers) recordBuilderHeartbeat(input builderHeartbeatInput) error {
	agentID := strings.TrimSpace(input.AgentID)
	if agentID == "" {
		return errors.New("builder agent id is required")
	}
	now := time.Now()
	agent := model.BuilderAgent{
		ID:                 agentID,
		Name:               fallback(strings.TrimSpace(input.Name), agentID),
		Labels:             strings.Join(normalizeStringList(input.Labels), ","),
		Scopes:             strings.Join(normalizeStringList(input.Scopes), ","),
		Executor:           fallback(strings.TrimSpace(input.Executor), "docker"),
		Status:             "online",
		MaxConcurrency:     fallbackInt(input.MaxConcurrency, 1),
		CurrentConcurrency: input.CurrentConcurrency,
		LastHeartbeatAt:    &now,
	}
	return h.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "labels", "scopes", "executor", "status", "max_concurrency", "current_concurrency", "last_heartbeat_at", "updated_at",
		}),
	}).Create(&agent).Error
}

func heartbeatFromClaimInput(input builderClaimInput) builderHeartbeatInput {
	return builderHeartbeatInput{
		AgentID:            input.AgentID,
		Name:               input.Name,
		Labels:             input.Labels,
		Scopes:             input.Scopes,
		Executor:           input.Executor,
		MaxConcurrency:     input.MaxConcurrency,
		CurrentConcurrency: input.CurrentConcurrency,
	}
}

func (h *Handlers) builderPayloadForRun(tx *gorm.DB, run model.BuildRun, job model.BuildJob) (builderTaskResponse, error) {
	var binding model.RepositoryBinding
	bindingQuery := tx.Where("project_id = ? and application_id = ?", run.ProjectID, run.ApplicationID)
	if strings.TrimSpace(run.ModuleID) != "" {
		var config model.ApplicationModule
		if err := tx.First(&config, "id = ? and project_id = ? and application_id = ?", run.ModuleID, run.ProjectID, run.ApplicationID).Error; err != nil {
			return builderTaskResponse{}, fmt.Errorf("module not found: %w", err)
		}
		if strings.TrimSpace(config.RepositoryBindingID) != "" {
			bindingQuery = bindingQuery.Where("id = ?", config.RepositoryBindingID)
		}
	}
	if err := bindingQuery.First(&binding).Error; err != nil {
		return builderTaskResponse{}, fmt.Errorf("repository binding not found: %w", err)
	}
	var gitAccount model.GitAccount
	if err := tx.First(&gitAccount, "id = ?", binding.GitAccountID).Error; err != nil {
		return builderTaskResponse{}, fmt.Errorf("git account not found: %w", err)
	}
	gitToken := h.secrets.Resolve(gitAccount.AccessTokenRef)
	if strings.TrimSpace(gitToken) == "" && !repositoryBindingLooksPublic(binding) {
		return builderTaskResponse{}, errors.New("git access token is missing")
	}
	var registry model.ArtifactRegistry
	if strings.TrimSpace(run.TargetRegistryID) == "" {
		return builderTaskResponse{}, errors.New("target registry is required")
	}
	if err := tx.First(&registry, "id = ?", run.TargetRegistryID).Error; err != nil {
		return builderTaskResponse{}, fmt.Errorf("target registry not found: %w", err)
	}
	credential, err := h.registryCredentialForBuild(run.CreatedBy, registry)
	if err != nil {
		return builderTaskResponse{}, err
	}
	registrySecret := h.secrets.Resolve(credential.TokenRef)
	if registrySecret == "" {
		registrySecret = h.secrets.Resolve(credential.PasswordRef)
	}
	if strings.TrimSpace(registrySecret) == "" {
		return builderTaskResponse{}, errors.New("registry credential secret is missing")
	}
	var project model.Project
	if err := tx.First(&project, "id = ?", run.ProjectID).Error; err != nil {
		return builderTaskResponse{}, fmt.Errorf("project not found: %w", err)
	}
	var application model.Application
	if err := tx.First(&application, "id = ? and project_id = ?", run.ApplicationID, run.ProjectID).Error; err != nil {
		return builderTaskResponse{}, fmt.Errorf("application not found: %w", err)
	}
	if strings.TrimSpace(run.TargetRepository) == "" {
		run.TargetRepository = buildTargetImageRepository(registry, project, application)
	}
	imageRef := fallback(strings.TrimSpace(run.ImageRef), buildImageRef(registry, run))
	var actor model.User
	if err := tx.First(&actor, "id = ?", run.CreatedBy).Error; err != nil {
		return builderTaskResponse{}, fmt.Errorf("build actor not found: %w", err)
	}
	buildEnv, err := h.buildVariablesForRunByIDs(tx, actor, run.ProjectID, buildVariableSetIDs(run.BuildVariableSetIDs))
	if err != nil {
		return builderTaskResponse{}, fmt.Errorf("build variables are unavailable: %w", err)
	}
	hooks, err := h.builderHookPayloadsForRun(tx, run, job)
	if err != nil {
		return builderTaskResponse{}, err
	}
	return builderTaskResponse{
		JobID:         job.ID,
		BuildRunID:    run.ID,
		ProjectID:     run.ProjectID,
		ApplicationID: run.ApplicationID,
		ModuleID:      run.ModuleID,
		Repository: builderRepositoryPayload{
			CloneURL:     binding.CloneURL,
			Owner:        binding.Owner,
			Repo:         binding.Repo,
			SourceBranch: fallback(run.SourceBranch, binding.DefaultBranch),
			SourceTag:    run.SourceTag,
			SourceCommit: run.SourceCommit,
			AccessToken:  gitToken,
		},
		Build: builderBuildPayload{
			DockerfilePath: fallback(run.DockerfilePath, "Dockerfile"),
			BuildContext:   fallback(run.BuildContext, "."),
			BuildDirectory: run.BuildDirectory,
			Env:            buildEnv,
			Hooks:          hooks,
		},
		Registry: builderRegistryPayload{
			Endpoint:         registryAuthEndpointForBuilder(registry.Endpoint),
			Username:         credential.Username,
			Password:         registrySecret,
			ImageRef:         imageRef,
			ImageNamePrefix:  buildImageNamePrefix(registry, run.TargetRepository),
			ImageTagTemplate: fallback(strings.TrimSpace(run.TargetTag), "latest"),
		},
	}, nil
}

func repositoryBindingLooksPublic(binding model.RepositoryBinding) bool {
	cloneURL := strings.ToLower(strings.TrimSpace(binding.CloneURL))
	return strings.HasPrefix(cloneURL, "https://github.com/") ||
		strings.HasPrefix(cloneURL, "https://gitea.com/") ||
		strings.HasPrefix(cloneURL, "https://gitlab.com/")
}

func (h *Handlers) builderHookPayloadsForRun(tx *gorm.DB, run model.BuildRun, job model.BuildJob) ([]builderHookPayload, error) {
	if strings.TrimSpace(run.ModuleID) == "" {
		return nil, nil
	}
	var module model.ApplicationModule
	if err := tx.First(&module, "id = ? and project_id = ? and application_id = ?", run.ModuleID, run.ProjectID, run.ApplicationID).Error; err != nil {
		return nil, err
	}
	if !module.BuildHooksEnabled {
		return nil, nil
	}
	var bindings []model.ApplicationModuleHookBinding
	if err := tx.Where("project_id = ? and application_id = ? and module_id = ?", run.ProjectID, run.ApplicationID, run.ModuleID).
		Order("run_order asc, created_at asc").
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	hookIDs := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		hookIDs = append(hookIDs, binding.HookConfigID)
	}
	var configs []model.ProjectHookConfig
	if err := tx.Where("project_id = ? and id in ? and phase in ?", run.ProjectID, hookIDs, []string{hookPhasePreBuild, hookPhasePostBuild}).
		Find(&configs).Error; err != nil {
		return nil, err
	}
	configsByID := make(map[string]model.ProjectHookConfig, len(configs))
	for _, config := range configs {
		configsByID[config.ID] = config
	}
	hooks := make([]builderHookPayload, 0, len(configs))
	for _, binding := range bindings {
		config, ok := configsByID[binding.HookConfigID]
		if !ok {
			continue
		}
		runRecord := model.HookRun{
			ID:             id.New("hrun"),
			ProjectID:      run.ProjectID,
			HookConfigID:   config.ID,
			BuildRunID:     run.ID,
			BuildJobID:     job.ID,
			ApplicationID:  run.ApplicationID,
			ModuleID:       run.ModuleID,
			Name:           config.Name,
			Phase:          config.Phase,
			Status:         "queued",
			ScriptSnapshot: config.Script,
			Shell:          config.Shell,
			TimeoutSeconds: config.TimeoutSeconds,
			FailurePolicy:  config.FailurePolicy,
		}
		if err := tx.Create(&runRecord).Error; err != nil {
			return nil, err
		}
		hooks = append(hooks, builderHookPayload{
			ID:             runRecord.ID,
			Name:           runRecord.Name,
			Phase:          runRecord.Phase,
			Script:         runRecord.ScriptSnapshot,
			Shell:          runRecord.Shell,
			TimeoutSeconds: runRecord.TimeoutSeconds,
			FailurePolicy:  runRecord.FailurePolicy,
		})
	}
	return hooks, nil
}

func (h *Handlers) registryCredentialForBuild(actorID string, registry model.ArtifactRegistry) (model.RegistryCredential, error) {
	var credential model.RegistryCredential
	if strings.TrimSpace(registry.CredentialRef) != "" {
		err := h.db.First(&credential, "id = ? and registry_id = ? and scope in ? and (access_scope = ? or created_by = ?)",
			registry.CredentialRef, registry.ID, []string{"push", "push-pull"}, "registry", actorID).Error
		if err == nil {
			return credential, nil
		}
	}
	err := h.db.Where("registry_id = ? and access_scope = ? and created_by = ? and scope in ?",
		registry.ID, "personal", actorID, []string{"push", "push-pull"}).Order("created_at desc").First(&credential).Error
	if err == nil {
		return credential, nil
	}
	if registry.Scope != "global" {
		err = h.db.Where("registry_id = ? and access_scope = ? and scope in ?",
			registry.ID, "registry", []string{"push", "push-pull"}).Order("created_at desc").First(&credential).Error
		if err == nil {
			return credential, nil
		}
	}
	return model.RegistryCredential{}, errors.New("usable registry credential not found")
}

func (h *Handlers) builderJobForAgent(ctx *gin.Context) (model.BuildJob, bool) {
	agentID := strings.TrimSpace(ctx.Query("agentId"))
	var job model.BuildJob
	if agentID == "" {
		writeError(ctx, http.StatusBadRequest, "agentId is required")
		return job, false
	}
	if err := h.db.First(&job, "id = ? and builder_id = ?", ctx.Param("jobId"), agentID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "builder task not found")
		return job, false
	}
	return job, true
}

func (h *Handlers) builderRunningJobForAgentLease(ctx *gin.Context) (model.BuildJob, bool) {
	job, ok := h.builderJobForAgent(ctx)
	if !ok {
		return job, false
	}
	if !builderLeaseTokenMatches(ctx, job) {
		writeError(ctx, http.StatusConflict, "builder task lease is no longer active")
		return job, false
	}
	if job.Status != "running" {
		writeError(ctx, http.StatusConflict, "builder task is not running")
		return job, false
	}
	return job, true
}

func builderLeaseTokenMatches(ctx *gin.Context, job model.BuildJob) bool {
	token := strings.TrimSpace(ctx.Query("leaseToken"))
	return token != "" && token == strings.TrimSpace(job.LeaseToken)
}

func (h *Handlers) appendBuildLog(job model.BuildJob, content string) error {
	content = trimBuildLogContent(content)
	if content == "" {
		return nil
	}
	var existing model.BuildLog
	err := h.db.First(&existing, "build_job_id = ?", job.ID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return h.db.Create(&model.BuildLog{
			ID:         id.New("blog"),
			BuildRunID: job.BuildRunID,
			BuildJobID: job.ID,
			ProjectID:  job.ProjectID,
			Content:    content,
		}).Error
	}
	if err != nil {
		return err
	}
	existing.Content = trimBuildLogContent(existing.Content + "\n" + content)
	return h.db.Save(&existing).Error
}

func nullableStartTime(existing *time.Time, now time.Time) any {
	if existing != nil {
		return existing
	}
	return &now
}

func registryAuthEndpointForBuilder(endpoint string) string {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" {
		return strings.TrimSpace(endpoint)
	}
	host := strings.ToLower(parsed.Host)
	if host == "registry-1.docker.io" || host == "docker.io" || host == "index.docker.io" {
		return "https://index.docker.io/v1/"
	}
	return parsed.Host
}

func buildImageRef(registry model.ArtifactRegistry, run model.BuildRun) string {
	repository := strings.Trim(strings.TrimSpace(run.TargetRepository), "/")
	if repository == "" {
		return ""
	}
	tag := renderBuildTagTemplate(fallback(strings.TrimSpace(run.TargetTag), "latest"), variables.Context{SourceBranch: run.SourceBranch, SourceTag: run.SourceTag, SourceCommit: run.SourceCommit})
	if hasRegistryHost(repository) || isDockerHubRegistry(registry) {
		return repository + ":" + tag
	}
	endpoint := registryImageHost(registry.Endpoint)
	if endpoint != "" {
		return endpoint + "/" + repository + ":" + tag
	}
	return repository + ":" + tag
}

func buildTargetImageRepository(registry model.ArtifactRegistry, project model.Project, application model.Application) string {
	projectSlug := dnsSafeSegment(project.Slug)
	appSlug := dnsSafeSegment(application.Slug)
	if strings.TrimSpace(application.Slug) == "" {
		appSlug = dnsSafeSegment(application.Name)
	}
	repository := projectSlug + "-" + appSlug
	if namespace := strings.Trim(strings.TrimSpace(registry.Namespace), "/"); namespace != "" {
		repository = namespace + "/" + repository
	}
	return buildImageNamePrefix(registry, repository)
}

func buildImageNamePrefix(registry model.ArtifactRegistry, repository string) string {
	repository = strings.Trim(strings.TrimSpace(repository), "/")
	if repository == "" {
		return ""
	}
	if hasRegistryHost(repository) || isDockerHubRegistry(registry) {
		return repository
	}
	host := registryImageHost(registry.Endpoint)
	if host == "" {
		return repository
	}
	return strings.TrimRight(host, "/") + "/" + repository
}

func isDockerHubRegistry(registry model.ArtifactRegistry) bool {
	provider := strings.ToLower(strings.TrimSpace(registry.Provider))
	if provider == "dockerhub" || provider == "docker-hub" {
		return true
	}
	host := registryImageHost(registry.Endpoint)
	return host == "docker.io" || host == "registry-1.docker.io" || host == "index.docker.io"
}

func hasRegistryHost(repository string) bool {
	first := strings.Split(strings.Trim(repository, "/"), "/")[0]
	return strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost"
}

func renderBuildTagTemplate(template string, ctx variables.Context) string {
	return sanitizeImageTag(variables.Render(fallback(strings.TrimSpace(template), "latest"), ctx))
}

func sanitizeImageTag(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "latest"
	}
	var builder strings.Builder
	for _, char := range value {
		if char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_' || char == '.' || char == '-' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('-')
	}
	output := strings.Trim(builder.String(), ".-")
	if output == "" {
		return "latest"
	}
	if len(output) > 128 {
		output = output[:128]
	}
	return output
}

func dnsSafeSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousDash := false
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			builder.WriteRune(char)
			previousDash = false
			continue
		}
		if !previousDash {
			builder.WriteByte('-')
			previousDash = true
		}
	}
	output := strings.Trim(builder.String(), "-")
	if output == "" {
		return "app"
	}
	return output
}

func registryImageHost(endpoint string) string {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" {
		return strings.TrimSpace(endpoint)
	}
	host := strings.ToLower(parsed.Host)
	if host == "registry-1.docker.io" || host == "index.docker.io" {
		return "docker.io"
	}
	return parsed.Host
}

func containerImageFromBuildRun(run model.BuildRun, imageRef string, digest string, sourceCommit string) model.ContainerImage {
	if strings.TrimSpace(run.TargetRegistryID) == "" || strings.TrimSpace(run.TargetRepository) == "" {
		return model.ContainerImage{}
	}
	return model.ContainerImage{
		ID:            id.New("img"),
		ProjectID:     run.ProjectID,
		ApplicationID: run.ApplicationID,
		RegistryID:    run.TargetRegistryID,
		Repository:    strings.Trim(strings.TrimSpace(run.TargetRepository), "/"),
		Tag:           fallback(strings.TrimSpace(run.TargetTag), "latest"),
		Digest:        strings.TrimSpace(digest),
		ImageRef:      strings.TrimSpace(imageRef),
		SourceType:    "build",
		BuildRunID:    run.ID,
		SourceCommit:  strings.TrimSpace(sourceCommit),
		ScanStatus:    "unknown",
		CreatedBy:     run.CreatedBy,
	}
}

func trimBuildLogContent(content string) string {
	content = strings.TrimSpace(content)
	const maxLogBytes = 1024 * 1024
	if len(content) <= maxLogBytes {
		return content
	}
	return content[len(content)-maxLogBytes:]
}

func normalizeStringList(values []string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			output = append(output, value)
		}
	}
	return output
}

type builderHeartbeatInput struct {
	AgentID            string   `json:"agentId"`
	Name               string   `json:"name"`
	Labels             []string `json:"labels"`
	Scopes             []string `json:"scopes"`
	Executor           string   `json:"executor"`
	MaxConcurrency     int      `json:"maxConcurrency"`
	CurrentConcurrency int      `json:"currentConcurrency"`
}

type builderClaimInput struct {
	AgentID            string   `json:"agentId"`
	Name               string   `json:"name"`
	Labels             []string `json:"labels"`
	Scopes             []string `json:"scopes"`
	Executor           string   `json:"executor"`
	MaxConcurrency     int      `json:"maxConcurrency"`
	CurrentConcurrency int      `json:"currentConcurrency"`
}

type builderLogsInput struct {
	Content string `json:"content"`
}

type builderProgressInput struct {
	Key string `json:"key"`
}

type builderRenewInput struct {
	ExecutorID   string `json:"executorId"`
	ExecutorName string `json:"executorName"`
}

type builderCompleteInput struct {
	ImageRef          string `json:"imageRef"`
	ImageDigest       string `json:"imageDigest"`
	SourceCommit      string `json:"sourceCommit"`
	SourceAuthorName  string `json:"sourceAuthorName"`
	SourceAuthorEmail string `json:"sourceAuthorEmail"`
	Message           string `json:"message"`
}

type builderFailInput struct {
	Message string `json:"message"`
}

type builderHookCompleteInput struct {
	Succeeded bool   `json:"succeeded"`
	ExitCode  int    `json:"exitCode"`
	Message   string `json:"message"`
}

type builderTaskResponse struct {
	JobID         string                   `json:"jobId"`
	LeaseToken    string                   `json:"leaseToken"`
	LeaseUntil    time.Time                `json:"leaseUntil"`
	BuildRunID    string                   `json:"buildRunId"`
	ProjectID     string                   `json:"projectId"`
	ApplicationID string                   `json:"applicationId"`
	ModuleID      string                   `json:"moduleId"`
	Repository    builderRepositoryPayload `json:"repository"`
	Build         builderBuildPayload      `json:"build"`
	Registry      builderRegistryPayload   `json:"registry"`
}

type builderRepositoryPayload struct {
	CloneURL     string `json:"cloneUrl"`
	Owner        string `json:"owner"`
	Repo         string `json:"repo"`
	SourceBranch string `json:"sourceBranch"`
	SourceTag    string `json:"sourceTag"`
	SourceCommit string `json:"sourceCommit"`
	AccessToken  string `json:"accessToken"`
}

type builderBuildPayload struct {
	DockerfilePath string               `json:"dockerfilePath"`
	BuildContext   string               `json:"buildContext"`
	BuildDirectory string               `json:"buildDirectory"`
	Env            map[string]string    `json:"env"`
	Hooks          []builderHookPayload `json:"hooks"`
}

type builderHookPayload struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Phase          string `json:"phase"`
	Script         string `json:"script"`
	Shell          string `json:"shell"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	FailurePolicy  string `json:"failurePolicy"`
}

type builderRegistryPayload struct {
	Endpoint         string `json:"endpoint"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	ImageRef         string `json:"imageRef"`
	ImageNamePrefix  string `json:"imageNamePrefix"`
	ImageTagTemplate string `json:"imageTagTemplate"`
}
