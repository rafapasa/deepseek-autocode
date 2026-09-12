package dto

// ===== Chat / Tool Calling =====

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ===== Issue =====

type Request struct {
	Demanda   string        `json:"demanda"`
	Estrutura []interface{} `json:"estrutura"`
	Raiz      string        `json:"raiz"`
	Rules     []string      `json:"rules"`
}

// ===== Planejamento =====

// PlanResponse é o que a LLM responde no turno de planejamento.

type Part struct {
	ID      int      `json:"id"`
	Demanda string   `json:"demanda"`
	Files   []string `json:"files"`             // paths relativos à raiz
	Summary string   `json:"summary,omitempty"` // preenchido após execução
}

type PlanResponse struct {
	Status      string       `json:"status"`
	Parts       []Part       `json:"parts,omitempty"`
	Reason      string       `json:"reason,omitempty"`
	FilesNeeded []FileNeeded `json:"files_needed,omitempty"`
}

type FileNeeded struct {
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty"`
}
