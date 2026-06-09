package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	dnsprovider "github.com/LiteyukiStudio/devops/internal/provider/dns"
	gitprovider "github.com/LiteyukiStudio/devops/internal/provider/git"
	kubeprovider "github.com/LiteyukiStudio/devops/internal/provider/kubernetes"
	"github.com/LiteyukiStudio/devops/internal/secret"
	"github.com/LiteyukiStudio/devops/internal/tasks"
	"github.com/hibiken/asynq"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type Runner struct {
	db                          *gorm.DB
	secrets                     secret.Store
	deployRolloutTimeoutSeconds int64
	certManagerClusterIssuer    string
	dnsResolver                 dnsprovider.Resolver
	namespaceFactory            func(kubeconfig string) (kubeprovider.NamespaceManager, error)
}

const (
	hookPhasePreDeployment  = "preDeployment"
	hookPhasePostDeployment = "postDeployment"
)

type Options struct {
	DeployRolloutTimeoutSeconds int64
	CertManagerClusterIssuer    string
}

func Run(redisAddr string, db *gorm.DB, options Options) error {
	runner := NewRunner(db, options)
	scheduler, err := startScheduler(redisAddr)
	if err != nil {
		return err
	}
	defer scheduler.Shutdown()

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 4,
			Queues: map[string]int{
				tasks.QueueDeploy: 3,
				tasks.QueueLight:  1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeDeployRun, runner.withTaskEvents(runner.handleDeployRun))
	mux.HandleFunc(tasks.TypeGatewayApply, runner.withTaskEvents(runner.handleGatewayApply))
	mux.HandleFunc(tasks.TypeGitAccountRefresh, runner.withTaskEvents(runner.handleGitAccountRefresh))
	mux.HandleFunc(tasks.TypeSyncStatus, runner.withTaskEvents(runner.handleSyncStatus))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.syncBuilderAgentStatus(ctx)

	return server.Run(mux)
}

func (r *Runner) withTaskEvents(handler func(context.Context, *asynq.Task) error) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		envelope := taskEnvelopeFromPayload(task.Type(), task.Payload())
		_ = r.recordTaskEvent(envelope, "running", "")
		err := handler(ctx, task)
		if err != nil {
			_ = r.recordTaskEvent(envelope, "failed", err.Error())
			return err
		}
		_ = r.recordTaskEvent(envelope, "succeeded", "")
		return nil
	}
}

func taskEnvelopeFromPayload(taskType string, payload []byte) tasks.TaskEnvelope {
	var raw struct {
		Envelope tasks.TaskEnvelope `json:"envelope"`
	}
	_ = json.Unmarshal(payload, &raw)
	envelope := raw.Envelope
	if strings.TrimSpace(envelope.TaskType) == "" {
		envelope.TaskType = taskType
	}
	if strings.TrimSpace(envelope.TaskID) == "" {
		envelope.TaskID = taskType
	}
	if strings.TrimSpace(envelope.DedupeKey) == "" {
		envelope.DedupeKey = envelope.TaskID
	}
	if strings.TrimSpace(envelope.TraceID) == "" {
		envelope.TraceID = envelope.TaskID
	}
	return envelope
}

func (r *Runner) recordTaskEvent(envelope tasks.TaskEnvelope, status string, message string) error {
	if r.db == nil {
		return nil
	}
	return r.db.Create(&model.WorkerTaskEvent{
		ID:          id.New("tke"),
		TaskID:      envelope.TaskID,
		TaskType:    envelope.TaskType,
		DedupeKey:   envelope.DedupeKey,
		ActorID:     envelope.ActorID,
		ResourceRef: envelope.ResourceRef,
		Status:      status,
		Message:     message,
		Attempt:     envelope.Attempt,
	}).Error
}

func (r *Runner) syncBuilderAgentStatus(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := r.markStaleBuilderAgentsOffline(); err != nil {
			log.Printf("builder agent status sync failed: %v", err)
		}
		if err := r.markExpiredBuildJobsLost(); err != nil {
			log.Printf("build job lease sync failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Runner) markStaleBuilderAgentsOffline() error {
	if r.db == nil {
		return nil
	}
	staleBefore := time.Now().Add(-90 * time.Second)
	return r.db.Model(&model.BuilderAgent{}).
		Where("status = ? and (last_heartbeat_at is null or last_heartbeat_at < ?)", "online", staleBefore).
		Updates(map[string]any{
			"status":              "offline",
			"current_concurrency": 0,
		}).Error
}

func (r *Runner) markExpiredBuildJobsLost() error {
	if r.db == nil {
		return nil
	}
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		var jobs []model.BuildJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? and lease_until is not null and lease_until < ?", "running", now).
			Order("lease_until asc").
			Limit(50).
			Find(&jobs).Error; err != nil {
			return err
		}
		for _, job := range jobs {
			finishedAt := now
			if err := tx.Model(&model.BuildJob{}).
				Where("id = ? and status = ?", job.ID, "running").
				Updates(expiredBuildJobUpdates(finishedAt)).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.BuildRun{}).
				Where("id = ? and project_id = ? and status = ?", job.BuildRunID, job.ProjectID, "running").
				Updates(map[string]any{
					"status":      "lost",
					"finished_at": &finishedAt,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Runner) handleSyncStatus(ctx context.Context, task *asynq.Task) error {
	log.Printf("received task type=%s payload=%s", task.Type(), string(task.Payload()))
	return r.syncReleaseRuntimeStatus(ctx)
}

func (r *Runner) syncReleaseRuntimeStatus(ctx context.Context) error {
	if r.db == nil {
		return nil
	}
	var releases []model.Release
	if err := r.db.
		Where("status in ?", []string{"pending", "running", "succeeded"}).
		Order("created_at desc").
		Limit(200).
		Find(&releases).Error; err != nil {
		return err
	}
	for _, release := range releases {
		if err := r.syncReleaseRuntimeSnapshot(ctx, release); err != nil {
			log.Printf("release runtime status sync skipped release=%s: %v", release.ID, err)
		}
	}
	return nil
}

func (r *Runner) syncReleaseRuntimeSnapshot(ctx context.Context, release model.Release) error {
	var project model.Project
	if err := r.db.First(&project, "id = ?", release.ProjectID).Error; err != nil {
		return err
	}
	var application model.Application
	if err := r.db.First(&application, "id = ? and project_id = ?", release.ApplicationID, release.ProjectID).Error; err != nil {
		return err
	}
	var environment model.Environment
	if err := r.db.First(&environment, "id = ? and project_id = ?", release.EnvironmentID, release.ProjectID).Error; err != nil {
		return err
	}
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return err
	}
	deploymentTarget := r.releaseDeploymentTarget(release)
	namespace := deploymentNamespace(project, environment)
	resourceName := applicationResourceName(deploymentTarget)
	snapshot, err := manager.GetDeploymentSnapshot(ctx, namespace, resourceName)
	if err != nil {
		if isKubernetesNotFound(err) {
			message := fmt.Sprintf("deployment_missing: Kubernetes Deployment %s/%s not found", namespace, resourceName)
			return r.markReleaseRuntimeDrift(release, message)
		}
		return err
	}
	if snapshot.Phase == kubeprovider.DeploymentFailed {
		return r.markReleaseRuntimeDrift(release, firstNonEmpty(snapshot.Message, "Deployment runtime check failed"))
	}
	if release.Status == "pending" || release.Status == "running" {
		if snapshot.Phase == kubeprovider.DeploymentSucceeded {
			r.appendReleaseLog(release, firstNonEmpty(snapshot.Message, "Deployment rollout completed"))
			return r.finishDeployRelease(release, "succeeded", firstNonEmpty(snapshot.Message, "Deployment rollout completed"))
		}
		return r.db.Model(&model.Release{}).Where("id = ?", release.ID).Updates(map[string]any{
			"status":  "running",
			"message": firstNonEmpty(snapshot.Message, release.Message),
		}).Error
	}
	return nil
}

func (r *Runner) markReleaseRuntimeDrift(release model.Release, message string) error {
	finishedAt := time.Now()
	if err := r.db.Model(&model.Release{}).Where("id = ?", release.ID).Updates(releaseFinishUpdates("failed", message, finishedAt)).Error; err != nil {
		return err
	}
	r.appendReleaseLog(release, "运行态漂移: "+message)
	return nil
}

func expiredBuildJobUpdates(finishedAt time.Time) map[string]any {
	return map[string]any{
		"status":      "lost",
		"message":     "lease_expired",
		"lease_token": "",
		"lease_until": nil,
		"finished_at": &finishedAt,
	}
}

type periodicTaskSpec struct {
	Cron    string
	Task    *asynq.Task
	Queue   string
	Timeout time.Duration
}

func periodicTaskSpecs() ([]periodicTaskSpec, error) {
	gitRefreshTask, err := tasks.NewGitAccountRefreshTask(tasks.GitAccountRefreshPayload{ActorID: "system"})
	if err != nil {
		return nil, err
	}
	return []periodicTaskSpec{
		{Cron: "@every 5m", Task: gitRefreshTask, Queue: tasks.QueueLight, Timeout: 10 * time.Minute},
		{Cron: "@every 1m", Task: asynq.NewTask(tasks.TypeSyncStatus, []byte("{}")), Queue: tasks.QueueLight, Timeout: 5 * time.Minute},
	}, nil
}

func startScheduler(redisAddr string) (*asynq.Scheduler, error) {
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: redisAddr}, &asynq.SchedulerOpts{})
	specs, err := periodicTaskSpecs()
	if err != nil {
		return nil, err
	}
	for _, spec := range specs {
		if _, err := scheduler.Register(spec.Cron, spec.Task, asynq.Queue(spec.Queue), asynq.Timeout(spec.Timeout)); err != nil {
			return nil, err
		}
	}
	go func() {
		if err := scheduler.Run(); err != nil {
			log.Printf("run scheduler: %v", err)
		}
	}()
	return scheduler, nil
}

func (r *Runner) handleGitAccountRefresh(ctx context.Context, task *asynq.Task) error {
	var payload tasks.GitAccountRefreshPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	accounts, err := r.gitAccountsDueForRefresh(time.Now())
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if err := r.refreshGitAccount(ctx, account); err != nil {
			log.Printf("refresh git account %s: %v", account.ID, err)
		}
	}
	return nil
}

func (r *Runner) handleGatewayApply(ctx context.Context, task *asynq.Task) error {
	var payload tasks.GatewayApplyPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	var route model.GatewayRoute
	if err := r.db.First(&route, "id = ? and project_id = ?", payload.GatewayRouteID, payload.ProjectID).Error; err != nil {
		return err
	}
	var project model.Project
	if err := r.db.First(&project, "id = ?", payload.ProjectID).Error; err != nil {
		return err
	}
	var application model.Application
	if err := r.db.First(&application, "id = ? and project_id = ?", route.ApplicationID, payload.ProjectID).Error; err != nil {
		return err
	}
	var environment model.Environment
	if err := r.db.First(&environment, "id = ? and project_id = ?", route.EnvironmentID, payload.ProjectID).Error; err != nil {
		return err
	}

	namespace := deploymentNamespace(project, environment)
	if err := r.ensureProjectNamespace(ctx, namespace, project, environment); err != nil {
		_ = r.db.Model(&route).Updates(map[string]any{"status": "failed"}).Error
		return err
	}
	if err := r.applyGatewayIngress(ctx, route, project, application, environment, namespace); err != nil {
		_ = r.db.Model(&route).Updates(map[string]any{"status": "failed"}).Error
		return err
	}
	certificateStatus, err := r.applyGatewayCertificate(ctx, route, project, namespace)
	if err != nil {
		_ = r.db.Model(&route).Updates(map[string]any{"status": "failed", "certificate_status": "failed"}).Error
		return err
	}
	updates := map[string]any{"status": "active", "dns_status": r.gatewayDNSStatus(ctx, route)}
	if certificateStatus != "" {
		updates["certificate_status"] = certificateStatus
	}
	return r.db.Model(&route).Updates(updates).Error
}

func (r *Runner) handleDeployRun(ctx context.Context, task *asynq.Task) error {
	var payload tasks.DeployRunPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	var release model.Release
	if err := r.db.First(&release, "id = ? and project_id = ?", payload.ReleaseID, payload.ProjectID).Error; err != nil {
		return err
	}
	var project model.Project
	if err := r.db.First(&project, "id = ?", payload.ProjectID).Error; err != nil {
		return err
	}
	var application model.Application
	if err := r.db.First(&application, "id = ? and project_id = ?", release.ApplicationID, payload.ProjectID).Error; err != nil {
		return err
	}
	var environment model.Environment
	if err := r.db.First(&environment, "id = ? and project_id = ?", release.EnvironmentID, payload.ProjectID).Error; err != nil {
		return err
	}
	deploymentTarget := r.releaseDeploymentTarget(release)

	now := time.Now()
	if release.StartedAt == nil {
		if err := r.db.Model(&release).Updates(map[string]any{"status": "running", "started_at": &now}).Error; err != nil {
			return err
		}
	}
	r.appendReleaseLog(release, fmt.Sprintf("开始部署 release=%s application=%s environment=%s image=%s", release.ID, application.Slug, environment.Slug, release.ImageRef))

	namespace := deploymentNamespace(project, environment)
	r.appendReleaseLog(release, fmt.Sprintf("确保命名空间 %s 存在", namespace))
	if err := r.ensureProjectNamespace(ctx, namespace, project, environment); err != nil {
		finishedAt := time.Now()
		_ = r.db.Model(&release).Updates(map[string]any{"status": "failed", "message": err.Error(), "finished_at": &finishedAt}).Error
		r.appendReleaseLog(release, "命名空间准备失败: "+err.Error())
		return err
	}
	r.appendReleaseLog(release, "下发 ConfigMap/Secret")
	if err := r.applyApplicationRuntimeConfig(ctx, release, project, application, environment, deploymentTarget, namespace); err != nil {
		finishedAt := time.Now()
		_ = r.db.Model(&release).Updates(map[string]any{"status": "failed", "message": err.Error(), "finished_at": &finishedAt}).Error
		r.appendReleaseLog(release, "运行配置下发失败: "+err.Error())
		return err
	}
	if err := r.runDeploymentHooks(ctx, hookPhasePreDeployment, release, project, application, environment, deploymentTarget, namespace); err != nil {
		finishedAt := time.Now()
		_ = r.db.Model(&release).Updates(map[string]any{"status": "failed", "message": err.Error(), "finished_at": &finishedAt}).Error
		r.appendReleaseLog(release, "preDeployment Hook 失败: "+err.Error())
		return err
	}
	r.appendReleaseLog(release, "下发 Deployment/Service/ConfigMap/Secret")
	if err := r.applyApplicationResources(ctx, release, project, application, environment, deploymentTarget, namespace); err != nil {
		finishedAt := time.Now()
		_ = r.db.Model(&release).Updates(map[string]any{"status": "failed", "message": err.Error(), "finished_at": &finishedAt}).Error
		r.appendReleaseLog(release, "资源下发失败: "+err.Error())
		return err
	}
	if err := r.db.Model(&release).Updates(map[string]any{
		"status":  "running",
		"message": fmt.Sprintf("Deployment/Service/ConfigMap/Secret 已下发到命名空间 %s", namespace),
	}).Error; err != nil {
		return err
	}
	r.appendReleaseLog(release, "等待 Deployment rollout 完成")
	message, err := r.waitForDeploymentRollout(ctx, release, application, environment, deploymentTarget, namespace)
	if err != nil {
		finishedAt := time.Now()
		_ = r.db.Model(&release).Updates(releaseFinishUpdates("failed", err.Error(), finishedAt)).Error
		r.appendReleaseLog(release, "部署失败: "+err.Error())
		return err
	}
	r.appendReleaseLog(release, firstNonEmpty(message, "Deployment rollout completed"))
	if err := r.runDeploymentHooks(ctx, hookPhasePostDeployment, release, project, application, environment, deploymentTarget, namespace); err != nil {
		finishedAt := time.Now()
		_ = r.db.Model(&release).Updates(releaseFinishUpdates("failed", err.Error(), finishedAt)).Error
		r.appendReleaseLog(release, "postDeployment Hook 失败: "+err.Error())
		return err
	}
	return r.finishDeployRelease(release, "succeeded", firstNonEmpty(message, "Deployment rollout completed"))
}

func NewRunner(db *gorm.DB, options Options) *Runner {
	deployRolloutTimeoutSeconds := options.DeployRolloutTimeoutSeconds
	if deployRolloutTimeoutSeconds <= 0 {
		deployRolloutTimeoutSeconds = 600
	}
	certManagerClusterIssuer := strings.TrimSpace(options.CertManagerClusterIssuer)
	if certManagerClusterIssuer == "" {
		certManagerClusterIssuer = "letsencrypt-http01"
	}
	return &Runner{
		db:                          db,
		secrets:                     secret.NewStore(db, nil),
		deployRolloutTimeoutSeconds: deployRolloutTimeoutSeconds,
		certManagerClusterIssuer:    certManagerClusterIssuer,
		dnsResolver:                 dnsprovider.NewNetResolver(),
		namespaceFactory: func(kubeconfig string) (kubeprovider.NamespaceManager, error) {
			return kubeprovider.NewClientFromKubeconfig(kubeconfig)
		},
	}
}

func (r *Runner) gitAccountsDueForRefresh(now time.Time) ([]model.GitAccount, error) {
	var accounts []model.GitAccount
	err := r.db.Where("status = ? and refresh_token_ref <> '' and expires_at is not null and expires_at <= ?", "connected", now.Add(5*time.Minute)).
		Find(&accounts).Error
	return accounts, err
}

func gitAccountDueForWorkerRefresh(account model.GitAccount, now time.Time) bool {
	return account.Status == "connected" &&
		strings.TrimSpace(account.RefreshTokenRef) != "" &&
		account.ExpiresAt != nil &&
		!account.ExpiresAt.After(now.Add(5*time.Minute))
}

func (r *Runner) refreshGitAccount(ctx context.Context, account model.GitAccount) error {
	var provider model.GitProvider
	if err := r.db.First(&provider, "id = ? and enabled = ?", account.ProviderID, true).Error; err != nil {
		return err
	}
	refreshToken := r.secrets.Resolve(account.RefreshTokenRef)
	if strings.TrimSpace(refreshToken) == "" {
		return r.expireGitAccount(account, "git account has no refresh token")
	}
	oauthConfig, err := gitprovider.OAuthConfig(provider, "", r.secrets.Resolve(provider.ClientSecretRef))
	if err != nil {
		return r.expireGitAccount(account, "git OAuth provider configuration is invalid")
	}
	tokenSource := oauthConfig.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Minute),
	})
	token, err := tokenSource.Token()
	if err != nil {
		return r.expireGitAccount(account, "git token refresh failed")
	}
	account.AccessTokenRef = r.secrets.Store(token.AccessToken, account.UserID, "git_account:"+account.ID+":access")
	if token.RefreshToken != "" {
		account.RefreshTokenRef = r.secrets.Store(token.RefreshToken, account.UserID, "git_account:"+account.ID+":refresh")
	}
	if !token.Expiry.IsZero() {
		account.ExpiresAt = &token.Expiry
	}
	account.Status = "connected"
	if err := r.db.Save(&account).Error; err != nil {
		return err
	}
	return r.auditGitAccountRefresh(account, true, account.Username)
}

func (r *Runner) expireGitAccount(account model.GitAccount, message string) error {
	account.Status = "expired"
	if err := r.db.Save(&account).Error; err != nil {
		return err
	}
	return r.auditGitAccountRefresh(account, false, message)
}

func (r *Runner) auditGitAccountRefresh(account model.GitAccount, success bool, message string) error {
	entry := model.AuditLog{
		ID:       id.New("aud"),
		UserID:   account.UserID,
		Action:   "git_account.refresh",
		Resource: account.ID,
		Success:  success,
		Message:  message,
	}
	return r.db.Create(&entry).Error
}

func (r *Runner) ensureProjectNamespace(ctx context.Context, namespace string, project model.Project, environment model.Environment) error {
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return err
	}
	return manager.EnsureNamespace(ctx, namespace, kubeprovider.ProjectNamespaceLabels(project.ID))
}

func (r *Runner) applyGatewayIngress(ctx context.Context, route model.GatewayRoute, project model.Project, application model.Application, environment model.Environment, namespace string) error {
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return err
	}
	return manager.ApplyGatewayIngress(ctx, gatewayIngressSpec(route, project, application, environment, namespace, r.gatewayServiceName(route, application, environment)))
}

func (r *Runner) gatewayServiceName(route model.GatewayRoute, application model.Application, environment model.Environment) string {
	var target model.DeploymentTarget
	err := r.db.Where("project_id = ? and application_id = ? and environment_id = ? and enabled = ?", route.ProjectID, application.ID, environment.ID, true).
		Order("created_at asc").
		First(&target).Error
	if err == nil {
		return applicationResourceName(target)
	}
	return dnsLabel(application.Slug)
}

func (r *Runner) gatewayDNSStatus(ctx context.Context, route model.GatewayRoute) string {
	if err := dnsprovider.CheckCNAME(ctx, r.dnsResolver, route.Host, route.CNAMETarget); err != nil {
		return "failed"
	}
	return "verified"
}

func (r *Runner) applyGatewayCertificate(ctx context.Context, route model.GatewayRoute, project model.Project, namespace string) (string, error) {
	if strings.TrimSpace(route.TLSMode) != "http-challenge" {
		return "", nil
	}
	manager, err := r.kubernetesManager(model.Environment{})
	if err != nil {
		return "", err
	}
	spec := gatewayCertificateSpec(route, project, namespace, r.certManagerClusterIssuer)
	if err := manager.ApplyCertificate(ctx, spec); err != nil {
		return "", err
	}
	snapshot, err := manager.GetCertificateSnapshot(ctx, spec.Namespace, spec.Name)
	if err != nil {
		return "", err
	}
	return snapshot.Phase, nil
}

func (r *Runner) applyApplicationResources(ctx context.Context, release model.Release, project model.Project, application model.Application, environment model.Environment, deploymentTarget model.DeploymentTarget, namespace string) error {
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return err
	}
	spec, err := applicationResourcesSpec(release, project, application, environment, deploymentTarget, namespace, r.deployRolloutTimeoutSeconds)
	if err != nil {
		return err
	}
	return manager.ApplyApplicationResources(ctx, spec)
}

func (r *Runner) applyApplicationRuntimeConfig(ctx context.Context, release model.Release, project model.Project, application model.Application, environment model.Environment, deploymentTarget model.DeploymentTarget, namespace string) error {
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return err
	}
	spec, err := applicationResourcesSpec(release, project, application, environment, deploymentTarget, namespace, r.deployRolloutTimeoutSeconds)
	if err != nil {
		return err
	}
	return manager.ApplyApplicationRuntimeConfig(ctx, spec)
}

func (r *Runner) waitForDeploymentRollout(ctx context.Context, release model.Release, application model.Application, environment model.Environment, deploymentTarget model.DeploymentTarget, namespace string) (string, error) {
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return "", err
	}
	resourceName := applicationResourceName(deploymentTarget)
	timeout := time.Duration(r.deployRolloutTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	rolloutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		snapshot, err := manager.GetDeploymentSnapshot(rolloutCtx, namespace, resourceName)
		if err != nil {
			return "", err
		}
		if snapshot.Message != "" {
			_ = r.db.Model(&model.Release{}).Where("id = ?", release.ID).Update("message", snapshot.Message).Error
			r.appendReleaseLog(release, snapshot.Message)
		}

		switch snapshot.Phase {
		case kubeprovider.DeploymentSucceeded:
			return snapshot.Message, nil
		case kubeprovider.DeploymentFailed:
			return "", errors.New(firstNonEmpty(snapshot.Message, "Deployment rollout failed"))
		}

		select {
		case <-rolloutCtx.Done():
			return "", fmt.Errorf("Deployment rollout timed out after %s", timeout)
		case <-ticker.C:
		}
	}
}

func (r *Runner) runDeploymentHooks(ctx context.Context, phase string, release model.Release, project model.Project, application model.Application, environment model.Environment, deploymentTarget model.DeploymentTarget, namespace string) error {
	var configs []model.ProjectHookConfig
	if err := r.db.Where("project_id = ? and phase = ?", project.ID, phase).
		Order("run_order asc, created_at asc").
		Find(&configs).Error; err != nil {
		return err
	}
	if len(configs) == 0 {
		return nil
	}
	manager, err := r.kubernetesManager(environment)
	if err != nil {
		return err
	}
	resourceName := applicationResourceName(deploymentTarget)
	buildContext := r.releaseBuildContext(release)
	for _, config := range configs {
		hookRun := model.HookRun{
			ID:                 id.New("hrun"),
			ProjectID:          project.ID,
			HookConfigID:       config.ID,
			ReleaseID:          release.ID,
			ApplicationID:      application.ID,
			ModuleID:           release.ModuleID,
			EnvironmentID:      environment.ID,
			DeploymentTargetID: deploymentTarget.ID,
			Name:               config.Name,
			Phase:              config.Phase,
			Status:             "running",
			ScriptSnapshot:     config.Script,
			Shell:              config.Shell,
			ImageRef:           release.ImageRef,
			TimeoutSeconds:     config.TimeoutSeconds,
			FailurePolicy:      config.FailurePolicy,
			StartedAt:          timePtr(time.Now()),
		}
		if err := r.db.Create(&hookRun).Error; err != nil {
			return err
		}
		r.appendReleaseLog(release, fmt.Sprintf("执行 %s Hook: %s", phase, config.Name))
		result, err := manager.RunHookJob(ctx, kubeprovider.HookJobSpec{
			Name:               hookJobName(hookRun),
			Namespace:          namespace,
			ProjectID:          project.ID,
			ApplicationID:      application.ID,
			ModuleID:           release.ModuleID,
			BuildRunID:         release.BuildRunID,
			EnvironmentID:      environment.ID,
			DeploymentTargetID: deploymentTarget.ID,
			ReleaseID:          release.ID,
			HookRunID:          hookRun.ID,
			Phase:              phase,
			Image:              release.ImageRef,
			GitBranch:          buildContext.GitBranch,
			GitTag:             buildContext.GitTag,
			GitRefName:         buildContext.GitRefName,
			GitRefType:         buildContext.GitRefType,
			GitRef:             buildContext.GitRef,
			GitSHA:             buildContext.GitSHA,
			GitShortSHA:        buildContext.GitShortSHA,
			Shell:              config.Shell,
			Script:             config.Script,
			TimeoutSeconds:     int32(normalizePositive(config.TimeoutSeconds, 300)),
			ConfigMapName:      resourceName + "-config",
			SecretName:         resourceName + "-secret",
		})
		if err != nil {
			result = kubeprovider.HookJobResult{Succeeded: false, ExitCode: 1, Message: err.Error()}
		}
		r.appendHookRunLog(hookRun, result.Logs)
		status := "succeeded"
		if !result.Succeeded {
			status = "failed"
		}
		finishedAt := time.Now()
		if updateErr := r.db.Model(&model.HookRun{}).Where("id = ?", hookRun.ID).Updates(map[string]any{
			"status":      status,
			"exit_code":   result.ExitCode,
			"message":     result.Message,
			"finished_at": &finishedAt,
		}).Error; updateErr != nil {
			return updateErr
		}
		if result.Logs != "" {
			r.appendReleaseLog(release, result.Logs)
		}
		if !result.Succeeded && config.FailurePolicy != "ignore" {
			return errors.New(firstNonEmpty(result.Message, phase+" hook failed"))
		}
	}
	return nil
}

func (r *Runner) appendReleaseLog(release model.Release, content string) {
	if r.db == nil {
		return
	}
	content = trimReleaseLogContent(content)
	if content == "" {
		return
	}
	var existing model.ReleaseLog
	err := r.db.First(&existing, "release_id = ? and project_id = ?", release.ID, release.ProjectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = r.db.Create(&model.ReleaseLog{
			ID:        id.New("rlog"),
			ReleaseID: release.ID,
			ProjectID: release.ProjectID,
			Content:   content,
		}).Error
		return
	}
	if err != nil {
		return
	}
	existing.Content = trimReleaseLogContent(existing.Content + "\n" + content)
	_ = r.db.Save(&existing).Error
}

func (r *Runner) appendHookRunLog(run model.HookRun, content string) {
	if r.db == nil {
		return
	}
	content = trimReleaseLogContent(content)
	if content == "" {
		return
	}
	var existing model.HookRunLog
	err := r.db.First(&existing, "hook_run_id = ? and project_id = ?", run.ID, run.ProjectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		_ = r.db.Create(&model.HookRunLog{
			ID:        id.New("hlog"),
			HookRunID: run.ID,
			ProjectID: run.ProjectID,
			Content:   content,
		}).Error
		return
	}
	if err != nil {
		return
	}
	existing.Content = trimReleaseLogContent(existing.Content + "\n" + content)
	_ = r.db.Save(&existing).Error
}

func (r *Runner) releaseBuildContext(release model.Release) deploymentHookBuildContext {
	var run model.BuildRun
	if strings.TrimSpace(release.BuildRunID) == "" || r.db.First(&run, "id = ? and project_id = ?", release.BuildRunID, release.ProjectID).Error != nil {
		return deploymentHookBuildContext{}
	}
	refName := firstNonEmpty(run.SourceTag, run.SourceBranch)
	refType := "branch"
	refValue := ""
	if strings.TrimSpace(run.SourceTag) != "" {
		refType = "tag"
		refValue = "refs/tags/" + strings.TrimSpace(run.SourceTag)
	} else if strings.TrimSpace(run.SourceBranch) != "" {
		refValue = "refs/heads/" + strings.TrimSpace(run.SourceBranch)
	}
	return deploymentHookBuildContext{
		GitBranch:   run.SourceBranch,
		GitTag:      run.SourceTag,
		GitRefName:  refName,
		GitRefType:  refType,
		GitRef:      refValue,
		GitSHA:      run.SourceCommit,
		GitShortSHA: shortCommit(run.SourceCommit),
	}
}

type deploymentHookBuildContext struct {
	GitBranch   string
	GitTag      string
	GitRefName  string
	GitRefType  string
	GitRef      string
	GitSHA      string
	GitShortSHA string
}

func trimReleaseLogContent(content string) string {
	content = strings.TrimSpace(content)
	const maxLogBytes = 1024 * 1024
	if len(content) <= maxLogBytes {
		return content
	}
	return content[len(content)-maxLogBytes:]
}

func (r *Runner) finishDeployRelease(release model.Release, status string, message string) error {
	finishedAt := time.Now()
	return r.db.Model(&model.Release{}).Where("id = ?", release.ID).Updates(releaseFinishUpdates(status, message, finishedAt)).Error
}

func releaseFinishUpdates(status string, message string, finishedAt time.Time) map[string]any {
	return map[string]any{
		"status":      status,
		"message":     firstNonEmpty(message, "Deployment "+status),
		"finished_at": &finishedAt,
	}
}

func (r *Runner) kubernetesManager(environment model.Environment) (kubeprovider.NamespaceManager, error) {
	var cluster model.RuntimeCluster
	if clusterID := strings.TrimSpace(environment.ClusterID); clusterID != "" {
		query, args := environmentClusterLookup(clusterID)
		err := r.db.First(&cluster, append([]any{query}, args...)...).Error
		if err != nil {
			return nil, fmt.Errorf("runtime cluster %s not found: %w", clusterID, err)
		}
	} else {
		err := r.db.Where("scope = ? and is_default = ? and type in ?", "global", true, []string{"kubernetes", "k3s"}).First(&cluster).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = r.db.Where("scope = ? and type in ?", "global", []string{"kubernetes", "k3s"}).Order("created_at asc").First(&cluster).Error
		}
		if err != nil {
			return nil, fmt.Errorf("runtime cluster not found: %w", err)
		}
	}

	kubeconfig := r.secrets.Resolve(cluster.KubeconfigRef)
	if strings.TrimSpace(kubeconfig) == "" {
		return nil, errors.New("runtime cluster kubeconfig is missing")
	}

	manager, err := r.namespaceFactory(kubeconfig)
	if err != nil {
		return nil, runtimeClusterKubeconfigError(err)
	}
	return manager, nil
}

func runtimeClusterKubeconfigError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "unable to read client-cert") ||
		strings.Contains(message, "unable to read client-key") ||
		strings.Contains(message, "unable to read certificate-authority") {
		return fmt.Errorf("运行集群 kubeconfig 引用了当前 Worker 无法读取的本地证书文件，请在集群页面重新保存已内联证书的 kubeconfig 后再部署: %w", err)
	}
	return fmt.Errorf("运行集群 kubeconfig 无效，无法创建 Kubernetes 客户端: %w", err)
}

func isKubernetesNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}

func environmentClusterLookup(clusterID string) (string, []any) {
	return "id = ? and type in ?", []any{strings.TrimSpace(clusterID), []string{"kubernetes", "k3s"}}
}

func projectNamespace(project model.Project) string {
	return idResourceName("ns", project.ID)
}

func deploymentNamespace(project model.Project, _ model.Environment) string {
	return projectNamespace(project)
}

func (r *Runner) releaseDeploymentTarget(release model.Release) model.DeploymentTarget {
	var target model.DeploymentTarget
	if strings.TrimSpace(release.ModuleID) != "" {
		if err := r.db.First(&target, "project_id = ? and application_id = ? and environment_id = ? and module_id = ?", release.ProjectID, release.ApplicationID, release.EnvironmentID, release.ModuleID).Error; err == nil {
			return target
		}
	}
	return model.DeploymentTarget{}
}

func applicationResourceName(deploymentTarget model.DeploymentTarget) string {
	return idResourceName("dplt", deploymentTarget.ID)
}

func hookJobName(run model.HookRun) string {
	return idResourceName("hook", run.ID)
}

func normalizePositive(value int, fallbackValue int) int {
	if value > 0 {
		return value
	}
	return fallbackValue
}

func shortCommit(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func idResourceName(prefix string, value string) string {
	suffix := shortID(value)
	if suffix == "" {
		return dnsLabel(prefix)
	}
	return dnsLabel(prefix + "-" + suffix)
}

func shortID(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "_"); index >= 0 {
		value = value[index+1:]
	}
	value = dnsLabelOptionalSegment(value)
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func gatewayIngressName(route model.GatewayRoute) string {
	return buildResourceName(route.ID, "liteyuki-gateway-")
}

func gatewayTLSSecretName(route model.GatewayRoute) string {
	if strings.TrimSpace(route.TLSMode) == "http-only" {
		return ""
	}
	return dnsLabel("tls-" + route.Host)
}

func gatewayIngressSpec(route model.GatewayRoute, project model.Project, application model.Application, environment model.Environment, namespace string, serviceName string) kubeprovider.GatewayIngressSpec {
	servicePort := route.ServicePort
	if servicePort <= 0 {
		servicePort = application.ServicePort
	}
	if servicePort <= 0 {
		servicePort = 80
	}
	return kubeprovider.GatewayIngressSpec{
		Name:          gatewayIngressName(route),
		Namespace:     namespace,
		ProjectID:     project.ID,
		ApplicationID: application.ID,
		EnvironmentID: environment.ID,
		RouteID:       route.ID,
		Host:          strings.TrimSpace(route.Host),
		Path:          route.Path,
		ServiceName:   firstNonEmpty(serviceName, dnsLabel(application.Slug)),
		ServicePort:   int32(servicePort),
		TLSSecretName: gatewayTLSSecretName(route),
	}
}

func gatewayCertificateSpec(route model.GatewayRoute, project model.Project, namespace string, clusterIssuer string) kubeprovider.CertificateSpec {
	return kubeprovider.CertificateSpec{
		Name:          gatewayIngressName(route),
		Namespace:     namespace,
		ProjectID:     project.ID,
		RouteID:       route.ID,
		Host:          strings.TrimSpace(route.Host),
		SecretName:    gatewayTLSSecretName(route),
		ClusterIssuer: strings.TrimSpace(clusterIssuer),
	}
}

func applicationResourcesSpec(release model.Release, project model.Project, application model.Application, environment model.Environment, deploymentTarget model.DeploymentTarget, namespace string, rolloutTimeoutSeconds int64) (kubeprovider.ApplicationResourcesSpec, error) {
	configData, err := mergeKeyValueMaps(environment.EnvVars, environment.ConfigRefs)
	if err != nil {
		return kubeprovider.ApplicationResourcesSpec{}, err
	}
	secretData, err := parseKeyValueMap(environment.SecretRefs)
	if err != nil {
		return kubeprovider.ApplicationResourcesSpec{}, err
	}
	servicePort := application.ServicePort
	if servicePort <= 0 {
		servicePort = 8080
	}
	replicas := environment.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	return kubeprovider.ApplicationResourcesSpec{
		Name:                  applicationResourceName(deploymentTarget),
		Namespace:             namespace,
		ProjectID:             project.ID,
		ApplicationID:         application.ID,
		EnvironmentID:         environment.ID,
		DeploymentTargetID:    deploymentTarget.ID,
		ReleaseID:             release.ID,
		Image:                 strings.TrimSpace(release.ImageRef),
		Replicas:              int32(replicas),
		ServicePort:           int32(servicePort),
		CPURequest:            strings.TrimSpace(environment.CPURequest),
		MemoryRequest:         strings.TrimSpace(environment.MemoryRequest),
		RolloutTimeoutSeconds: int32(rolloutTimeoutSeconds),
		ConfigData:            configData,
		SecretData:            secretData,
	}, nil
}

func mergeKeyValueMaps(values ...string) (map[string]string, error) {
	merged := map[string]string{}
	for _, value := range values {
		parsed, err := parseKeyValueMap(value)
		if err != nil {
			return nil, err
		}
		for key, item := range parsed {
			merged[key] = item
		}
	}
	return merged, nil
}

func parseKeyValueMap(value string) (map[string]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]string{}, nil
	}
	if strings.HasPrefix(value, "{") {
		var raw map[string]any
		if err := json.Unmarshal([]byte(value), &raw); err != nil {
			return nil, err
		}
		parsed := make(map[string]string, len(raw))
		for key, item := range raw {
			parsed[strings.TrimSpace(key)] = fmt.Sprint(item)
		}
		return compactKeyValueMap(parsed), nil
	}
	parsed := map[string]string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, item, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid key-value line %q", line)
		}
		parsed[strings.TrimSpace(key)] = strings.TrimSpace(item)
	}
	return compactKeyValueMap(parsed), nil
}

func compactKeyValueMap(values map[string]string) map[string]string {
	compacted := map[string]string{}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		compacted[key] = value
	}
	return compacted
}

func buildResourceName(buildRunID, prefix string) string {
	id := strings.ToLower(strings.TrimSpace(buildRunID))
	id = strings.TrimPrefix(id, "bldr_")
	var builder strings.Builder
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('-')
	}
	suffix := strings.Trim(builder.String(), "-")
	if suffix == "" {
		suffix = "run"
	}
	maxSuffix := 63 - len(prefix)
	if maxSuffix < 1 {
		maxSuffix = 1
	}
	if len(suffix) > maxSuffix {
		suffix = suffix[:maxSuffix]
	}
	return prefix + suffix
}

func dnsLabel(value string) string {
	label := dnsLabelOptionalSegment(value)
	if len(label) > 63 {
		label = strings.TrimRight(label[:63], "-")
	}
	if label == "" {
		return "liteyuki"
	}
	return label
}

func dnsLabelOptionalSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previousDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			previousDash = false
			continue
		}
		if !previousDash {
			builder.WriteByte('-')
			previousDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func logTask(ctx context.Context, task *asynq.Task) error {
	log.Printf("received task type=%s payload=%s", task.Type(), string(task.Payload()))
	return nil
}
