package storage

import (
	"context"

	"kaiban/internal/api/env"
	"kaiban/internal/api/storage/postgres"
)

type Storage struct {
	Postgres *postgres.Client
}

func PrepareStorage(e *env.Env) (*Storage, error) {
	client, err := postgres.New(context.Background(), e.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return &Storage{Postgres: client}, nil
}
