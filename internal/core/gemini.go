package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type GeminiClient struct {
	apiKey string
	url    string
	model  string
	http   *http.Client
}

type geminiFunctionCall struct {
	Name             string `json:"name"`
	Arguments        string `json:"arguments"`
	ThoughtSignature string `json:"thought_signature,omitempty"`
}

type geminiToolCall struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"`
	Function         geminiFunctionCall `json:"function"`
	ThoughtSignature string             `json:"thought_signature,omitempty"`
}

type geminiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []geminiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type geminiPayload struct {
	Model       string          `json:"model"`
	Messages    []geminiMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []dto.Tool      `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
}

func NewGeminiClient(apiKey string) LlmInterface {
	if apiKey == "" {
		panic("GEMINI_API_KEY não definida (use --key ou export GEMINI_API_KEY)")
	}
	return &GeminiClient{
		apiKey: apiKey,
		url:    "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		model:  "gemini-3.8-flash",
		http:   &http.Client{Timeout: 300 * time.Second},
	}
}

func NewGeminiClientWithModel(apiKey, model string) LlmInterface {
	client := NewGeminiClient(apiKey).(*GeminiClient)
	if model != "" {
		if len(model) > 7 && model[:7] == "models/" {
			model = model[7:]
		}
		client.model = model
	}
	return client
}

// prepareMessages converte []dto.Message garantindo o campo thought_signature em TODOS os níveis da chamada da função
func prepareMessages(messages []dto.Message) []geminiMessage {
	out := make([]geminiMessage, len(messages))
	for i, m := range messages {
		gm := geminiMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			gm.ToolCalls = make([]geminiToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				sig := tc.ThoughtSignature
				if sig == "" {
					sig = tc.Function.ThoughtSignature
				}
				if sig == "" {
					sig = "skip_thought_signature"
				}

				gm.ToolCalls[j] = geminiToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: geminiFunctionCall{
						Name:             tc.Function.Name,
						Arguments:        tc.Function.Arguments,
						ThoughtSignature: sig,
					},
					ThoughtSignature: sig,
				}
			}
		}
		out[i] = gm
	}
	return out
}

// executeWithRetry faz a tentativa automática em caso de instabilidade temporária (HTTP 503)
func (c *GeminiClient) executeWithRetry(req *http.Request) (*http.Response, error) {
	maxRetries := 3
	var resp *http.Response
	var err error

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < maxRetries; i++ {
		reqClone := req.Clone(req.Context())
		if bodyBytes != nil {
			reqClone.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		resp, err = c.http.Do(reqClone)
		if err == nil && resp.StatusCode != 503 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(2 * time.Second)
	}

	reqClone := req.Clone(req.Context())
	if bodyBytes != nil {
		reqClone.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return c.http.Do(reqClone)
}

func (c *GeminiClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := geminiPayload{
		Model:       c.model,
		Messages:    prepareMessages(messages),
		Temperature: 0.2,
		MaxTokens:   8192,
	}
	if len(tools) > 0 {
		payload.Tools = tools
		payload.ToolChoice = "auto"
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.executeWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gemini retornou %d: %s", resp.StatusCode, string(data))
	}

	var result dto.ChatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta do gemini: %v\n%s", err, string(data))
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("resposta do gemini sem choices: %s", string(data))
	}

	return &result, nil
}

func (c *GeminiClient) ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error {
	payload := geminiPayload{
		Model:       c.model,
		Messages:    prepareMessages(messages),
		Temperature: 0.2,
		Stream:      true,
	}
	if len(tools) > 0 {
		payload.Tools = tools
		payload.ToolChoice = "auto"
	}

	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.executeWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		d, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini stream %d: %s", resp.StatusCode, string(d))
	}

	return parseStream(resp.Body, onDelta, onToolCall)
}
