package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/secret"
	"github.com/LiteyukiStudio/devops/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func (h *Handlers) gitOAuthRedirectURL(ctx *gin.Context) string {
	return h.externalBaseURL(ctx) + "/api/v1/git/oauth/callback"
}

func (h *Handlers) gitWebhookURL(ctx *gin.Context, bindingID string) string {
	return h.externalBaseURL(ctx) + "/api/v1/git/webhooks/" + url.PathEscape(bindingID)
}

func (h *Handlers) externalBaseURL(_ *gin.Context) string {
	if value := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"); value != "" {
		return value
	}
	return ""
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
	if !h.canUseGitProvider(ctx, provider) {
		return provider, false
	}
	return provider, true
}

func (h *Handlers) findGitAccountForUser(ctx *gin.Context, userID, accountID string) (model.GitAccount, bool) {
	var account model.GitAccount
	user, ok := h.currentUser(ctx)
	if !ok {
		return account, false
	}
	if user.ID != userID {
		writeError(ctx, http.StatusForbidden, "无权访问该 Git 账号")
		return account, false
	}
	if err := h.db.First(&account, "id = ?", strings.TrimSpace(accountID)).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "git account not found")
		return account, false
	}
	if !h.canUseGitAccount(ctx, user, account) {
		return account, false
	}
	provider, ok := h.findEnabledGitProvider(ctx, account.ProviderID)
	if !ok {
		return account, false
	}
	if account.AccessScope == "provider" {
		if !h.canUseGitProvider(ctx, provider) {
			return account, false
		}
	}

	if account.Status != "connected" {
		writeError(ctx, http.StatusBadRequest, "Git 账号未连接")
		return account, false
	}
	return account, true
}

func (h *Handlers) canUseGitProvider(ctx *gin.Context, provider model.GitProvider) bool {
	user, ok := h.currentUser(ctx)
	if !ok {
		return false
	}
	if service.CanUseGitProvider(user, provider, h.projects.UserHasProject) {
		return true
	}
	writeError(ctx, http.StatusForbidden, "无权访问该 Git Provider")
	return false
}

func (h *Handlers) canManageGitProvider(ctx *gin.Context, user model.User, provider model.GitProvider) bool {
	switch provider.Scope {
	case "global":
		if user.Role == "platform_admin" {
			return true
		}
	case "user":
		if provider.OwnerRef == user.ID {
			return true
		}
	case "project":
		if user.Role == "platform_admin" {
			return true
		}
		if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, provider.OwnerRef, "owner", "admin"); ok {
			return true
		}
	}
	writeError(ctx, http.StatusForbidden, "无权维护该 Git Provider")
	return false
}

func (h *Handlers) canUseGitAccount(ctx *gin.Context, user model.User, account model.GitAccount) bool {
	if service.CanUseGitAccount(user, account, h.projects.UserHasProject) {
		return true
	}
	writeError(ctx, http.StatusForbidden, "无权访问该 Git 凭据")
	return false
}

func (h *Handlers) canManageGitAccount(ctx *gin.Context, user model.User, account model.GitAccount) bool {
	switch normalizeGitAccessScope(account.AccessScope) {
	case "personal":
		if account.UserID == user.ID {
			return true
		}
		writeError(ctx, http.StatusForbidden, "无权维护该个人 Git 凭据")
		return false
	case "provider":
		provider, ok := h.findEnabledGitProvider(ctx, account.ProviderID)
		if !ok {
			return false
		}
		return h.canManageGitProvider(ctx, user, provider)
	default:
		writeError(ctx, http.StatusBadRequest, "无效的凭据范围")
		return false
	}
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
			"git_account_id": binding.GitAccountID,
			"repository_url": binding.CloneURL,
		}).Error
}

func writeGitUpstreamError(ctx *gin.Context, err error) {
	if err != nil {
		fmt.Printf("git upstream error: %v\n", err)
	}
	writeError(ctx, http.StatusBadGateway, "Git 上游接口调用失败，请检查凭据权限或稍后重试")
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

func normalizeGitScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "project":
		return "project"
	case "user":
		return "user"
	case "global":
		return "global"
	default:
		return "user"
	}
}

func (h *Handlers) normalizeAndSetGitScopeOwner(ctx *gin.Context, user model.User, scope string, ownerRef *string, _ *model.GitProvider) bool {
	switch scope {
	case "global":
		*ownerRef = ""
		return true
	case "user":
		*ownerRef = user.ID
		return true
	case "project":
		projectID := strings.TrimSpace(*ownerRef)
		if projectID == "" {
			writeError(ctx, http.StatusBadRequest, "project scope 需要选择所属项目")
			return false
		}
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return false
		}
		*ownerRef = projectID
		return true
	default:
		writeError(ctx, http.StatusBadRequest, "invalid scope")
		return false
	}
}

func (h *Handlers) normalizeAndSetGitScopeOwnerForAccount(ctx *gin.Context, user model.User, scope string, ownerRef *string, _ *model.GitAccount) bool {
	switch scope {
	case "global":
		*ownerRef = ""
		return true
	case "user":
		*ownerRef = user.ID
		return true
	case "project":
		projectID := strings.TrimSpace(*ownerRef)
		if projectID == "" {
			writeError(ctx, http.StatusBadRequest, "project scope 需要选择所属项目")
			return false
		}
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return false
		}
		*ownerRef = projectID
		return true
	default:
		writeError(ctx, http.StatusBadRequest, "invalid scope")
		return false
	}
}

func (h *Handlers) providerIsSingleFor(ctx *gin.Context, providerID, providerType string) bool {
	if providerType != "github" {
		return true
	}
	query := h.db.Model(&model.GitProvider{}).Where("type = ?", providerType)
	if providerID != "" {
		query = query.Where("id <> ?", providerID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return false
	}
	if count == 0 {
		return true
	}
	return false
}

func (h *Handlers) requireSingleGitHubProvider(ctx *gin.Context, providerID string) error {
	if h.providerIsSingleFor(ctx, providerID, "github") {
		return nil
	}
	writeError(ctx, http.StatusBadRequest, "GitHub Provider 仅支持一个实例")
	return fmt.Errorf("github provider already exists")
}

func normalizeGitAccountStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "expired", "revoked":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "connected"
	}
}

func normalizeGitAccessScope(value string) string {
	return service.NormalizeGitAccessScope(strings.ToLower(strings.TrimSpace(value)))
}

func gitProviderResponses(providers []model.GitProvider) []gin.H {
	responses := make([]gin.H, 0, len(providers))
	for _, provider := range providers {
		responses = append(responses, gitProviderResponse(provider))
	}
	return responses
}

func (h *Handlers) gitProviderResponsesForUser(user model.User, providers []model.GitProvider) []gin.H {
	responses := make([]gin.H, 0, len(providers))
	for _, provider := range providers {
		response := gitProviderResponse(provider)
		if !h.canInspectScopedResourceConfig(user, provider.Scope, provider.OwnerRef) {
			response["baseUrl"] = ""
			response["clientId"] = ""
		}
		responses = append(responses, response)
	}
	return responses
}

func gitProviderResponse(provider model.GitProvider) gin.H {
	return gin.H{
		"id":              provider.ID,
		"type":            provider.Type,
		"name":            provider.Name,
		"baseUrl":         provider.BaseURL,
		"scope":           provider.Scope,
		"ownerRef":        provider.OwnerRef,
		"authType":        provider.AuthType,
		"clientId":        provider.ClientID,
		"clientSecretSet": secret.HasValue(provider.ClientSecretRef),
		"enabled":         provider.Enabled,
		"createdAt":       provider.CreatedAt,
	}
}

func gitAccountResponses(accounts []model.GitAccount) []gin.H {
	responses := make([]gin.H, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, gitAccountResponse(account))
	}
	return responses
}

func gitAccountResponse(account model.GitAccount) gin.H {
	return gin.H{
		"id":              account.ID,
		"userId":          account.UserID,
		"scope":           account.Scope,
		"ownerRef":        account.OwnerRef,
		"providerId":      account.ProviderID,
		"externalUserId":  account.ExternalUserID,
		"username":        account.Username,
		"avatarUrl":       account.AvatarURL,
		"scopes":          account.Scopes,
		"accessScope":     normalizeGitAccessScope(account.AccessScope),
		"accessTokenSet":  secret.HasValue(account.AccessTokenRef),
		"refreshTokenSet": secret.HasValue(account.RefreshTokenRef),
		"expiresAt":       account.ExpiresAt,
		"status":          account.Status,
		"createdAt":       account.CreatedAt,
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

func normalizeGitBaseURL(providerType string, baseURL string) string {
	providerType = normalizeGitProviderType(providerType)
	if providerType == "github" {
		return defaultGitBaseURL("github")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return defaultGitBaseURL(providerType)
	}
	return baseURL
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
	Type         string `json:"type"`
	Name         string `json:"name" binding:"required"`
	BaseURL      string `json:"baseUrl"`
	Scope        string `json:"scope"`
	OwnerRef     string `json:"ownerRef"`
	AuthType     string `json:"authType"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Enabled      bool   `json:"enabled"`
}

type gitAccountInput struct {
	ProviderID     string   `json:"providerId" binding:"required"`
	Scope          string   `json:"scope"`
	OwnerRef       string   `json:"ownerRef"`
	ExternalUserID string   `json:"externalUserId"`
	Username       string   `json:"username" binding:"required"`
	AvatarURL      string   `json:"avatarUrl"`
	AccessToken    string   `json:"accessToken"`
	RefreshToken   string   `json:"refreshToken"`
	Scopes         []string `json:"scopes"`
	AccessScope    string   `json:"accessScope"`
	Status         string   `json:"status"`
}

type repositoryBindingInput struct {
	ApplicationID string `json:"applicationId" binding:"required"`
	GitAccountID  string `json:"gitAccountId" binding:"required"`
	Owner         string `json:"owner" binding:"required"`
	Repo          string `json:"repo" binding:"required"`
	CloneURL      string `json:"cloneUrl"`
	DefaultBranch string `json:"defaultBranch"`
	WebhookStatus string `json:"webhookStatus"`
}

type repositoryBindingResponse struct {
	model.RepositoryBinding
	ProviderName      string `json:"providerName"`
	ProviderType      string `json:"providerType"`
	AccountUsername   string `json:"accountUsername"`
	AccountOwnerEmail string `json:"accountOwnerEmail"`
	AccountOwnerName  string `json:"accountOwnerName"`
	ApplicationName   string `json:"applicationName"`
}
