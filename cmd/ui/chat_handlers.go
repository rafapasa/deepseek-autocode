package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/deepseek-autocode/internal/chat"
	"github.com/rafapasa/deepseek-autocode/internal/config"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type ChatHandler struct {
	cfg     *config.Config
	service *chat.Service
}

func NewChatHandler(cfg *config.Config) *ChatHandler {
	if err := chat.MergeHistoryByProject(); err != nil {
		log.Printf("[chat] merge histórico: %v", err)
	}
	return &ChatHandler{cfg: cfg, service: chat.NewService(cfg)}
}

func (h *ChatHandler) CreateChat(c *fiber.Ctx) error {
	var body struct {
		Project string `json:"project"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	session, err := h.service.CreateSession(body.Project)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"id": session.ID, "project": session.Project})
}

func (h *ChatHandler) ListChats(c *fiber.Ctx) error {
	list, err := chat.ListSessions()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"chats": list})
}

func (h *ChatHandler) GetChat(c *fiber.Ctx) error {
	s, err := chat.LoadSession(c.Params("id"))
	if err != nil {
		return c.Status(404).SendString("chat não encontrado")
	}
	return c.JSON(s)
}

func (h *ChatHandler) PostMessage(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if body.Content == "" {
		return c.Status(400).SendString("content vazio")
	}

	session, err := h.service.AddUserMessage(id, body.Content)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	projectRoot := h.service.ResolveProjectRoot(session.Project)
	tools := chat.GetTools()
	llm := chat.NewChatLLM(h.cfg)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for iter := 0; iter < 16; iter++ {
			messages := chat.SanitizeMessages(session.Messages)
			var full string
			var toolCalls []dto.ToolCall

			err := llm.StreamChat(messages, tools,
				func(delta string) {
					full += delta
					writeSSEChat(w, fiber.Map{"type": "delta", "content": delta})
				},
				func(tc dto.ToolCall) {
					if tc.Type == "" {
						tc.Type = "function"
					}
					if tc.ID == "" {
						tc.ID = fmt.Sprintf("call_%d_%d", iter, len(toolCalls))
					}
					toolCalls = append(toolCalls, tc)
				},
			)
			if err != nil {
				writeSSEChat(w, fiber.Map{"type": "error", "error": err.Error()})
				return
			}

			if len(toolCalls) == 0 {
				if full != "" {
					session.Messages = append(session.Messages, dto.Message{Role: "assistant", Content: full})
					if err := h.service.PersistTurn(session); err != nil {
						log.Printf("[chat] persist: %v", err)
					}
				}
				writeSSEChat(w, fiber.Map{"type": "done", "content": full})
				return
			}

			assistant := dto.Message{Role: "assistant", Content: full, ToolCalls: toolCalls}
			session.Messages = append(session.Messages, assistant)

			for _, tc := range toolCalls {
				result := chat.ExecuteTool(projectRoot, tc.Function.Name, tc.Function.Arguments)
				writeSSEChat(w, toolEvent(tc.Function.Name, tc.Function.Arguments, result))
				content := result.Content
				if !result.Success {
					content = "ERRO: " + result.Error
					if content == "ERRO: " {
						content = "ERRO: falha na tool " + tc.Function.Name
					}
				}
				session.Messages = append(session.Messages, dto.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    content,
				})
			}
			if err := h.service.PersistTurn(session); err != nil {
				log.Printf("[chat] persist tools: %v", err)
			}
		}
		writeSSEChat(w, fiber.Map{"type": "done", "content": "limite de iterações atingido"})
	})

	return nil
}


func (h *ChatHandler) RenameChat(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if strings.TrimSpace(body.Title) == "" {
		return c.Status(400).SendString("título vazio")
	}
	sess, err := chat.RenameSession(id, body.Title)
	if err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"id": sess.ID, "title": sess.Title})
}

func (h *ChatHandler) DeleteChat(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := chat.DeleteSession(id); err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *ChatHandler) ExportResumo(c *fiber.Ctx) error {
	s, err := chat.LoadSession(c.Params("id"))
	if err != nil {
		return c.Status(404).SendString("chat não encontrado")
	}
	body := chat.BuildResumo(s)
	name := s.Title
	if name == "" {
		name = "chat"
	}
	c.Set("Content-Type", "text/markdown; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizeFileName(name)+".md"))
	return c.SendString(body)
}

func sanitizeFileName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "chat"
	}
	var b strings.Builder
	for _, r := range s {
		if r == '/' || r == '\\' || r == ':' || r < 32 {
			b.WriteByte('-')
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "chat"
	}
	return out
}

func (h *ChatHandler) ExportChat(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Demanda string `json:"demanda"`
	}
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERRO] %v", err.Error())
	}
	path, err := h.service.ExportToIssue(id, body.Demanda)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func writeSSEChat(w *bufio.Writer, payload interface{}) {
	line := fmt.Sprintf("data: %s\n\n", jsonStringChat(payload))
	if _, err := w.WriteString(line); err != nil {
		log.Printf("[ERRO] %v", err)
	}
	_ = w.Flush()
}

func jsonStringChat(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func toolEvent(name, argsJSON string, result chat.ToolResult) fiber.Map {
	path := ""
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args != nil {
		if v, ok := args["path"].(string); ok {
			path = v
		} else if v, ok := args["dir"].(string); ok {
			path = v
		}
	}
	if path == "" && (name == "list_files" || name == "list_dir") {
		path = "."
	}
	kind := "read"
	switch name {
	case "list_files", "list_dir":
		kind = "list"
	case "write_file", "apply_patch":
		kind = "write"
	}
	return fiber.Map{
		"type":    "tool",
		"name":    name,
		"kind":    kind,
		"path":    path,
		"ok":      result.Success,
		"error":   result.Error,
		"bytes":   len(result.Content),
	}
}
