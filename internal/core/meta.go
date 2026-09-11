package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type LlamaClient struct {
	apiKey string
	url    string
	model  string
}

func NewLlamaClient() *LlamaClient {
	key := os.Getenv("LLAMA_API_KEY")
	if key == "" {
		key = os.Getenv("META_API_KEY") // fallback
	}
	if key == "" {
		panic("LLAMA_API_KEY ou META_API_KEY não definida")
	}

	model := os.Getenv("LLAMA_MODEL")
	if model == "" {
		// Modelo recomendado para código. Alternativas:
		// Llama-4-Scout-17B-16E-Instruct-FP8 (contexto gigante 10M)
		// Llama-3.3-70B-Instruct (mais barato)
		model = "Llama-4-Maverick-17B-128E-Instruct-FP8"
	}

	url := os.Getenv("LLAMA_API_URL")
	if url == "" {
		// endpoint compatível com OpenAI - troca direta da DeepSeek
		url = "https://api.llama.com/compat/v1/chat/completions"
	}

	return &LlamaClient{
		apiKey: key,
		url:    url,
		model:  model,
	}
}

// Implementa exatamente a mesma interface que seu DeepSeekClient
func (c *LlamaClient) Chat(prompt string) (string, error) {
	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
		// ISSO GARANTE QUE ELE RESPEITE SEUS DTOs RIGOROSAMENTE
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", c.url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("llama api erro %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %s", string(data))
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("resposta inválida: %s", string(data))
	}

	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	content, ok := msg["content"].(string)
	if !ok {
		return "", fmt.Errorf("conteúdo inválido: %s", string(data))
	}

	return content, nil
}
