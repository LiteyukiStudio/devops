package api

import (
	"context"

	"github.com/LiteyukiStudio/devops/internal/config"
	"github.com/LiteyukiStudio/devops/internal/repository"
	"github.com/LiteyukiStudio/devops/internal/secret"
	"github.com/LiteyukiStudio/devops/internal/tasks"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const rememberCookiePrefix = "lyd_remember_"
const sessionCookieName = "lyd_session"

type Handlers struct {
	db                  *gorm.DB
	configs             *configCache
	mode                string
	rateLimiter         *rateLimiter
	oauthStates         oauthStateStore
	projects            repository.ProjectRepository
	secrets             secret.Store
	branchCache         *gitBranchCache
	registrySearchCache *registrySearchCache
	taskClient          taskEnqueuer
	builderToken        string
}

type taskEnqueuer interface {
	EnqueueDeployRun(ctx context.Context, payload tasks.DeployRunPayload) (*asynq.TaskInfo, error)
	EnqueueGatewayApply(ctx context.Context, payload tasks.GatewayApplyPayload) (*asynq.TaskInfo, error)
}

func NewHandlers(db *gorm.DB) *Handlers {
	mode := config.RuntimeMode()
	if mode == "development" {
		ensureDevelopmentAdmin(db)
	}
	cfg := config.Load()
	handlers := &Handlers{db: db, configs: newConfigCache(db), mode: mode, rateLimiter: newRateLimiter(cfg.RedisAddr), oauthStates: newOAuthStateStore(cfg.RedisAddr), projects: repository.NewProjectRepository(db), branchCache: newGitBranchCache(), registrySearchCache: newRegistrySearchCache()}
	if cfg.RedisAddr != "" {
		handlers.taskClient = tasks.NewClient(cfg.RedisAddr)
	}
	handlers.builderToken = cfg.BuilderToken
	handlers.secrets = secret.NewStore(db, handlers.audit)
	return handlers
}
