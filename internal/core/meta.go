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

type LlamaClient struct {
	apiKey string
	url    string
	http   *http.Client
}

func NewLlamaClient(apiKey string) LlmInterface {
	if apiKey == "" {
		panic("GROQ_API_KEY não definida. Use: export GROQ_API_KEY=gsk_...")
	}
	return &LlamaClient{
		apiKey: apiKey,
		url:    "https://api.groq.com/openai/v1/chat/completions",
		http:   &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *LlamaClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := dto.ChatRequest{
		Model:       "llama-3.3-70b-versatile",
		Messages:    messages,
		Temperature: 0.2,
		MaxTokens:   8192,
	}
	if len(tools) > 0 {
		payload.Tools = tools
		payload.ToolChoice = "auto"
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", c.url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar groq: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("llama retornou %d: %s", resp.StatusCode, string(data))
	}

	var result dto.ChatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta llama: %v\n%s", err, string(data))
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("resposta llama sem choices: %s", string(data))
	}

	return &result, nil
}
