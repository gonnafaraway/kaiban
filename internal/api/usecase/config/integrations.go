package config

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/secret"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/usecase/core"
)

func (u *Manager) ListIntegrations(ctx context.Context) ([]*domint.Integration, error) {
	items, err := u.Repo.Integrations.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list integrations")
	}
	return items, nil
}

func (u *Manager) UpsertIntegration(ctx context.Context, id *uuid.UUID, typ, name, base string, cred map[string]string, status string) (*domint.Integration, error) {
	now := time.Now().UTC()
	var item *domint.Integration
	var err error
	if id != nil {
		item, err = u.Repo.Integrations.Get(ctx, *id)
		if err != nil {
			return nil, errors.Wrap(err, "get integration")
		}
		item.Name, item.BaseURL, item.Type = name, base, domint.Type(typ)
		if len(cred) > 0 {
			if item.Credentials == nil {
				item.Credentials = map[string]string{}
			}
			// The API shows credentials masked, so an empty or echoed
			// placeholder means "keep the stored secret".
			for k, v := range cred {
				if v == "" || v == secret.Mask(item.Credentials[k]) {
					continue
				}
				item.Credentials[k] = v
			}
		}
		if status != "" {
			item.Status = domint.Status(status)
		}
		item.UpdatedAt = now
		if err := u.Repo.Integrations.Update(ctx, item); err != nil {
			return nil, errors.Wrap(err, "update")
		}
		return item, nil
	}
	item = &domint.Integration{
		ID: uuid.New(), Type: domint.Type(typ), Name: name, BaseURL: base,
		Credentials: cred, Status: domint.StatusDisabled, CreatedAt: now, UpdatedAt: now,
	}
	if status != "" {
		item.Status = domint.Status(status)
	}
	if err := u.Repo.Integrations.Create(ctx, item); err != nil {
		return nil, errors.Wrap(err, "create")
	}
	return item, nil
}

func (u *Manager) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	if err := u.Repo.Integrations.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete integration")
	}
	return nil
}

// probeTimeout bounds one connectivity check.
const probeTimeout = 10 * time.Second

// TestIntegration probes the type-specific health endpoint with the stored
// credentials and stores the verdict.
func (u *Manager) TestIntegration(ctx context.Context, id uuid.UUID) (*domint.Integration, error) {
	item, err := u.Repo.Integrations.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get integration")
	}
	item.UpdatedAt = time.Now().UTC()
	if probeErr := probeIntegration(ctx, item); probeErr != nil {
		item.Status = domint.StatusError
		item.LastError = probeErr.Error()
	} else {
		item.Status = domint.StatusEnabled
		item.LastError = ""
	}
	if err := u.Repo.Integrations.Update(ctx, item); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	return item, nil
}

// probeIntegration calls the "who am I" endpoint of that integration type, so a
// server that answers 200 to anonymous requests cannot mask bad credentials.
func probeIntegration(ctx context.Context, item *domint.Integration) error {
	kind := string(item.Type)
	probeURL := integration.HealthProbeURL(kind, item.BaseURL)
	if probeURL == "" {
		return errors.New("base_url is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return errors.Wrap(err, "build probe request")
	}
	email, tok := core.IntegrationCreds(item)
	for k, v := range integration.HealthProbeAuth(kind, email, tok) {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: probeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "probe request")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return errors.Errorf("HTTP %d on %s", resp.StatusCode, probeURL)
	}
	return nil
}
