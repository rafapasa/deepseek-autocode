package core

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

type DeepSeekClient struct {
    apiKey string
    url    string
}

func NewDeepSeekClient() *DeepSeekClient {
    key := os.Getenv("DEEPSEEK_API_KEY")
    if key == "" {
        panic("DEEPSEEK_API_KEY não definida")
    }
    return &DeepSeekClient{
        apiKey: key,
        url:    "https://api.deepseek.com/v1/chat/completions",
    }
}

func (c *DeepSeekClient) Chat(prompt string) (string, error) {
    payload := map[string]interface{}{
        "model": "deepseek-chat",
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
        "temperature": 0.2,
    }

    body, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", c.url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    data, _ := io.ReadAll(resp.Body)

    var result map[string]interface{}
    json.Unmarshal(data, &result)

    choices, ok := result["choices"].([]interface{})
    if !ok || len(choices) == 0 {
        return "", fmt.Errorf("resposta inválida: %s", string(data))
    }

    msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
    content := msg["content"].(string)

    return content, nil
}
