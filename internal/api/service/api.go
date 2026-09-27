package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/repository"
	"kaiban/internal/api/usecase/kanban"
)

type APIService struct {
	addr string
	app  *fiber.App
	log  *slog.Logger
}

func PrepareAPIService(addr string, app *fiber.App, log *slog.Logger) *APIService {
	return &APIService{addr: addr, app: app, log: log}
}

func (s *APIService) Run() error {
	s.log.Info("http listen", "addr", s.addr)
	return s.app.Listen(s.addr)
}

type JobsService struct {
	repo *repository.Repository
	uc   *kanban.UseCase
	n    int
	log  *slog.Logger
}

func PrepareJobsService(repo *repository.Repository, uc *kanban.UseCase, n int, log *slog.Logger) *JobsService {
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
	// Limit in-flight jobs to WorkerN (lease batch size alone is not a pool).
	slots := make(chan struct{}, s.n)
	for range ticker.C {
		free := s.n - len(slots)
		if free <= 0 {
			continue
		}
		jobs, err := s.repo.Jobs.Lease(ctx, free, jobLease)
		if err != nil {
			s.log.Error("lease", "error", err)
			continue
		}
		for _, j := range jobs {
			slots <- struct{}{}
			go func(j *job.Job) {
				defer func() { <-slots }()
				s.handle(j)
			}(j)
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
					s.log.Warn("renew lease", "error", err, "job", j.ID.String())
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
		s.log.Error("job", "error", err, "job", j.ID.String())
	}
	if err := s.repo.Jobs.Complete(context.Background(), j.ID, st, msg); err != nil {
		s.log.Error("complete job", "error", err, "job", j.ID.String())
	}
}
