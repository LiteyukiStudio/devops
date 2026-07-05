package worker

import (
	"context"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/notification"
)

func (r *Runner) emitNotificationEvent(ctx context.Context, event notification.Event) {
	if r.db == nil {
		return
	}
	_, _ = (notification.Service{DB: r.db, Enqueuer: r.taskClient}).Emit(ctx, event)
}

func (r *Runner) emitBuildFailed(ctx context.Context, run model.BuildRun, message string) {
	project, application, target := r.notificationContext(run.ProjectID, run.ApplicationID, run.DeploymentTargetID)
	r.emitNotificationEvent(ctx, notification.Event{
		Type:             "build.failed",
		Severity:         notification.SeverityError,
		Project:          entityRef(project.ID, project.Name, project.Slug),
		Application:      entityRef(application.ID, application.Name, application.Slug),
		DeploymentTarget: entityRef(target.ID, target.Name, target.Stage),
		Build: notification.BuildContext{
			ID:      run.ID,
			Status:  "failed",
			Message: strings.TrimSpace(message),
			Image:   run.ImageRef,
			GitRef:  firstNonEmpty(run.SourceBranch, run.SourceTag, run.SourceCommit),
			GitSHA:  run.SourceCommit,
		},
		OccurredAt: time.Now(),
		Message:    firstNonEmpty(message, "Build failed"),
	})
}

func (r *Runner) emitReleaseFailed(ctx context.Context, release model.Release, message string) {
	project, application, target := r.notificationContext(release.ProjectID, release.ApplicationID, release.DeploymentTargetID)
	r.emitNotificationEvent(ctx, notification.Event{
		Type:             "release.failed",
		Severity:         notification.SeverityError,
		Project:          entityRef(project.ID, project.Name, project.Slug),
		Application:      entityRef(application.ID, application.Name, application.Slug),
		DeploymentTarget: entityRef(target.ID, target.Name, target.Stage),
		Release: notification.ReleaseContext{
			ID:       release.ID,
			Status:   "failed",
			Revision: release.Revision,
			ImageRef: release.ImageRef,
			Message:  strings.TrimSpace(message),
		},
		OccurredAt: time.Now(),
		Message:    firstNonEmpty(message, "Release failed"),
	})
}

func (r *Runner) emitHookFailed(ctx context.Context, run model.HookRun, message string) {
	project, application, target := r.notificationContext(run.ProjectID, run.ApplicationID, run.DeploymentTargetID)
	r.emitNotificationEvent(ctx, notification.Event{
		Type:             "hook.failed",
		Severity:         notification.SeverityError,
		Project:          entityRef(project.ID, project.Name, project.Slug),
		Application:      entityRef(application.ID, application.Name, application.Slug),
		DeploymentTarget: entityRef(target.ID, target.Name, target.Stage),
		Hook: notification.HookContext{
			ID:      run.ID,
			Name:    run.Name,
			Phase:   run.Phase,
			Status:  "failed",
			Message: strings.TrimSpace(message),
		},
		OccurredAt: time.Now(),
		Message:    firstNonEmpty(message, "Hook failed"),
	})
}

func (r *Runner) emitGatewayApplyFailed(ctx context.Context, route model.GatewayRoute, message string) {
	project, application, target := r.notificationContext(route.ProjectID, route.ApplicationID, route.DeploymentTargetID)
	r.emitNotificationEvent(ctx, notification.Event{
		Type:             "gateway.apply_failed",
		Severity:         notification.SeverityError,
		Project:          entityRef(project.ID, project.Name, project.Slug),
		Application:      entityRef(application.ID, application.Name, application.Slug),
		DeploymentTarget: entityRef(target.ID, target.Name, target.Stage),
		Gateway: notification.GatewayContext{
			ID:      route.ID,
			Domain:  route.Host,
			Path:    route.Path,
			Status:  "failed",
			Message: strings.TrimSpace(message),
		},
		OccurredAt: time.Now(),
		Message:    firstNonEmpty(message, "Gateway route apply failed"),
	})
}

func (r *Runner) notificationContext(projectID string, applicationID string, targetID string) (model.Project, model.Application, model.DeploymentTarget) {
	var project model.Project
	var application model.Application
	var target model.DeploymentTarget
	_ = r.db.First(&project, "id = ?", projectID).Error
	if strings.TrimSpace(applicationID) != "" {
		_ = r.db.First(&application, "id = ?", applicationID).Error
	}
	if strings.TrimSpace(targetID) != "" {
		_ = r.db.First(&target, "id = ?", targetID).Error
	}
	return project, application, target
}

func entityRef(id string, name string, slug string) notification.EntityRef {
	return notification.EntityRef{ID: id, Name: name, Slug: slug}
}
