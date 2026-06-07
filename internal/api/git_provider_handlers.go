package api

import (
	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	gitprovider "github.com/LiteyukiStudio/devops/internal/provider/git"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"strings"
	"time"
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

	projectID := strings.TrimSpace(ctx.Query("projectId"))
	conditions := []string{"scope = 'global'", "(scope = 'user' and owner_ref = ?)"}
	args := []any{user.ID}
	if projectID != "" {
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return
		}
		conditions = append(conditions, "(scope = 'project' and owner_ref = ?)")
		args = append(args, projectID)
	} else if user.Role == "platform_admin" {
		conditions = append(conditions, "scope = 'project'")
	} else {
		projectIDs := h.projectIDsForUser(user.ID)
		if len(projectIDs) > 0 {
			conditions = append(conditions, "(scope = 'project' and owner_ref in ?)")
			args = append(args, projectIDs)
		}
	}
	query = query.Where(strings.Join(conditions, " or "), args...)

	var providers []model.GitProvider
	query = applySearch(ctx, query, "name", "base_url")
	if err := query.Find(&providers).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, h.gitProviderResponsesForUser(user, providers))
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
	baseURL := strings.TrimSpace(h.externalBaseURL(ctx))
	if baseURL == "" {
		writeError(ctx, http.StatusInternalServerError, "PUBLIC_BASE_URL is required")
		return
	}

	state := "git_" + randomHex(32)
	oauthState := model.GitOAuthState{
		ID:             id.New("gst"),
		StateHash:      hashToken(state),
		ProviderID:     provider.ID,
		UserID:         user.ID,
		RedirectPath:   sanitizeRedirectPath(ctx.DefaultQuery("redirect", "/projects")),
		FrontendOrigin: sanitizeFrontendOrigin(ctx.Query("frontendOrigin"), baseURL),
		ExpiresAt:      time.Now().Add(gitOAuthStateTTL),
	}
	if err := h.db.Create(&oauthState).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(user.ID, "git.oauth.start", provider.ID, true, oauthState.RedirectPath)

	oauthConfig, err := gitprovider.OAuthConfig(provider, baseURL+"/api/v1/git/oauth/callback", h.secrets.Resolve(provider.ClientSecretRef))
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.egressPolicyForUser(user).ValidateURL(oauthConfig.Endpoint.AuthURL); err != nil {
		writeError(ctx, http.StatusForbidden, "Git OAuth 授权地址不符合访问策略")
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
	baseURL := strings.TrimSpace(h.externalBaseURL(ctx))
	if baseURL == "" {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_callback_invalid")
		return
	}

	var oauthState model.GitOAuthState
	if err := h.db.First(&oauthState, "state_hash = ? and expires_at > ?", hashToken(plainState), time.Now()).Error; err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_state_invalid")
		return
	}
	_ = h.db.Delete(&oauthState).Error

	var stateUser model.User
	if err := h.db.First(&stateUser, "id = ? and disabled = ?", oauthState.UserID, false).Error; err != nil {
		h.audit(oauthState.UserID, "git.oauth.complete", oauthState.ProviderID, false, "user disabled or missing")
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_user_invalid")
		return
	}

	provider, ok := h.findEnabledGitProvider(ctx, oauthState.ProviderID)
	if !ok {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_provider_disabled")
		return
	}
	oauthConfig, err := gitprovider.OAuthConfig(provider, baseURL+"/api/v1/git/oauth/callback", h.secrets.Resolve(provider.ClientSecretRef))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_provider_invalid")
		return
	}
	egressCtx := h.egressContextForUser(ctx.Request.Context(), stateUser, 15*time.Second)
	token, err := oauthConfig.Exchange(egressCtx, code)
	if err != nil {
		h.audit(oauthState.UserID, "git.oauth.complete", provider.ID, false, "code exchange failed")
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_code_invalid")
		return
	}

	client := gitprovider.NewClientWithPolicy(provider, token.AccessToken, h.egressPolicyForUser(stateUser))
	gitUser, err := client.CurrentUser(egressCtx)
	if err != nil {
		h.audit(oauthState.UserID, "git.oauth.complete", provider.ID, false, "git user failed")
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_user_failed")
		return
	}
	account, err := h.upsertGitAccountFromOAuth(oauthState.UserID, provider, gitUser, token)
	if err != nil {
		h.audit(oauthState.UserID, "git.oauth.complete", provider.ID, false, "save failed")
		ctx.Redirect(http.StatusFound, "/login?error=git_oauth_save_failed")
		return
	}
	h.audit(stateUser.ID, "git.oauth.complete", provider.ID, true, account.ID)

	redirectTarget := buildFrontendRedirect(baseURL, oauthState.FrontendOrigin, oauthState.RedirectPath, account.ID)
	ctx.Redirect(http.StatusFound, redirectTarget)
}

func buildFrontendRedirect(defaultOrigin, frontendOrigin, path, accountID string) string {
	targetPath := sanitizeRedirectPath(path)
	targetOrigin := sanitizeFrontendOrigin(frontendOrigin, defaultOrigin)
	separator := "?"
	if strings.Contains(targetPath, "?") {
		separator = "&"
	}
	return targetOrigin + targetPath + separator + "gitAccountId=" + url.QueryEscape(accountID)
}

func sanitizeFrontendOrigin(raw, defaultOrigin string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return strings.TrimRight(defaultOrigin, "/")
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return strings.TrimRight(defaultOrigin, "/")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return strings.TrimRight(defaultOrigin, "/")
	}

	defaultParsed, defaultErr := url.Parse(defaultOrigin)
	if defaultErr != nil || defaultParsed.Hostname() == "" {
		return strings.TrimRight(parsed.Scheme+"//"+parsed.Host, "/")
	}

	candidateHost := strings.ToLower(parsed.Hostname())
	referenceHost := strings.ToLower(defaultParsed.Hostname())
	if candidateHost == referenceHost || isLoopbackPair(candidateHost, referenceHost) {
		return strings.TrimRight(parsed.Scheme+"://"+parsed.Host, "/")
	}
	return strings.TrimRight(defaultParsed.Scheme+"://"+defaultParsed.Host, "/")
}

func isLoopbackPair(left, right string) bool {
	return (left == "localhost" && right == "127.0.0.1") || (left == "127.0.0.1" && right == "localhost")
}

func (h *Handlers) CreateGitProvider(ctx *gin.Context) {
	if !h.requirePlatformAdmin(ctx) {
		return
	}

	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}

	var input gitProviderInput
	if !bindJSON(ctx, &input) {
		return
	}

	scope := normalizeGitScope(input.Scope)
	ownerRef := strings.TrimSpace(input.OwnerRef)
	providerType := normalizeGitProviderType(input.Type)
	if providerType == "github" {
		scope = "global"
		ownerRef = ""
		if err := h.requireSingleGitHubProvider(ctx, ""); err != nil {
			return
		}
	} else if !h.normalizeAndSetGitScopeOwner(ctx, user, scope, &ownerRef, nil) {
		return
	}

	provider := model.GitProvider{
		ID:       id.New("gitp"),
		Type:     providerType,
		Name:     strings.TrimSpace(input.Name),
		BaseURL:  normalizeGitBaseURL(providerType, input.BaseURL),
		Scope:    scope,
		OwnerRef: ownerRef,
		AuthType: normalizeGitAuthType(input.AuthType),
		ClientID: strings.TrimSpace(input.ClientID),
		Enabled:  input.Enabled,
	}
	if strings.TrimSpace(input.ClientSecret) != "" {
		provider.ClientSecretRef = h.secrets.Store(input.ClientSecret, user.ID, "git_provider:"+provider.ID)
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
	h.audit(user.ID, "git_provider.create", provider.ID, true, provider.Type)
	ctx.JSON(http.StatusCreated, gitProviderResponse(provider))
}

func (h *Handlers) UpdateGitProvider(ctx *gin.Context) {
	if !h.requirePlatformAdmin(ctx) {
		return
	}

	user, ok := h.currentUser(ctx)
	if !ok {
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

	providerType := normalizeGitProviderType(input.Type)
	scope := normalizeGitScope(input.Scope)
	ownerRef := strings.TrimSpace(input.OwnerRef)
	if providerType == "github" {
		scope = "global"
		ownerRef = ""
		if !h.providerIsSingleFor(ctx, provider.ID, providerType) {
			writeError(ctx, http.StatusBadRequest, "GitHub Provider 仅支持单个实例配置")
			return
		}
	} else if !h.normalizeAndSetGitScopeOwner(ctx, user, scope, &ownerRef, &provider) {
		return
	}

	provider.Type = providerType
	provider.Name = strings.TrimSpace(input.Name)
	provider.BaseURL = normalizeGitBaseURL(providerType, input.BaseURL)
	provider.Scope = scope
	provider.OwnerRef = ownerRef
	provider.AuthType = normalizeGitAuthType(input.AuthType)
	provider.ClientID = strings.TrimSpace(input.ClientID)
	if strings.TrimSpace(input.ClientSecret) != "" {
		provider.ClientSecretRef = h.secrets.Store(input.ClientSecret, user.ID, "git_provider:"+provider.ID)
	}
	provider.Enabled = input.Enabled
	if provider.Name == "" {
		writeError(ctx, http.StatusBadRequest, "请输入 Git Provider 名称")
		return
	}
	if err := h.db.Save(&provider).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(user.ID, "git_provider.update", provider.ID, true, provider.Type)
	ctx.JSON(http.StatusOK, gitProviderResponse(provider))
}

func (h *Handlers) DeleteGitProvider(ctx *gin.Context) {
	if !h.requirePlatformAdmin(ctx) {
		return
	}
	user, ok := h.currentUser(ctx)
	if !ok {
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
	h.audit(user.ID, "git_provider.delete", provider.ID, true, provider.Type)
	ctx.Status(http.StatusNoContent)
}
