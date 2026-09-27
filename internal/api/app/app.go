package app

import (
	"context"

	"kaiban/internal/api/env"
	"kaiban/internal/api/handlers"
	"kaiban/internal/api/infrastructure/local/log"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/repository/pg"
	"kaiban/internal/api/service"
	"kaiban/internal/api/storage"
	httptransport "kaiban/internal/api/transport/http"
	"kaiban/internal/api/usecase/kanban"
)

func Run() error {
	e, err := env.PrepareEnv()
	if err != nil {
		return err
	}
	logger, err := log.NewLogger()
	if err != nil {
		logger = log.NewFallbackLogger()
	}

	storages, err := storage.PrepareStorage(e)
	if err != nil {
		return err
	}
	if err := pg.ApplySchema(context.Background(), storages, e); err != nil {
		return err
	}
	repo, err := pg.PrepareRepository(storages)
	if err != nil {
		return err
	}
	hub := httptransport.NewHub()
	uc := kanban.Prepare(repo, integration.NewOpenAI(), e.GitWorkDir, hub)
	fiberApp := httptransport.NewApp(logger)
	handlers.Register(fiberApp, handlers.Deps{UC: uc, Hub: hub})

	apiSvc := service.PrepareAPIService(e.HTTPAddr, fiberApp, logger)
	jobsSvc := service.PrepareJobsService(repo, uc, e.WorkerN, logger)
	return service.RunServices(apiSvc, jobsSvc)
}
