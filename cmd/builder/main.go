package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/LiteyukiStudio/devops/internal/builder"
	"github.com/LiteyukiStudio/devops/internal/config"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent := builder.New(builder.Options{
		APIURL:            cfg.BuilderAPIURL,
		Token:             cfg.BuilderSharedToken,
		AgentID:           cfg.BuilderAgentID,
		Name:              cfg.BuilderAgentName,
		Executor:          cfg.BuilderExecutor,
		ExecutorImage:     cfg.BuilderExecutorImage,
		MaxConcurrency:    cfg.BuilderMaxConcurrency,
		PollInterval:      time.Duration(cfg.BuilderPollIntervalSeconds) * time.Second,
		WorkspaceRoot:     cfg.BuilderWorkspaceRoot,
		WorkspaceHostRoot: cfg.BuilderWorkspaceHostRoot,
		NPMRegistry:       cfg.BuilderNPMRegistry,
	})
	if err := agent.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("run builder: %v", err)
	}
}
