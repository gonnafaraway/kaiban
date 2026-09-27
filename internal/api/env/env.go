package env

import (
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	HTTPAddr        string
	DatabaseURL     string
	GitWorkDir      string
	WorkerN         int
	LLMBaseURL      string
	LLMAPIKey       string
	LLMModel        string
	JiraURL         string
	JiraEmail       string
	JiraToken       string
	ConfluenceURL   string
	ConfluenceEmail string
	ConfluenceToken string
	GitLabURL       string
	GitLabToken     string
	GitHubURL       string
	GitHubToken     string
}

func PrepareEnv() (*Env, error) {
	_ = godotenv.Load()
	return &Env{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://kaiban:kaiban@localhost:5433/kaiban?sslmode=disable"),
		GitWorkDir:      getenv("GIT_WORK_DIR", "/tmp/kaiban-git"),
		WorkerN:         2,
		LLMBaseURL:      getenv("OPENAI_API_BASE", getenv("OPENAI_BASE_URL", "https://api.openai.com/v1")),
		LLMAPIKey:       getenv("OPENAI_API_KEY", ""),
		LLMModel:        getenv("OPENAI_MODEL", "gpt-4.1"),
		JiraURL:         getenv("JIRA_URL", ""),
		JiraEmail:       getenv("JIRA_EMAIL", ""),
		JiraToken:       getenv("JIRA_TOKEN", getenv("JIRA_API_TOKEN", "")),
		ConfluenceURL:   getenv("CONFLUENCE_URL", ""),
		ConfluenceEmail: getenv("CONFLUENCE_EMAIL", ""),
		ConfluenceToken: getenv("CONFLUENCE_TOKEN", ""),
		GitLabURL:       getenv("GITLAB_URL", ""),
		GitLabToken:     getenv("GITLAB_TOKEN", getenv("GITLAB_PRIVATE_TOKEN", "")),
		GitHubURL:       getenv("GITHUB_URL", getenv("GITHUB_API_BASE", "https://api.github.com")),
		GitHubToken:     getenv("GITHUB_TOKEN", getenv("GH_TOKEN", "")),
	}, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
