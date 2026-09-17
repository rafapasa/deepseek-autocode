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

type DeepSeekClient struct {
	apiKey string
	url    string
	http   *http.Client
}

func NewDeepSeekClient(apiKey string) LlmInterface {
	if apiKey == "" {
		panic("DEEPSEEK_API_KEY não definida (use --key ou export DEEPSEEK_API_KEY)")
	}
	return &DeepSeekClient{
		apiKey: apiKey,
		url:    "https://api.deepseek.com/v1/chat/completions",
		http:   &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *DeepSeekClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := dto.ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Temperature: 0.2,
		// MaxTokens:   8192,
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
		return nil, fmt.Errorf("deepseek retornou %d: %s", resp.StatusCode, string(data))
	}

	var result dto.ChatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("erro ao parsear resposta: %v\n%s", err, string(data))
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("resposta sem choices: %s", string(data))
	}

	return &result, nil
}
