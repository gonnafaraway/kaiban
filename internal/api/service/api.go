package service

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/repository"
	"kaiban/internal/api/usecase/kanban"
)

type APIService struct {
	addr string
	app  *fiber.App
	log  *zap.Logger
}

func PrepareAPIService(addr string, app *fiber.App, log *zap.Logger) *APIService {
	return &APIService{addr: addr, app: app, log: log}
}

func (s *APIService) Run() error {
	s.log.Info("http listen", zap.String("addr", s.addr))
	return s.app.Listen(s.addr)
}

type JobsService struct {
	repo *repository.Repository
	uc   *kanban.UseCase
	n    int
	log  *zap.Logger
}

func PrepareJobsService(repo *repository.Repository, uc *kanban.UseCase, n int, log *zap.Logger) *JobsService {
	if n < 1 {
		n = 1
	}
	return &JobsService{repo: repo, uc: uc, n: n, log: log}
}

const (
	jobLease     = 10 * time.Minute
	jobHeartbeat = 2 * time.Minute
)

func (s *JobsService) Run() error {
	ctx := context.Background()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		jobs, err := s.repo.Jobs.Lease(ctx, s.n, jobLease)
		if err != nil {
			s.log.Error("lease", zap.Error(err))
			continue
		}
		for _, j := range jobs {
			go s.handle(j)
		}
	}
	return nil
}

func (s *JobsService) handle(j *job.Job) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		t := time.NewTicker(jobHeartbeat)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if err := s.repo.Jobs.RenewLease(context.Background(), j.ID, jobLease); err != nil {
					s.log.Warn("renew lease", zap.Error(err), zap.String("job", j.ID.String()))
				}
			}
		}
	}()

	err := s.uc.ExecuteAgentJob(ctx, j)
	close(done)

	st := job.StatusSucceeded
	msg := ""
	if err != nil {
		st = job.StatusFailed
		msg = err.Error()
		s.log.Error("job", zap.Error(err), zap.String("job", j.ID.String()))
	}
	_ = s.repo.Jobs.Complete(context.Background(), j.ID, st, msg)
}
