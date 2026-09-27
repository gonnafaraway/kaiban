package integration

import (
	"time"

	"github.com/google/uuid"
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
		out[k] = mask(v)
	}
	return out
}

func mask(v string) string {
	if len(v) <= 4 {
		if v == "" {
			return ""
		}
		return "****"
	}
	return "****" + v[len(v)-4:]
}
