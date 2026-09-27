package pg

import (
	"context"

	"github.com/google/uuid"

	domint "kaiban/internal/api/domain/integration"
)

func (s *Store) ListIntegrations(ctx context.Context) ([]*domint.Integration, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id,type,name,base_url,credentials,status,COALESCE(last_error,''),created_at,updated_at FROM integrations ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domint.Integration
	for rows.Next() {
		i, err := scanInt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) GetIntegration(ctx context.Context, id uuid.UUID) (*domint.Integration, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT id,type,name,base_url,credentials,status,COALESCE(last_error,''),created_at,updated_at FROM integrations WHERE id=$1`, id)
	return scanInt(row)
}

func (s *Store) CreateIntegration(ctx context.Context, i *domint.Integration) error {
	cred, err := marshalJSON(i.Credentials, "integration credentials")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `INSERT INTO integrations (id,type,name,base_url,credentials,status,last_error,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		i.ID, string(i.Type), i.Name, i.BaseURL, cred, string(i.Status), i.LastError, i.CreatedAt, i.UpdatedAt)
	return err
}

func (s *Store) UpdateIntegration(ctx context.Context, i *domint.Integration) error {
	cred, err := marshalJSON(i.Credentials, "integration credentials")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `UPDATE integrations SET type=$2,name=$3,base_url=$4,credentials=$5,status=$6,last_error=$7,updated_at=$8 WHERE id=$1`,
		i.ID, string(i.Type), i.Name, i.BaseURL, cred, string(i.Status), i.LastError, i.UpdatedAt)
	return err
}

func (s *Store) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM integrations WHERE id=$1`, id)
	return err
}

func scanInt(row rowScanner) (*domint.Integration, error) {
	i := &domint.Integration{Credentials: map[string]string{}}
	var typ, st string
	var cred []byte
	err := row.Scan(&i.ID, &typ, &i.Name, &i.BaseURL, &cred, &st, &i.LastError, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, mapNoRows(err, "scan integration")
	}
	i.Type = domint.Type(typ)
	i.Status = domint.Status(st)
	if err := unmarshalJSON(cred, &i.Credentials, "integration credentials"); err != nil {
		return nil, err
	}
	return i, nil
}
