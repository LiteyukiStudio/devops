package api

import (
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/variables"
	"net/url"
	"strings"
)

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
	namespace := strings.Trim(strings.TrimSpace(registry.Namespace), "/")
	if namespace == "" {
		namespace = projectSlug
	}
	if namespace != "" {
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
