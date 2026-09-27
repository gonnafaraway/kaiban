package pg

import (
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

	"kaiban/internal/api/repository"
	"kaiban/internal/api/storage/postgres"
)

type Store struct {
	db *postgres.Client
}

func New(db *postgres.Client) *Store { return &Store{db: db} }

type rowScanner interface {
	Scan(dest ...any) error
}

// mapNoRows translates the driver "no rows" error into repository.ErrNotFound so
// callers above the repository layer never touch pgx.
func mapNoRows(err error, what string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.Wrap(repository.ErrNotFound, what)
	}
	return errors.Wrap(err, what)
}

func marshalJSON(v any, what string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "marshal "+what)
	}
	return b, nil
}

func unmarshalJSON(data []byte, dest any, what string) error {
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return errors.Wrap(err, "unmarshal "+what)
	}
	return nil
}
