package pg

import (
	"context"

	"kaiban/internal/api/domain/user"
)

func (s *Store) GetLocalUser(ctx context.Context) (*user.User, error) {
	u := &user.User{}
	err := s.db.Pool.QueryRow(ctx, `SELECT id, login, display_name, created_at FROM users WHERE login='local'`).
		Scan(&u.ID, &u.Login, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, mapNoRows(err, "scan local user")
	}
	return u, nil
}
