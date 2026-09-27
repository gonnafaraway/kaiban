export type Artifacts = {
  jira_issue: string;
  confluence_url: string;
  gitlab_repo: string;
  github_repo: string;
};

export type OutputField = {
  key: string;
  label: string;
  required: boolean;
  type: "string" | "url" | "markdown";
};

export type BudgetLimits = {
  max_tokens?: number | null;
  max_cost_usd?: number | null;
  max_wall_sec?: number | null;
  max_tool_calls?: number | null;
  max_llm_steps?: number | null;
};

export type ColumnReport = {
  column_id: string;
  report_md: string;
  created_at: string;
};

export type Task = {
  id: string;
  title: string;
  description: string;
  variables: Record<string, string>;
  column_id: string;
  execution_status: string;
  git_branch: string;
  git_pr_url?: string;
  git_push_status?: string;
  git_pr_status?: string;
  current_report: string;
  reports?: ColumnReport[];
  context_data: {
    extra_instructions?: string;
    retry_notes?: string[];
    artifacts?: Artifacts;
    stage_outputs?: Record<string, Record<string, string>>;
    budget?: BudgetLimits;
  };
  created_at: string;
  updated_at: string;
  archived_at?: string | null;
};

export type Column = {
  id: string;
  name: string;
  name_i18n: Record<string, string>;
  system_prompt_default: string;
  system_prompt_template: string;
  user_custom_prompt: string | null;
  output_fields?: OutputField[];
  budget?: BudgetLimits;
  requires_git_diff?: boolean;
  order_index: number;
};

export type ContextPackItem = {
  title: string;
  body: string;
  enabled: boolean;
  order: number;
};

export type Settings = {
  llm_base_url: string;
  llm_api_key: string;
  llm_model: string;
  git_repo_url: string;
  git_default_branch: string;
  locale: "ru" | "en";
  max_tokens?: number;
  max_cost_usd?: number;
  max_wall_sec?: number;
  max_tool_calls?: number;
  max_llm_steps?: number;
  price_input_per_1k?: number;
  price_output_per_1k?: number;
  context_pack?: ContextPackItem[];
};

export type AgentRun = {
  id: string;
  task_id: string;
  column_id: string;
  status: string;
  model: string;
  prompt_hash: string;
  context_pack_hash?: string;
  stop_reason: string;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  tool_calls: number;
  llm_steps: number;
  started_at: string;
  finished_at?: string | null;
  events?: AgentRunEvent[];
};

export type DiffFile = {
  path: string;
  insertions: number;
  deletions: number;
};

export type DiffSummary = {
  base: string;
  branch: string;
  files: DiffFile[];
  insertions: number;
  deletions: number;
  commits: string[];
  empty: boolean;
  pr_url?: string;
  push_status?: string;
  pr_status?: string;
  error?: string;
};

export type AgentRunEvent = {
  id: string;
  run_id: string;
  seq: number;
  kind: string;
  message: string;
  payload?: Record<string, unknown>;
  created_at: string;
};

export type AuditEvent = {
  id: string;
  action: string;
  actor_type: string;
  actor_id: string;
  payload: Record<string, unknown>;
  created_at: string;
};

export type Integration = {
  id: string;
  type: string;
  name: string;
  base_url: string;
  credentials: Record<string, string>;
  status: string;
  last_error: string;
};

export type McpServer = {
  id: string;
  name: string;
  endpoint: string;
  headers: Record<string, string>;
  capabilities: Record<string, unknown>;
  status: string;
  last_error: string;
};

export type ConfigBundle = {
  version: number;
  exported_at?: string;
  settings?: Settings & { id?: string; updated_at?: string };
  integrations?: Array<{
    id?: string;
    type: string;
    name: string;
    base_url: string;
    credentials: Record<string, string>;
    status: string;
  }>;
  mcp_servers?: Array<{
    id?: string;
    name: string;
    endpoint: string;
    headers: Record<string, string>;
    capabilities?: Record<string, unknown>;
    status: string;
  }>;
  columns?: Array<
    Partial<Column> & {
      id?: string;
      name: string;
      order_index: number;
    }
  >;
};

export type ImportConfigResult = {
  mode: string;
  version: number;
  settings_updated: boolean;
  integrations: number;
  mcp_servers: number;
  columns: number;
  replaced_integrations: boolean;
  replaced_mcp_servers: boolean;
  replaced_columns: boolean;
};

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error?.message ?? res.statusText);
  }
  return data as T;
}

export const api = {
  columns: () => req<Column[]>("/api/v1/columns"),
  tasks: () => req<Task[]>("/api/v1/tasks"),
  archive: () => req<Task[]>("/api/v1/archive"),
  task: (id: string) => req<Task>(`/api/v1/tasks/${id}`),
  events: (id: string) => req<AuditEvent[]>(`/api/v1/tasks/${id}/events`),
  runs: (taskId: string) => req<AgentRun[]>(`/api/v1/tasks/${taskId}/runs`),
  runDetail: (runId: string) => req<AgentRun>(`/api/v1/runs/${runId}`),
  createTask: (body: {
    title: string;
    description: string;
    variables: Record<string, string>;
    artifacts: Artifacts;
  }) => req<Task>("/api/v1/tasks", { method: "POST", body: JSON.stringify(body) }),
  patchTask: (
    id: string,
    body: Partial<{ title: string; description: string; artifacts: Artifacts; budget: BudgetLimits }>,
  ) => req<Task>(`/api/v1/tasks/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  run: (id: string) => req<Task>(`/api/v1/tasks/${id}/run`, { method: "POST" }),
  approve: (id: string, comment: string) =>
    req<Task>(`/api/v1/tasks/${id}/approve`, { method: "POST", body: JSON.stringify({ comment }) }),
  retry: (id: string, comment: string) =>
    req<Task>(`/api/v1/tasks/${id}/retry`, { method: "POST", body: JSON.stringify({ comment }) }),
  returnTo: (id: string, column_id: string, comment: string) =>
    req<Task>(`/api/v1/tasks/${id}/return`, { method: "POST", body: JSON.stringify({ column_id, comment }) }),
  archiveTask: (id: string) => req<Task>(`/api/v1/tasks/${id}/archive`, { method: "POST" }),
  unarchiveTask: (id: string) => req<Task>(`/api/v1/tasks/${id}/unarchive`, { method: "POST" }),
  settings: () => req<Settings>("/api/v1/settings"),
  saveSettings: (body: Partial<Settings>) => req<Settings>("/api/v1/settings", { method: "PUT", body: JSON.stringify(body) }),
  exportConfig: () => req<ConfigBundle>("/api/v1/settings/export"),
  importConfig: (body: ConfigBundle) =>
    req<ImportConfigResult>("/api/v1/settings/import", { method: "POST", body: JSON.stringify(body) }),
  taskDiff: (id: string) => req<DiffSummary>(`/api/v1/tasks/${id}/diff`),
  taskDiffRaw: async (id: string) => {
    const res = await fetch(`/api/v1/tasks/${id}/diff/raw`);
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error((data as { error?: { message?: string } })?.error?.message ?? res.statusText);
    }
    return res.text();
  },
  patchColumn: (id: string, body: Partial<Column>) =>
    req<Column>(`/api/v1/columns/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  resetOverlay: (id: string) => req<Column>(`/api/v1/columns/${id}/reset-overlay`, { method: "POST" }),
  restoreDefault: (id: string) => req<Column>(`/api/v1/columns/${id}/restore-default-prompt`, { method: "POST" }),
  integrations: () => req<Integration[]>("/api/v1/integrations"),
  createIntegration: (body: object) => req<Integration>("/api/v1/integrations", { method: "POST", body: JSON.stringify(body) }),
  testIntegration: (id: string) => req<Integration>(`/api/v1/integrations/${id}/test`, { method: "POST" }),
  deleteIntegration: (id: string) => req<void>(`/api/v1/integrations/${id}`, { method: "DELETE" }),
  mcp: () => req<McpServer[]>("/api/v1/mcp-servers"),
  createMcp: (body: object) => req<McpServer>("/api/v1/mcp-servers", { method: "POST", body: JSON.stringify(body) }),
  patchMcp: (id: string, body: Partial<{ name: string; endpoint: string; headers: Record<string, string>; status: string }>) =>
    req<McpServer>(`/api/v1/mcp-servers/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  testMcp: (id: string) => req<McpServer>(`/api/v1/mcp-servers/${id}/test`, { method: "POST" }),
  deleteMcp: (id: string) => req<void>(`/api/v1/mcp-servers/${id}`, { method: "DELETE" }),
};
