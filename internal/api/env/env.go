package env

import (
	"os"
	"time"

	carloenv "github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type Env struct {
	HTTPAddr        string        `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL     string        `env:"DATABASE_URL" envDefault:"postgres://kaiban:kaiban@localhost:5433/kaiban?sslmode=disable"`
	GitWorkDir      string        `env:"GIT_WORK_DIR" envDefault:"/tmp/kaiban-git"`
	WorkerN         int           `env:"WORKER_N" envDefault:"2"`
	LLMBaseURL      string        `env:"OPENAI_API_BASE" envDefault:"https://api.openai.com/v1"`
	LLMAPIKey       string        `env:"OPENAI_API_KEY"`
	LLMModel        string        `env:"OPENAI_MODEL" envDefault:"gpt-4.1"`
	LLMHTTPTimeout  time.Duration `env:"LLM_HTTP_TIMEOUT" envDefault:"10m"`
	JiraURL         string        `env:"JIRA_URL"`
	JiraEmail       string        `env:"JIRA_EMAIL"`
	JiraToken       string        `env:"JIRA_TOKEN"`
	ConfluenceURL   string        `env:"CONFLUENCE_URL"`
	ConfluenceEmail string        `env:"CONFLUENCE_EMAIL"`
	ConfluenceToken string        `env:"CONFLUENCE_TOKEN"`
	GitLabURL       string        `env:"GITLAB_URL"`
	GitLabToken     string        `env:"GITLAB_TOKEN"`
	GitHubURL       string        `env:"GITHUB_URL" envDefault:"https://api.github.com"`
	GitHubToken     string        `env:"GITHUB_TOKEN"`
}

func PrepareEnv() (*Env, error) {
	var e Env
	if err := carloenv.Parse(&e); err != nil {
		return nil, errors.Wrap(err, "parse env")
	}
	// Alternate names kept from the previous custom getenv fallbacks.
	e.LLMBaseURL = alias("OPENAI_API_BASE", "OPENAI_BASE_URL", e.LLMBaseURL)
	e.JiraToken = alias("JIRA_TOKEN", "JIRA_API_TOKEN", e.JiraToken)
	e.GitLabToken = alias("GITLAB_TOKEN", "GITLAB_PRIVATE_TOKEN", e.GitLabToken)
	e.GitHubURL = alias("GITHUB_URL", "GITHUB_API_BASE", e.GitHubURL)
	e.GitHubToken = alias("GITHUB_TOKEN", "GH_TOKEN", e.GitHubToken)
	return &e, nil
}

// alias returns the fallback variable when the primary one is unset or empty.
// The process environment is left untouched: parsed keeps the value from Parse
// (including envDefault) whenever the primary variable is set.
func alias(primary, fallback, parsed string) string {
	if os.Getenv(primary) != "" {
		return parsed
	}
	if v := os.Getenv(fallback); v != "" {
		return v
	}
	return parsed
}
