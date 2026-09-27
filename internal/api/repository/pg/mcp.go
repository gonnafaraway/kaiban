package pg

import (
	"context"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/mcpserver"
)

func (s *Store) ListMCP(ctx context.Context) ([]*mcpserver.Server, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id,name,endpoint,headers,capabilities,status,COALESCE(last_error,''),created_at,updated_at FROM mcp_servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*mcpserver.Server
	for rows.Next() {
		m, err := scanMCP(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMCP(ctx context.Context, id uuid.UUID) (*mcpserver.Server, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT id,name,endpoint,headers,capabilities,status,COALESCE(last_error,''),created_at,updated_at FROM mcp_servers WHERE id=$1`, id)
	return scanMCP(row)
}

func (s *Store) CreateMCP(ctx context.Context, m *mcpserver.Server) error {
	h, err := marshalJSON(m.Headers, "mcp headers")
	if err != nil {
		return err
	}
	c, err := marshalJSON(m.Capabilities, "mcp capabilities")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `INSERT INTO mcp_servers (id,name,endpoint,headers,capabilities,status,last_error,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		m.ID, m.Name, m.Endpoint, h, c, string(m.Status), m.LastError, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *Store) UpdateMCP(ctx context.Context, m *mcpserver.Server) error {
	h, err := marshalJSON(m.Headers, "mcp headers")
	if err != nil {
		return err
	}
	c, err := marshalJSON(m.Capabilities, "mcp capabilities")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `UPDATE mcp_servers SET name=$2,endpoint=$3,headers=$4,capabilities=$5,status=$6,last_error=$7,updated_at=$8 WHERE id=$1`,
		m.ID, m.Name, m.Endpoint, h, c, string(m.Status), m.LastError, m.UpdatedAt)
	return err
}

func (s *Store) DeleteMCP(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM mcp_servers WHERE id=$1`, id)
	return err
}

func scanMCP(row rowScanner) (*mcpserver.Server, error) {
	m := &mcpserver.Server{Headers: map[string]string{}, Capabilities: map[string]any{}}
	var h, c []byte
	var st string
	err := row.Scan(&m.ID, &m.Name, &m.Endpoint, &h, &c, &st, &m.LastError, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, mapNoRows(err, "scan mcp server")
	}
	m.Status = mcpserver.Status(st)
	if err := unmarshalJSON(h, &m.Headers, "mcp headers"); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(c, &m.Capabilities, "mcp capabilities"); err != nil {
		return nil, err
	}
	return m, nil
}
