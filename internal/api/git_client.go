package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"golang.org/x/oauth2"
)

const gitHTTPTimeout = 15 * time.Second

type gitClient struct {
	httpClient *http.Client
	provider   model.GitProvider
	token      string
}

type gitRepository struct {
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"`
	CloneURL      string `json:"cloneUrl"`
	DefaultBranch string `json:"defaultBranch"`
	Private       bool   `json:"private"`
}

type gitBranch struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

type gitFileContent struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Ref      string `json:"ref"`
	SHA      string `json:"sha"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type gitWebhookCreateResult struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Secret string `json:"-"`
}

func newGitClient(provider model.GitProvider, token string) gitClient {
	return gitClient{
		httpClient: &http.Client{Timeout: gitHTTPTimeout},
		provider:   provider,
		token:      strings.TrimSpace(token),
	}
}

func (c gitClient) listRepositories(ctx context.Context, search string, page, pageSize int) ([]gitRepository, error) {
	switch c.provider.Type {
	case "github":
		var repos []githubRepositoryResponse
		err := c.getJSON(ctx, c.apiURL("/user/repos", map[string]string{
			"affiliation": "owner,collaborator,organization_member",
			"sort":        "updated",
			"page":        strconv.Itoa(page),
			"per_page":    strconv.Itoa(pageSize),
		}), &repos)
		return filterGitRepositories(githubRepositories(repos), search), err
	case "gitea":
		var repos []giteaRepositoryResponse
		err := c.getJSON(ctx, c.apiURL("/user/repos", map[string]string{
			"page":  strconv.Itoa(page),
			"limit": strconv.Itoa(pageSize),
		}), &repos)
		return filterGitRepositories(giteaRepositories(repos), search), err
	default:
		return nil, fmt.Errorf("git provider type %q is not supported", c.provider.Type)
	}
}

func (c gitClient) listBranches(ctx context.Context, owner, repo string) ([]gitBranch, error) {
	var branches []gitBranchResponse
	err := c.getJSON(ctx, c.apiURL(fmt.Sprintf("/repos/%s/%s/branches", pathEscape(owner), pathEscape(repo)), nil), &branches)
	if err != nil {
		return nil, err
	}
	output := make([]gitBranch, 0, len(branches))
	for _, branch := range branches {
		output = append(output, gitBranch{Name: branch.Name, SHA: branch.Commit.SHA})
	}
	return output, nil
}

func (c gitClient) readFile(ctx context.Context, owner, repo, filePath, ref string) (gitFileContent, error) {
	filePath = strings.TrimLeft(strings.TrimSpace(filePath), "/")
	if filePath == "" {
		return gitFileContent{}, fmt.Errorf("file path is required")
	}
	params := map[string]string{}
	if strings.TrimSpace(ref) != "" {
		params["ref"] = strings.TrimSpace(ref)
	}
	var file gitContentResponse
	err := c.getJSON(ctx, c.apiURL(fmt.Sprintf("/repos/%s/%s/contents/%s", pathEscape(owner), pathEscape(repo), pathEscapePath(filePath)), params), &file)
	if err != nil {
		return gitFileContent{}, err
	}
	content := strings.TrimSpace(file.Content)
	if strings.EqualFold(file.Encoding, "base64") {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		if err != nil {
			return gitFileContent{}, err
		}
		content = string(decoded)
	}
	return gitFileContent{
		Path:     file.Path,
		Name:     file.Name,
		Ref:      strings.TrimSpace(ref),
		SHA:      file.SHA,
		Content:  content,
		Encoding: "utf-8",
	}, nil
}

func (c gitClient) createWebhook(ctx context.Context, owner, repo, callbackURL, secret string) (gitWebhookCreateResult, error) {
	switch c.provider.Type {
	case "github":
		payload := ginH{
			"name":   "web",
			"active": true,
			"events": []string{"push", "create"},
			"config": ginH{
				"url":          callbackURL,
				"content_type": "json",
				"secret":       secret,
				"insecure_ssl": "0",
			},
		}
		var response githubWebhookResponse
		if err := c.postJSON(ctx, c.apiURL(fmt.Sprintf("/repos/%s/%s/hooks", pathEscape(owner), pathEscape(repo)), nil), payload, &response); err != nil {
			return gitWebhookCreateResult{}, err
		}
		return gitWebhookCreateResult{ID: strconv.FormatInt(response.ID, 10), URL: response.Config.URL, Secret: secret}, nil
	case "gitea":
		payload := ginH{
			"type":   "gitea",
			"active": true,
			"events": []string{"push", "create"},
			"config": ginH{
				"url":          callbackURL,
				"content_type": "json",
				"secret":       secret,
			},
		}
		var response giteaWebhookResponse
		if err := c.postJSON(ctx, c.apiURL(fmt.Sprintf("/repos/%s/%s/hooks", pathEscape(owner), pathEscape(repo)), nil), payload, &response); err != nil {
			return gitWebhookCreateResult{}, err
		}
		return gitWebhookCreateResult{ID: strconv.FormatInt(response.ID, 10), URL: callbackURL, Secret: secret}, nil
	default:
		return gitWebhookCreateResult{}, fmt.Errorf("git provider type %q is not supported", c.provider.Type)
	}
}

func (c gitClient) getJSON(ctx context.Context, requestURL string, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeGitResponse(resp, output)
}

func (c gitClient) postJSON(ctx context.Context, requestURL string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeGitResponse(resp, output)
}

func (c gitClient) authorize(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if c.provider.Type == "github" {
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c gitClient) apiURL(apiPath string, params map[string]string) string {
	base := strings.TrimRight(c.provider.BaseURL, "/")
	switch c.provider.Type {
	case "github":
		if base == "" || base == "https://github.com" {
			base = "https://api.github.com"
		} else if !strings.Contains(base, "/api/") {
			base += "/api/v3"
		}
	case "gitea":
		base += "/api/v1"
	}
	parsed, _ := url.Parse(base + apiPath)
	query := parsed.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func gitOAuthConfig(provider model.GitProvider, redirectURL string) (*oauth2.Config, error) {
	endpoint, err := gitOAuthEndpoint(provider)
	if err != nil {
		return nil, err
	}
	scopes := []string{"repo", "read:user"}
	if provider.Type == "gitea" {
		scopes = []string{"read:repository", "write:repository", "read:user"}
	}
	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: resolveStoredSecretRef(provider.ClientSecretRef),
		Endpoint:     endpoint,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}, nil
}

func gitOAuthEndpoint(provider model.GitProvider) (oauth2.Endpoint, error) {
	base := strings.TrimRight(provider.BaseURL, "/")
	switch provider.Type {
	case "github":
		if base == "" {
			base = "https://github.com"
		}
		return oauth2.Endpoint{
			AuthURL:  base + "/login/oauth/authorize",
			TokenURL: base + "/login/oauth/access_token",
		}, nil
	case "gitea":
		return oauth2.Endpoint{
			AuthURL:  base + "/login/oauth/authorize",
			TokenURL: base + "/login/oauth/access_token",
		}, nil
	default:
		return oauth2.Endpoint{}, fmt.Errorf("git provider type %q is not supported", provider.Type)
	}
}

func (c gitClient) currentUser(ctx context.Context) (gitUserResponse, error) {
	switch c.provider.Type {
	case "github", "gitea":
		var user gitUserResponse
		err := c.getJSON(ctx, c.apiURL("/user", nil), &user)
		return user, err
	default:
		return gitUserResponse{}, fmt.Errorf("git provider type %q is not supported", c.provider.Type)
	}
}

func decodeGitResponse(resp *http.Response, output any) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("git api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if output == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, output)
}

func filterGitRepositories(repos []gitRepository, search string) []gitRepository {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return repos
	}
	filtered := make([]gitRepository, 0, len(repos))
	for _, repo := range repos {
		if strings.Contains(strings.ToLower(repo.FullName), search) || strings.Contains(strings.ToLower(repo.Name), search) {
			filtered = append(filtered, repo)
		}
	}
	return filtered
}

func githubRepositories(repos []githubRepositoryResponse) []gitRepository {
	output := make([]gitRepository, 0, len(repos))
	for _, repo := range repos {
		output = append(output, gitRepository{
			Owner:         repo.Owner.Login,
			Name:          repo.Name,
			FullName:      repo.FullName,
			CloneURL:      repo.CloneURL,
			DefaultBranch: repo.DefaultBranch,
			Private:       repo.Private,
		})
	}
	return output
}

func giteaRepositories(repos []giteaRepositoryResponse) []gitRepository {
	output := make([]gitRepository, 0, len(repos))
	for _, repo := range repos {
		output = append(output, gitRepository{
			Owner:         repo.Owner.UserName,
			Name:          repo.Name,
			FullName:      repo.FullName,
			CloneURL:      repo.CloneURL,
			DefaultBranch: repo.DefaultBranch,
			Private:       repo.Private,
		})
	}
	return output
}

func pathEscape(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

func pathEscapePath(value string) string {
	parts := strings.Split(strings.Trim(value, "/"), "/")
	for index, part := range parts {
		parts[index] = pathEscape(part)
	}
	return strings.Join(parts, "/")
}

type ginH map[string]any

type gitUserResponse struct {
	ID        any    `json:"id"`
	Login     string `json:"login"`
	UserName  string `json:"username"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (u gitUserResponse) externalID() string {
	return strings.TrimSpace(fmt.Sprint(u.ID))
}

func (u gitUserResponse) username() string {
	if strings.TrimSpace(u.Login) != "" {
		return strings.TrimSpace(u.Login)
	}
	if strings.TrimSpace(u.UserName) != "" {
		return strings.TrimSpace(u.UserName)
	}
	return strings.TrimSpace(u.Name)
}

type githubRepositoryResponse struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type giteaRepositoryResponse struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	Owner         struct {
		UserName string `json:"username"`
	} `json:"owner"`
}

type gitBranchResponse struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type gitContentResponse struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type githubWebhookResponse struct {
	ID     int64 `json:"id"`
	Config struct {
		URL string `json:"url"`
	} `json:"config"`
}

type giteaWebhookResponse struct {
	ID int64 `json:"id"`
}
