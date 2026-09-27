package env

import (
	"os"

	carloenv "github.com/caarlos0/env/v11"
)

type Env struct {
	HTTPAddr        string `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL     string `env:"DATABASE_URL" envDefault:"postgres://kaiban:kaiban@localhost:5433/kaiban?sslmode=disable"`
	GitWorkDir      string `env:"GIT_WORK_DIR" envDefault:"/tmp/kaiban-git"`
	WorkerN         int    `env:"WORKER_N" envDefault:"2"`
	LLMBaseURL      string `env:"OPENAI_API_BASE" envDefault:"https://api.openai.com/v1"`
	LLMAPIKey       string `env:"OPENAI_API_KEY"`
	LLMModel        string `env:"OPENAI_MODEL" envDefault:"gpt-4.1"`
	JiraURL         string `env:"JIRA_URL"`
	JiraEmail       string `env:"JIRA_EMAIL"`
	JiraToken       string `env:"JIRA_TOKEN"`
	ConfluenceURL   string `env:"CONFLUENCE_URL"`
	ConfluenceEmail string `env:"CONFLUENCE_EMAIL"`
	ConfluenceToken string `env:"CONFLUENCE_TOKEN"`
	GitLabURL       string `env:"GITLAB_URL"`
	GitLabToken     string `env:"GITLAB_TOKEN"`
	GitHubURL       string `env:"GITHUB_URL" envDefault:"https://api.github.com"`
	GitHubToken     string `env:"GITHUB_TOKEN"`
}

func PrepareEnv() (*Env, error) {
	// Preserve alternate names used by the previous custom getenv fallbacks.
	prefer("OPENAI_API_BASE", "OPENAI_BASE_URL")
	prefer("JIRA_TOKEN", "JIRA_API_TOKEN")
	prefer("GITLAB_TOKEN", "GITLAB_PRIVATE_TOKEN")
	prefer("GITHUB_URL", "GITHUB_API_BASE")
	prefer("GITHUB_TOKEN", "GH_TOKEN")

	var e Env
	if err := carloenv.Parse(&e); err != nil {
		return nil, err
	}
	return &e, nil
}

// prefer copies fallback into primary when primary is unset or empty.
func prefer(primary, fallback string) {
	if os.Getenv(primary) != "" {
		return
	}
	if v := os.Getenv(fallback); v != "" {
		_ = os.Setenv(primary, v)
	}
}
