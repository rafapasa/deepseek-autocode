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
		panic("LLAMA_API_KEY não definida (use --key ou export LLAMA_API_KEY)")
	}
	return &LlamaClient{
		apiKey: apiKey,
		// endpoint compatível OpenAI oficial da Meta
		url:  "https://api.llama.com/compat/v1/chat/completions",
		http: &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *LlamaClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := dto.ChatRequest{
		Model:       "Llama-4-Maverick-17B-128E-Instruct-FP8", // flagship 400B MoE - 17B ativo
		Messages:    messages,
		Temperature: 0.2,
		MaxTokens:   32768,
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