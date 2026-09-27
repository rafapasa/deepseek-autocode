package chat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rafapasa/deepseek-autocode/internal/config"
)

type ChatLLM struct {
	cfg    *config.Config
	client *http.Client
}

func NewChatLLM(cfg *config.Config) *ChatLLM {
	return &ChatLLM{cfg: cfg, client: &http.Client{Timeout: 120 * time.Second}}
}

type ChatCompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []LLMMessage     `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
	Stream      bool             `json:"stream"`
	Temperature float32          `json:"temperature,omitempty"`
}

type LLMMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

func (c *ChatLLM) BuildSystemPrompt(baseContent, projetoContent, projectRoot string) string {
	return fmt.Sprintf(`Você é o eTools-Code, assistente sênior da eTools Tecnologia.
Cores: Azul #1E3A5F e Verde #16A34A - use em qualquer UI gerada.

BASE.JSON:
%s

PROJETO.JSON:
%s

REGRAS:
1. Use list_files + read_file antes de afirmar
2. Crie com write_file, sempre dentro do projeto
3. Responda em pt-BR, código limpo
`, truncate(baseContent, 8000), truncate(projetoContent, 8000))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncado]"
}

func (c *ChatLLM) StreamChat(messages []LLMMessage, tools []ToolDefinition, onDelta func(string), onToolCall func(ToolCall)) error {
	var url, model, apiKey string
	if c.cfg.LlmClient == config.LLM_META {
		url = "https://api.groq.com/openai/v1/chat/completions"
		model = "meta-llama/llama-4-maverick-17b-128e-instruct"
		apiKey = c.cfg.MetaApiKey
		if apiKey == "" {
			apiKey = c.cfg.ResolveAPIKey()
		}
	} else {
		url = "https://api.deepseek.com/chat/completions"
		model = "deepseek-chat"
		apiKey = c.cfg.DeepSeekApiKey
		if apiKey == "" {
			apiKey = c.cfg.ResolveAPIKey()
		}
	}
	if apiKey == "" {
		return fmt.Errorf("API key vazia")
	}

	body := ChatCompletionRequest{Model: model, Messages: messages, Tools: tools, ToolChoice: "auto", Stream: true, Temperature: 0.2}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		d, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM %d: %s", resp.StatusCode, string(d))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		trim := bytes.TrimSpace(line)
		if len(trim) == 0 || !bytes.HasPrefix(trim, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(trim, []byte("data: "))
		if string(data) == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal(data, &chunk) != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		d := chunk.Choices[0].Delta
		if d.Content != "" && onDelta != nil {
			onDelta(d.Content)
		}
		if len(d.ToolCalls) > 0 && onToolCall != nil {
			for _, tc := range d.ToolCalls {
				onToolCall(tc)
			}
		}
	}
	return nil
}
