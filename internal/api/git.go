package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const gitOAuthStateTTL = 10 * time.Minute

func (h *Handlers) ListGitProviders(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	query := h.db.Order("created_at desc")
	if user.Role != "platform_admin" {
		query = query.Where("enabled = ?", true)
	}

	var providers []model.GitProvider
	if err := query.Find(&providers).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, providers)
}

func (h *Handlers) StartGitOAuth(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	provider, ok := h.findEnabledGitProvider(ctx, ctx.Param("providerId"))
	if !ok {
		return
	}
	if provider.AuthType != "oauth" {
		writeError(ctx, http.StatusBadRequest, "git provider does not use oauth")
		return
	}
	if strings.TrimSpace(provider.ClientID) == "" || strings.TrimSpace(provider.ClientSecretRef) == "" {
		writeError(ctx, http.StatusBadRequest, "git provider oauth client is not configured")
		return
	}

	state := "git_" + randomHex(32)
	oauthState := model.GitOAuthState{
		ID:           id.New("gst"),
		StateHash:    hashToken(state),
		ProviderID:   provider.ID,
		UserID:       user.ID,
		RedirectPath: sanitizeRedirectPath(ctx.DefaultQuery("redirect", "/projects")),
		ExpiresAt:    time.Now().Add(gitOAuthStateTTL),
	}
	if err := h.db.Create(&oauthState).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	oauthConfig, err := gitOAuthConfig(provider, h.gitOAuthRedirectURL(ctx))
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.Redirect(http.StatusFound, oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline))
}

func (h *Handlers) CompleteGitOAuth(ctx *gin.Context) {
	plainState := strings.TrimSpace(ctx.Query("state"))
	code := strings.TrimSpace(ctx.Query("code"))
	if plainState == "" || code == "" {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_callback_invalid")
		return
	}

	var oauthState model.GitOAuthState
	if err := h.db.First(&oauthState, "state_hash = ? and expires_at > ?", hashToken(plainState), time.Now()).Error; err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_state_invalid")
		return
	}
	_ = h.db.Delete(&oauthState).Error

	provider, ok := h.findEnabledGitProvider(ctx, oauthState.ProviderID)
	if !ok {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_provider_disabled")
		return
	}
	oauthConfig, err := gitOAuthConfig(provider, h.gitOAuthRedirectURL(ctx))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_provider_invalid")
		return
	}
	token, err := oauthConfig.Exchange(ctx.Request.Context(), code)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_code_invalid")
		return
	}

	client := newGitClient(provider, token.AccessToken)
	gitUser, err := client.currentUser(ctx.Request.Context())
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_user_failed")
		return
	}
	account, err := h.upsertGitAccountFromOAuth(oauthState.UserID, provider, gitUser, token)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_save_failed")
		return
	}

	redirectPath := oauthState.RedirectPath
	separator := "?"
	if strings.Contains(redirectPath, "?") {
		separator = "&"
	}
	ctx.Redirect(http.StatusFound, redirectPath+separator+"gitAccountId="+url.QueryEscape(account.ID))
}

func (h *Handlers) CreateGitProvider(ctx *gin.Context) {
	if !h.requirePlatformAdmin(ctx) {
		return
	}

	var input gitProviderInput
	if !bindJSON(ctx, &input) {
		return
	}

	provider := model.GitProvider{
		ID:              id.New("gitp"),
		Type:            normalizeGitProviderType(input.Type),
		Name:            strings.TrimSpace(input.Name),
		BaseURL:         strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
		AuthType:        normalizeGitAuthType(input.AuthType),
		ClientID:        strings.TrimSpace(input.ClientID),
		ClientSecretRef: strings.TrimSpace(input.ClientSecretRef),
		Enabled:         input.Enabled,
	}
	if provider.Name == "" {
		writeError(ctx, http.StatusBadRequest, "请输入 Git Provider 名称")
		return
	}
	if provider.BaseURL == "" {
		provider.BaseURL = defaultGitBaseURL(provider.Type)
	}

	if err := h.db.Create(&provider).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, provider)
}

func (h *Handlers) UpdateGitProvider(ctx *gin.Context) {
	if !h.requirePlatformAdmin(ctx) {
		return
	}

	var provider model.GitProvider
	if err := h.db.First(&provider, "id = ?", ctx.Param("providerId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git provider not found")
		return
	}

	var input gitProviderInput
	if !bindJSON(ctx, &input) {
		return
	}

	provider.Type = normalizeGitProviderType(input.Type)
	provider.Name = strings.TrimSpace(input.Name)
	provider.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	provider.AuthType = normalizeGitAuthType(input.AuthType)
	provider.ClientID = strings.TrimSpace(input.ClientID)
	provider.ClientSecretRef = strings.TrimSpace(input.ClientSecretRef)
	provider.Enabled = input.Enabled
	if provider.Name == "" {
		writeError(ctx, http.StatusBadRequest, "请输入 Git Provider 名称")
		return
	}
	if provider.BaseURL == "" {
		provider.BaseURL = defaultGitBaseURL(provider.Type)
	}

	if err := h.db.Save(&provider).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, provider)
}

func (h *Handlers) DeleteGitProvider(ctx *gin.Context) {
	if !h.requirePlatformAdmin(ctx) {
		return
	}

	var provider model.GitProvider
	if err := h.db.First(&provider, "id = ?", ctx.Param("providerId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git provider not found")
		return
	}
	if err := h.db.Delete(&provider).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) ListGitAccounts(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var accounts []model.GitAccount
	if err := h.db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&accounts).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, accounts)
}

func (h *Handlers) CreateGitAccount(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var input gitAccountInput
	if !bindJSON(ctx, &input) {
		return
	}
	if _, ok := h.findEnabledGitProvider(ctx, input.ProviderID); !ok {
		return
	}

	account := model.GitAccount{
		ID:              id.New("gita"),
		UserID:          user.ID,
		ProviderID:      strings.TrimSpace(input.ProviderID),
		ExternalUserID:  strings.TrimSpace(input.ExternalUserID),
		Username:        strings.TrimSpace(input.Username),
		AvatarURL:       strings.TrimSpace(input.AvatarURL),
		AccessTokenRef:  strings.TrimSpace(input.AccessTokenRef),
		RefreshTokenRef: strings.TrimSpace(input.RefreshTokenRef),
		Scopes:          strings.Join(normalizeList(input.Scopes, false), ","),
		Status:          normalizeGitAccountStatus(input.Status),
	}
	if account.Username == "" {
		writeError(ctx, http.StatusBadRequest, "请输入 Git 账号用户名")
		return
	}

	if err := h.db.Create(&account).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, account)
}

func (h *Handlers) UpdateGitAccount(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var account model.GitAccount
	if err := h.db.First(&account, "id = ? and user_id = ?", ctx.Param("accountId"), user.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git account not found")
		return
	}

	var input gitAccountInput
	if !bindJSON(ctx, &input) {
		return
	}
	if _, ok := h.findEnabledGitProvider(ctx, input.ProviderID); !ok {
		return
	}

	account.ProviderID = strings.TrimSpace(input.ProviderID)
	account.ExternalUserID = strings.TrimSpace(input.ExternalUserID)
	account.Username = strings.TrimSpace(input.Username)
	account.AvatarURL = strings.TrimSpace(input.AvatarURL)
	account.AccessTokenRef = strings.TrimSpace(input.AccessTokenRef)
	account.RefreshTokenRef = strings.TrimSpace(input.RefreshTokenRef)
	account.Scopes = strings.Join(normalizeList(input.Scopes, false), ",")
	account.Status = normalizeGitAccountStatus(input.Status)
	if account.Username == "" {
		writeError(ctx, http.StatusBadRequest, "请输入 Git 账号用户名")
		return
	}

	if err := h.db.Save(&account).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, account)
}

func (h *Handlers) DeleteGitAccount(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var account model.GitAccount
	if err := h.db.First(&account, "id = ? and user_id = ?", ctx.Param("accountId"), user.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git account not found")
		return
	}
	if err := h.db.Delete(&account).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) RefreshGitAccount(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	account, ok := h.findGitAccountForUser(ctx, user.ID, ctx.Param("accountId"))
	if !ok {
		return
	}
	provider, ok := h.findEnabledGitProvider(ctx, account.ProviderID)
	if !ok {
		return
	}
	refreshToken := resolveStoredSecretRef(account.RefreshTokenRef)
	if refreshToken == "" {
		writeError(ctx, http.StatusBadRequest, "git account has no refresh token")
		return
	}
	oauthConfig, err := gitOAuthConfig(provider, h.gitOAuthRedirectURL(ctx))
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	tokenSource := oauthConfig.TokenSource(ctx.Request.Context(), &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Minute),
	})
	token, err := tokenSource.Token()
	if err != nil {
		account.Status = "expired"
		_ = h.db.Save(&account).Error
		writeError(ctx, http.StatusBadRequest, "git token refresh failed")
		return
	}
	account.AccessTokenRef = storedSecretRef(token.AccessToken)
	if token.RefreshToken != "" {
		account.RefreshTokenRef = storedSecretRef(token.RefreshToken)
	}
	if !token.Expiry.IsZero() {
		account.ExpiresAt = &token.Expiry
	}
	account.Status = "connected"
	if err := h.db.Save(&account).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, account)
}

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
	repos, err := client.listRepositories(ctx.Request.Context(), ctx.Query("search"), page, pageSize)
	if err != nil {
		writeError(ctx, http.StatusBadGateway, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": repos, "page": page, "pageSize": pageSize})
}

func (h *Handlers) ListGitBranches(ctx *gin.Context) {
	client, ok := h.gitClientForCurrentUserAccount(ctx, ctx.Param("accountId"))
	if !ok {
		return
	}
	branches, err := client.listBranches(ctx.Request.Context(), ctx.Param("owner"), ctx.Param("repo"))
	if err != nil {
		writeError(ctx, http.StatusBadGateway, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, branches)
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
	file, err := client.readFile(ctx.Request.Context(), ctx.Param("owner"), ctx.Param("repo"), filePath, ctx.Query("ref"))
	if err != nil {
		writeError(ctx, http.StatusBadGateway, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, file)
}

func (h *Handlers) ListRepositoryBindings(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}

	var bindings []repositoryBindingResponse
	err := h.db.Table("repository_bindings").
		Select("repository_bindings.*, git_providers.name as provider_name, git_providers.type as provider_type, git_accounts.username as account_username, applications.name as application_name").
		Joins("join git_providers on git_providers.id = repository_bindings.git_provider_id").
		Joins("join git_accounts on git_accounts.id = repository_bindings.git_account_id").
		Joins("join applications on applications.id = repository_bindings.application_id").
		Where("repository_bindings.project_id = ? and repository_bindings.deleted_at is null", ctx.Param("projectId")).
		Order("repository_bindings.created_at desc").
		Scan(&bindings).Error
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
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
	existing.CredentialRef = binding.CredentialRef

	if err := h.db.Save(&existing).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	h.syncApplicationRepositoryURL(existing)
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
	client := newGitClient(provider, resolveStoredSecretRef(account.AccessTokenRef))
	secret := randomHex(32)
	result, err := client.createWebhook(ctx.Request.Context(), binding.Owner, binding.Repo, h.gitWebhookURL(ctx, binding.ID), secret)
	if err != nil {
		binding.WebhookStatus = "failed"
		_ = h.db.Save(&binding).Error
		writeError(ctx, http.StatusBadGateway, err.Error())
		return
	}
	binding.WebhookStatus = "created"
	binding.WebhookID = result.ID
	binding.WebhookSecret = storedSecretRef(secret)
	if err := h.db.Save(&binding).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
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
	if !verifyGitWebhookSignature(ctx.Request.Header, body, resolveStoredSecretRef(binding.WebhookSecret)) {
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
		CredentialRef: strings.TrimSpace(input.CredentialRef),
	}, true
}

func (h *Handlers) gitClientForCurrentUserAccount(ctx *gin.Context, accountID string) (gitClient, bool) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return gitClient{}, false
	}
	account, ok := h.findGitAccountForUser(ctx, user.ID, accountID)
	if !ok {
		return gitClient{}, false
	}
	provider, ok := h.findEnabledGitProvider(ctx, account.ProviderID)
	if !ok {
		return gitClient{}, false
	}
	token := resolveStoredSecretRef(account.AccessTokenRef)
	if token == "" {
		writeError(ctx, http.StatusBadRequest, "git account has no access token")
		return gitClient{}, false
	}
	return newGitClient(provider, token), true
}

func (h *Handlers) upsertGitAccountFromOAuth(userID string, provider model.GitProvider, gitUser gitUserResponse, token *oauth2.Token) (model.GitAccount, error) {
	externalID := gitUser.externalID()
	username := gitUser.username()
	if externalID == "" || username == "" {
		return model.GitAccount{}, fmt.Errorf("git user identity is incomplete")
	}
	var account model.GitAccount
	err := h.db.First(&account, "user_id = ? and provider_id = ? and external_user_id = ?", userID, provider.ID, externalID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return account, err
	}
	if err == gorm.ErrRecordNotFound {
		account = model.GitAccount{
			ID:             id.New("gita"),
			UserID:         userID,
			ProviderID:     provider.ID,
			ExternalUserID: externalID,
		}
	}
	account.Username = username
	account.AvatarURL = strings.TrimSpace(gitUser.AvatarURL)
	account.AccessTokenRef = storedSecretRef(token.AccessToken)
	if token.RefreshToken != "" {
		account.RefreshTokenRef = storedSecretRef(token.RefreshToken)
	}
	account.Scopes = strings.Join(normalizeList(tokenScopes(token), false), ",")
	if !token.Expiry.IsZero() {
		account.ExpiresAt = &token.Expiry
	}
	account.Status = "connected"
	if err == gorm.ErrRecordNotFound {
		return account, h.db.Create(&account).Error
	}
	return account, h.db.Save(&account).Error
}

func tokenScopes(token *oauth2.Token) []string {
	scope, _ := token.Extra("scope").(string)
	if scope == "" {
		return nil
	}
	return strings.Fields(strings.ReplaceAll(scope, ",", " "))
}

func (h *Handlers) gitOAuthRedirectURL(ctx *gin.Context) string {
	return h.externalBaseURL(ctx) + "/api/v1/git/oauth/callback"
}

func (h *Handlers) gitWebhookURL(ctx *gin.Context, bindingID string) string {
	return h.externalBaseURL(ctx) + "/api/v1/git/webhooks/" + url.PathEscape(bindingID)
}

func (h *Handlers) externalBaseURL(ctx *gin.Context) string {
	proto := ctx.GetHeader("X-Forwarded-Proto")
	if proto == "" {
		if ctx.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := ctx.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = ctx.Request.Host
	}
	return proto + "://" + host
}

func verifyGitWebhookSignature(header http.Header, body []byte, secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	expected := hmacSHA256Hex(body, secret)
	for _, value := range []string{
		header.Get("X-Hub-Signature-256"),
		header.Get("X-Gitea-Signature"),
		header.Get("X-Gogs-Signature"),
	} {
		value = strings.TrimSpace(strings.TrimPrefix(value, "sha256="))
		if value != "" && hmac.Equal([]byte(value), []byte(expected)) {
			return true
		}
	}
	return false
}

func hmacSHA256Hex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func gitWebhookEvent(header http.Header) string {
	for _, key := range []string{"X-GitHub-Event", "X-Gitea-Event", "X-Gogs-Event"} {
		if value := strings.TrimSpace(header.Get(key)); value != "" {
			return value
		}
	}
	return "unknown"
}

func gitWebhookCommitSHA(body []byte) string {
	var payload struct {
		After string `json:"after"`
		SHA   string `json:"sha"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if strings.TrimSpace(payload.After) != "" {
		return strings.TrimSpace(payload.After)
	}
	return strings.TrimSpace(payload.SHA)
}

func positiveInt(value string, fallbackValue int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 1 {
		return fallbackValue
	}
	return parsed
}

func (h *Handlers) findEnabledGitProvider(ctx *gin.Context, providerID string) (model.GitProvider, bool) {
	var provider model.GitProvider
	if err := h.db.First(&provider, "id = ? and enabled = ?", strings.TrimSpace(providerID), true).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git provider not found")
		return provider, false
	}
	return provider, true
}

func (h *Handlers) findGitAccountForUser(ctx *gin.Context, userID, accountID string) (model.GitAccount, bool) {
	var account model.GitAccount
	if err := h.db.First(&account, "id = ? and user_id = ?", strings.TrimSpace(accountID), userID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git account not found")
		return account, false
	}
	if account.Status != "connected" {
		writeError(ctx, http.StatusBadRequest, "Git 账号未连接")
		return account, false
	}
	return account, true
}

func (h *Handlers) findApplicationByID(ctx *gin.Context, applicationID string) (model.Application, bool) {
	var app model.Application
	err := h.db.First(&app, "id = ? and project_id = ?", strings.TrimSpace(applicationID), ctx.Param("projectId")).Error
	if err != nil {
		writeError(ctx, http.StatusNotFound, "application not found")
		return app, false
	}
	return app, true
}

func (h *Handlers) syncApplicationRepositoryURL(binding model.RepositoryBinding) {
	_ = h.db.Model(&model.Application{}).
		Where("id = ? and project_id = ?", binding.ApplicationID, binding.ProjectID).
		Updates(map[string]any{
			"source_type":    "repository",
			"repository_url": binding.CloneURL,
		}).Error
}

func normalizeGitProviderType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gitea", "gitlab":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "github"
	}
}

func normalizeGitAuthType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "github-app", "pat":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "oauth"
	}
}

func normalizeGitAccountStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "expired", "revoked":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "connected"
	}
}

func normalizeWebhookStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "created", "disabled", "failed":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "pending"
	}
}

func defaultGitBaseURL(providerType string) string {
	switch providerType {
	case "github":
		return "https://github.com"
	default:
		return ""
	}
}

func defaultCloneURL(provider model.GitProvider, owner, repo string) string {
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultGitBaseURL(provider.Type)
	}
	if baseURL == "" {
		return ""
	}
	return baseURL + "/" + owner + "/" + repo + ".git"
}

type gitProviderInput struct {
	Type            string `json:"type"`
	Name            string `json:"name" binding:"required"`
	BaseURL         string `json:"baseUrl"`
	AuthType        string `json:"authType"`
	ClientID        string `json:"clientId"`
	ClientSecretRef string `json:"clientSecretRef"`
	Enabled         bool   `json:"enabled"`
}

type gitAccountInput struct {
	ProviderID      string   `json:"providerId" binding:"required"`
	ExternalUserID  string   `json:"externalUserId"`
	Username        string   `json:"username" binding:"required"`
	AvatarURL       string   `json:"avatarUrl"`
	AccessTokenRef  string   `json:"accessTokenRef"`
	RefreshTokenRef string   `json:"refreshTokenRef"`
	Scopes          []string `json:"scopes"`
	Status          string   `json:"status"`
}

type repositoryBindingInput struct {
	ApplicationID string `json:"applicationId" binding:"required"`
	GitAccountID  string `json:"gitAccountId" binding:"required"`
	Owner         string `json:"owner" binding:"required"`
	Repo          string `json:"repo" binding:"required"`
	CloneURL      string `json:"cloneUrl"`
	DefaultBranch string `json:"defaultBranch"`
	WebhookStatus string `json:"webhookStatus"`
	CredentialRef string `json:"credentialRef"`
}

type repositoryBindingResponse struct {
	model.RepositoryBinding
	ProviderName    string `json:"providerName"`
	ProviderType    string `json:"providerType"`
	AccountUsername string `json:"accountUsername"`
	ApplicationName string `json:"applicationName"`
}
