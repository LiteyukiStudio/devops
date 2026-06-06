package worker

import (
	"context"
	"log"

	"github.com/hibiken/asynq"
)

const (
	TypeBuildRun   = "build:run"
	TypeDeployRun  = "deploy:run"
	TypeSyncStatus = "sync:status"
)

func Run(redisAddr string) error {
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{Concurrency: 4},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeBuildRun, logTask)
	mux.HandleFunc(TypeDeployRun, logTask)
	mux.HandleFunc(TypeSyncStatus, logTask)

	return server.Run(mux)
}

func logTask(ctx context.Context, task *asynq.Task) error {
	log.Printf("received task type=%s payload=%s", task.Type(), string(task.Payload()))
	return nil
}
