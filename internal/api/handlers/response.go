package handlers

import (
	"strings"
	"time"

	"github.com/google/uuid"

	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/domain/secret"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/usecase/config"
)

type taskResponse struct {
	ID              uuid.UUID            `json:"id"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	Variables       map[string]string    `json:"variables"`
	ColumnID        uuid.UUID            `json:"column_id"`
	ExecutionStatus task.ExecutionStatus `json:"execution_status"`
	GitBranch       string               `json:"git_branch"`
	GitPRURL        string               `json:"git_pr_url"`
	GitPushStatus   string               `json:"git_push_status"`
	GitPRStatus     string               `json:"git_pr_status"`
	CurrentReport   string               `json:"current_report"`
	ContextData     task.ContextData     `json:"context_data"`
	CreatedBy       uuid.UUID            `json:"created_by"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	ArchivedAt      *time.Time           `json:"archived_at"`
	Reports         any                  `json:"reports"`
}

func taskPayload(t *task.Task, reports any) taskResponse {
	return taskResponse{
		ID: t.ID, Title: t.Title, Description: t.Description, Variables: t.Variables,
		ColumnID: t.ColumnID, ExecutionStatus: t.ExecutionStatus, GitBranch: t.GitBranch,
		GitPRURL: t.GitPRURL, GitPushStatus: t.GitPushStatus, GitPRStatus: t.GitPRStatus,
		CurrentReport: t.CurrentReport, ContextData: t.ContextData, CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, ArchivedAt: t.ArchivedAt, Reports: reports,
	}
}

type settingsResponse struct {
	ID               uuid.UUID                  `json:"id"`
	LLMProvider      string                     `json:"llm_provider"`
	LLMBaseURL       string                     `json:"llm_base_url"`
	LLMAPIKey        string                     `json:"llm_api_key"`
	LLMModel         string                     `json:"llm_model"`
	GitRepoURL       string                     `json:"git_repo_url"`
	GitDefaultBranch string                     `json:"git_default_branch"`
	Locale           string                     `json:"locale"`
	MaxTokens        int                        `json:"max_tokens"`
	MaxCostUSD       float64                    `json:"max_cost_usd"`
	MaxWallSec       int                        `json:"max_wall_sec"`
	MaxToolCalls     int                        `json:"max_tool_calls"`
	MaxLLMSteps      int                        `json:"max_llm_steps"`
	PriceInputPer1K  float64                    `json:"price_input_per_1k"`
	PriceOutputPer1K float64                    `json:"price_output_per_1k"`
	ContextPack      []settings.ContextPackItem `json:"context_pack"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

func maskSettings(s *settings.Settings) settingsResponse {
	provider := s.LLMProvider
	if provider == "" {
		provider = settings.InferProviderFromURL(s.LLMBaseURL)
	}
	return settingsResponse{
		ID: s.ID, LLMProvider: provider, LLMBaseURL: s.LLMBaseURL, LLMAPIKey: s.MaskedKey(),
		LLMModel: s.LLMModel, GitRepoURL: s.GitRepoURL, GitDefaultBranch: s.GitDefaultBranch,
		Locale: s.Locale, MaxTokens: s.MaxTokens, MaxCostUSD: s.MaxCostUSD, MaxWallSec: s.MaxWallSec,
		MaxToolCalls: s.MaxToolCalls, MaxLLMSteps: s.MaxLLMSteps,
		PriceInputPer1K: s.PriceInputPer1K, PriceOutputPer1K: s.PriceOutputPer1K,
		ContextPack: s.ContextPack, UpdatedAt: s.UpdatedAt,
	}
}

type llmModelsResponse struct {
	Provider string                  `json:"provider"`
	Models   []config.LLMModelOption `json:"models"`
}

type integrationResponse struct {
	ID          uuid.UUID         `json:"id"`
	Type        domint.Type       `json:"type"`
	Name        string            `json:"name"`
	BaseURL     string            `json:"base_url"`
	Credentials map[string]string `json:"credentials"`
	Status      domint.Status     `json:"status"`
	LastError   string            `json:"last_error"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func maskInt(i *domint.Integration) integrationResponse {
	return integrationResponse{
		ID: i.ID, Type: i.Type, Name: i.Name, BaseURL: i.BaseURL,
		Credentials: i.MaskedCredentials(), Status: i.Status, LastError: i.LastError,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

type mcpResponse struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Endpoint     string            `json:"endpoint"`
	Headers      map[string]string `json:"headers"`
	Capabilities map[string]any    `json:"capabilities"`
	Status       mcpserver.Status  `json:"status"`
	LastError    string            `json:"last_error"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func maskMCP(s *mcpserver.Server) mcpResponse {
	h := map[string]string{}
	for k, v := range s.Headers {
		if strings.EqualFold(k, "Authorization") {
			h[k] = secret.Mask(v)
		} else {
			h[k] = v
		}
	}
	return mcpResponse{
		ID: s.ID, Name: s.Name, Endpoint: s.Endpoint, Headers: h,
		Capabilities: s.Capabilities, Status: s.Status, LastError: s.LastError,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
