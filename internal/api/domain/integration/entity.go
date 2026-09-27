package integration

import (
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/secret"
)

type Type string

const (
	TypeJira       Type = "jira"
	TypeConfluence Type = "confluence"
	TypeGitLab     Type = "gitlab"
	TypeGitHub     Type = "github"
)

type Status string

const (
	StatusDisabled Status = "disabled"
	StatusEnabled  Status = "enabled"
	StatusError    Status = "error"
)

type Integration struct {
	ID          uuid.UUID
	Type        Type
	Name        string
	BaseURL     string
	Credentials map[string]string
	Status      Status
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (i *Integration) MaskedCredentials() map[string]string {
	out := map[string]string{}
	for k, v := range i.Credentials {
		out[k] = secret.Mask(v)
	}
	return out
}
