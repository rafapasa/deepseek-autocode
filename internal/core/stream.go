package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

func parseStream(body io.Reader, onDelta func(string), onToolCall func(dto.ToolCall)) error {
	toolCallAcc := make(map[int]*dto.ToolCall)

	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
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
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id,omitempty"`
						Type     string `json:"type,omitempty"`
						Function struct {
							Name      string `json:"name,omitempty"`
							Arguments string `json:"arguments,omitempty"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
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
		if len(d.ToolCalls) > 0 {
			for _, tc := range d.ToolCalls {
				acc, ok := toolCallAcc[tc.Index]
				if !ok {
					acc = &dto.ToolCall{Type: "function"}
					toolCallAcc[tc.Index] = acc
				}
				if tc.ID != "" {
					acc.ID = tc.ID
				}
				if tc.Type != "" {
					acc.Type = tc.Type
				}
				if tc.Function.Name != "" {
					acc.Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					acc.Function.Arguments += tc.Function.Arguments
				}
			}
		}
	}
	if onToolCall != nil {
		idxs := make([]int, 0, len(toolCallAcc))
		for i := range toolCallAcc {
			idxs = append(idxs, i)
		}
		sort.Ints(idxs)
		for _, i := range idxs {
			tc := toolCallAcc[i]
			if tc.Function.Name == "" && tc.ID == "" {
				continue
			}
			if tc.Type == "" {
				tc.Type = "function"
			}
			if tc.ID == "" {
				tc.ID = fmt.Sprintf("call_%d", i)
			}
			onToolCall(*tc)
		}
	}
	return nil
}
