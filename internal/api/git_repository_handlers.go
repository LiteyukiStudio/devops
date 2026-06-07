package api

import (
	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	gitprovider "github.com/LiteyukiStudio/devops/internal/provider/git"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
	"time"
)

func (h *Handlers) ListGitRepositories(ctx *gin.Context) {
	client, ok := h.gitClientForCurrentUserAccount(ctx, ctx.Param("accountId"))
	if !ok {
		return
	}
	page := positiveInt(ctx.DefaultQuery("page", "1"), 1)
	pageSize := positiveInt(ctx.DefaultQuery("pageSize", "50"), 50)
	if pageSize > 100 {
		pageSize = 100
	}
	repos, err := client.ListRepositories(ctx.Request.Context(), ctx.Query("search"), page, pageSize)
	if err != nil {
		writeGitUpstreamError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": repos, "page": page, "pageSize": pageSize})
}

func (h *Handlers) ListGitBranches(ctx *gin.Context) {
	client, ok := h.gitClientForCurrentUserAccount(ctx, ctx.Param("accountId"))
	if !ok {
		return
	}
	ref := strings.TrimSpace(ctx.Query("ref"))
	cacheKey := gitBranchCacheKey(ctx.Param("accountId"), ctx.Param("owner"), ctx.Param("repo"), ref)
	branches, ok := h.branchCache.get(cacheKey)
	if !ok {
		var err error
		branches, err = client.ListBranches(ctx.Request.Context(), ctx.Param("owner"), ctx.Param("repo"))
		if err != nil {
			writeGitUpstreamError(ctx, err)
			return
		}
		h.branchCache.set(cacheKey, branches)
	}
	limit := positiveInt(ctx.DefaultQuery("limit", "50"), 50)
	result := filterGitBranches(branches, ctx.Query("search"), limit)
	ctx.JSON(http.StatusOK, gin.H{
		"items":        result.items,
		"total":        len(branches),
		"matchedTotal": result.matchedTotal,
		"limited":      len(result.items) < result.matchedTotal,
	})
}

func (h *Handlers) ReadGitFile(ctx *gin.Context) {
	client, ok := h.gitClientForCurrentUserAccount(ctx, ctx.Param("accountId"))
	if !ok {
		return
	}
	filePath := strings.TrimSpace(ctx.Query("path"))
	if filePath == "" {
		writeError(ctx, http.StatusBadRequest, "file path is required")
		return
	}
	file, err := client.ReadFile(ctx.Request.Context(), ctx.Param("owner"), ctx.Param("repo"), filePath, ctx.Query("ref"))
	if err != nil {
		writeGitUpstreamError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, file)
}

func (h *Handlers) ListGitContents(ctx *gin.Context) {
	client, ok := h.gitClientForCurrentUserAccount(ctx, ctx.Param("accountId"))
	if !ok {
		return
	}
	items, err := client.ListContents(ctx.Request.Context(), ctx.Param("owner"), ctx.Param("repo"), ctx.Query("path"), ctx.Query("ref"))
	if err != nil {
		writeGitUpstreamError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, items)
}

func (h *Handlers) GetGitRepositoryBuildOptions(ctx *gin.Context) {
	client, ok := h.gitClientForCurrentUserAccount(ctx, ctx.Param("accountId"))
	if !ok {
		return
	}
	started := time.Now()
	options, err := client.DiscoverBuildOptions(ctx.Request.Context(), ctx.Param("owner"), ctx.Param("repo"), ctx.Query("ref"))
	if err != nil {
		writeGitUpstreamError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"dockerfiles": options.Dockerfiles,
		"directories": options.Directories,
		"strategy":    options.Strategy,
		"truncated":   options.Truncated,
		"durationMs":  time.Since(started).Milliseconds(),
	})
}

func (h *Handlers) ListRepositoryBindings(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}

	var bindings []repositoryBindingResponse
	err := h.db.Table("repository_bindings").
		Select("repository_bindings.*, git_providers.name as provider_name, git_providers.type as provider_type, git_accounts.username as account_username, users.email as account_owner_email, users.name as account_owner_name, applications.name as application_name").
		Joins("join git_providers on git_providers.id = repository_bindings.git_provider_id and git_providers.deleted_at is null").
		Joins("join git_accounts on git_accounts.id = repository_bindings.git_account_id and git_accounts.deleted_at is null").
		Joins("join users on users.id = git_accounts.user_id and users.deleted_at is null").
		Joins("join applications on applications.id = repository_bindings.application_id and applications.deleted_at is null").
		Where("repository_bindings.project_id = ? and repository_bindings.deleted_at is null", ctx.Param("projectId")).
		Order("repository_bindings.created_at desc").
		Scan(&bindings).Error
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	for index := range bindings {
		bindings[index].CredentialRef = ""
	}
	ctx.JSON(http.StatusOK, bindings)
}

func (h *Handlers) CreateRepositoryBinding(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin", "developer"); !ok {
		return
	}

	var input repositoryBindingInput
	if !bindJSON(ctx, &input) {
		return
	}

	binding, ok := h.repositoryBindingFromInput(ctx, user.ID, input)
	if !ok {
		return
	}

	if err := h.db.Create(&binding).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	h.syncApplicationRepositoryURL(binding)
	binding.CredentialRef = ""
	ctx.JSON(http.StatusCreated, binding)
}

func (h *Handlers) UpdateRepositoryBinding(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin", "developer"); !ok {
		return
	}

	var existing model.RepositoryBinding
	if err := h.db.First(&existing, "id = ? and project_id = ?", ctx.Param("bindingId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "repository binding not found")
		return
	}

	var input repositoryBindingInput
	if !bindJSON(ctx, &input) {
		return
	}

	binding, ok := h.repositoryBindingFromInput(ctx, user.ID, input)
	if !ok {
		return
	}
	existing.ApplicationID = binding.ApplicationID
	existing.GitProviderID = binding.GitProviderID
	existing.GitAccountID = binding.GitAccountID
	existing.Owner = binding.Owner
	existing.Repo = binding.Repo
	existing.CloneURL = binding.CloneURL
	existing.DefaultBranch = binding.DefaultBranch
	existing.WebhookStatus = binding.WebhookStatus
	existing.CredentialRef = ""

	if err := h.db.Save(&existing).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	h.syncApplicationRepositoryURL(existing)
	existing.CredentialRef = ""
	ctx.JSON(http.StatusOK, existing)
}

func (h *Handlers) DeleteRepositoryBinding(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin", "developer"); !ok {
		return
	}

	var binding model.RepositoryBinding
	if err := h.db.First(&binding, "id = ? and project_id = ?", ctx.Param("bindingId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "repository binding not found")
		return
	}
	if err := h.db.Delete(&binding).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) CreateRepositoryWebhook(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin", "developer"); !ok {
		return
	}
	var binding model.RepositoryBinding
	if err := h.db.First(&binding, "id = ? and project_id = ?", ctx.Param("bindingId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "repository binding not found")
		return
	}
	account, ok := h.findGitAccountForUser(ctx, user.ID, binding.GitAccountID)
	if !ok {
		return
	}
	provider, ok := h.findEnabledGitProvider(ctx, binding.GitProviderID)
	if !ok {
		return
	}
	client := gitprovider.NewClientWithPolicy(provider, h.secrets.Resolve(account.AccessTokenRef), h.egressPolicyForUser(user))
	secret := randomHex(32)
	result, err := client.CreateWebhook(ctx.Request.Context(), binding.Owner, binding.Repo, h.gitWebhookURL(ctx, binding.ID), secret)
	if err != nil {
		binding.WebhookStatus = "failed"
		_ = h.db.Save(&binding).Error
		writeGitUpstreamError(ctx, err)
		return
	}
	binding.WebhookStatus = "created"
	binding.WebhookID = result.ID
	binding.WebhookSecret = h.secrets.Store(secret, user.ID, "repository_binding:"+binding.ID+":webhook")
	if err := h.db.Save(&binding).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(user.ID, "git_webhook.create", binding.ID, true, binding.WebhookID)
	ctx.JSON(http.StatusOK, binding)
}

func (h *Handlers) ReceiveGitWebhook(ctx *gin.Context) {
	var binding model.RepositoryBinding
	if err := h.db.First(&binding, "id = ?", ctx.Param("bindingId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "repository binding not found")
		return
	}
	body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, 2*1024*1024))
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid webhook body")
		return
	}
	if !verifyGitWebhookSignature(ctx.Request.Header, body, h.secrets.Resolve(binding.WebhookSecret)) {
		writeError(ctx, http.StatusUnauthorized, "invalid webhook signature")
		return
	}
	event := gitWebhookEvent(ctx.Request.Header)
	commitSHA := gitWebhookCommitSHA(body)
	now := time.Now()
	binding.WebhookStatus = "created"
	binding.LastEvent = event
	binding.LastCommitSHA = commitSHA
	binding.LastWebhookAt = &now
	if err := h.db.Save(&binding).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"accepted": true, "event": event, "commitSha": commitSHA})
}

func (h *Handlers) repositoryBindingFromInput(ctx *gin.Context, userID string, input repositoryBindingInput) (model.RepositoryBinding, bool) {
	account, ok := h.findGitAccountForUser(ctx, userID, input.GitAccountID)
	if !ok {
		return model.RepositoryBinding{}, false
	}
	provider, ok := h.findEnabledGitProvider(ctx, account.ProviderID)
	if !ok {
		return model.RepositoryBinding{}, false
	}

	app, ok := h.findApplicationByID(ctx, input.ApplicationID)
	if !ok {
		return model.RepositoryBinding{}, false
	}

	owner := strings.TrimSpace(input.Owner)
	repo := strings.TrimSpace(input.Repo)
	if owner == "" || repo == "" {
		writeError(ctx, http.StatusBadRequest, "请输入仓库 owner 和 repo")
		return model.RepositoryBinding{}, false
	}

	cloneURL := strings.TrimSpace(input.CloneURL)
	if cloneURL == "" {
		cloneURL = defaultCloneURL(provider, owner, repo)
	}

	return model.RepositoryBinding{
		ID:            id.New("rpb"),
		ProjectID:     ctx.Param("projectId"),
		ApplicationID: app.ID,
		GitProviderID: provider.ID,
		GitAccountID:  account.ID,
		Owner:         owner,
		Repo:          repo,
		CloneURL:      cloneURL,
		DefaultBranch: fallback(strings.TrimSpace(input.DefaultBranch), "main"),
		WebhookStatus: normalizeWebhookStatus(input.WebhookStatus),
		CredentialRef: "",
	}, true
}

func (h *Handlers) gitClientForCurrentUserAccount(ctx *gin.Context, accountID string) (gitprovider.Client, bool) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return gitprovider.Client{}, false
	}
	account, ok := h.findGitAccountForUser(ctx, user.ID, accountID)
	if !ok {
		return gitprovider.Client{}, false
	}
	provider, ok := h.findEnabledGitProvider(ctx, account.ProviderID)
	if !ok {
		return gitprovider.Client{}, false
	}
	if gitAccountNeedsRefresh(account) {
		account, ok = h.refreshGitAccountForUser(ctx, user, account, provider)
		if !ok {
			return gitprovider.Client{}, false
		}
	}
	token := h.secrets.Resolve(account.AccessTokenRef)
	if token == "" {
		writeError(ctx, http.StatusBadRequest, "git account has no access token")
		return gitprovider.Client{}, false
	}
	return gitprovider.NewClientWithPolicy(provider, token, h.egressPolicyForUser(user)), true
}
