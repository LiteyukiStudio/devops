package api

import (
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/resourcename"
)

func runtimeProjectNamespace(project model.Project) string {
	return resourcename.ProjectNamespace(project.ID)
}

func deploymentTargetResourceName(target model.DeploymentTarget) string {
	return resourcename.DeploymentTarget(target.ID)
}

func shortResourceID(value string) string {
	return resourcename.ShortID(value)
}

func runtimeIDResourceName(prefix string, value string) string {
	return resourcename.FromID(prefix, value)
}

func runtimeShortID(value string) string {
	return resourcename.ShortID(value)
}

func runtimeDNSLabel(value string) string {
	return resourcename.DNSLabel(value)
}
