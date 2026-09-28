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

func NewGeminiClient(apiKey string) LlmInterface {
	if apiKey == "" {
		panic("GEMINI_API_KEY não definida (use --key ou export GEMINI_API_KEY)")
	}
	return &GeminiClient{
		apiKey: apiKey,
		url:    "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		model:  "gemini-2.5-flash",
		http:   &http.Client{Timeout: 300 * time.Second},
	}
}

// NewGeminiClientWithModel permite definir outro modelo do Gemini (ex: gemini-2.5-pro, gemini-2.0-flash)
func NewGeminiClientWithModel(apiKey, model string) LlmInterface {
	client := NewGeminiClient(apiKey).(*GeminiClient)
	if model != "" {
		client.model = model
	}
	return client
}

func (c *GeminiClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := dto.ChatRequest{
		Model:       c.model,
		Messages:    messages,
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

	resp, err := c.http.Do(req)
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
	payload := dto.ChatRequest{
		Model:       c.model,
		Messages:    messages,
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

	resp, err := c.http.Do(req)
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
