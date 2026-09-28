package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"

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
	_, err := h.service.AddUserMessage(id, body.Content)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	llmMessages, err := h.service.GetLLMMessages(id)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	session, _ := chat.LoadSession(id)
	projectRoot := h.service.ResolveProjectRoot(session.Project)
	tools := chat.GetTools()
	llm := chat.NewChatLLM(h.cfg)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		llmMessages := llmMessages
		for iter := 0; iter < 8; iter++ {
			var full string
			var toolCalls []dto.ToolCall
			err := llm.StreamChat(llmMessages, tools,
				func(delta string) { full += delta; writeSSEChat(w, fiber.Map{"type": "delta", "content": delta}) },
				func(tc dto.ToolCall) { toolCalls = append(toolCalls, tc) },
			)
			if err != nil {
				writeSSEChat(w, fiber.Map{"type": "error", "error": err.Error()})
				return
			}
			if len(toolCalls) == 0 {
				h.service.AddAssistantMessage(id, full)
				writeSSEChat(w, fiber.Map{"type": "done", "content": full})
				return
			}
			// executa tools e alimenta de volta
			for _, tc := range toolCalls {
				result := chat.ExecuteTool(projectRoot, tc.Function.Name, tc.Function.Arguments)
				writeSSEChat(w, fiber.Map{"type": "tool", "name": tc.Function.Name, "result": result})
				content := result.Content
				if !result.Success {
					content = "ERRO: " + result.Error
				}
				llmMessages = append(llmMessages, dto.Message{Role: "assistant", ToolCalls: []dto.ToolCall{tc}})
				llmMessages = append(llmMessages, dto.Message{Role: "tool", ToolCallID: tc.ID, Content: content})
				h.service.AddToolMessage(id, tc.ID, content)
			}
		}
	})
	return nil
}

func (h *ChatHandler) ExportChat(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Demanda string `json:"demanda"`
	}
	c.BodyParser(&body)
	path, err := h.service.ExportToIssue(id, body.Demanda)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *ChatHandler) closeFile(f multipart.File) {
	if err := f.Close(); err != nil {
		log.Printf("[eTools-Code] erro fechando: %v", err)
	}
}

func writeSSEChat(w *bufio.Writer, payload interface{}) {
	line := fmt.Sprintf("data: %s\n\n", jsonStringChat(payload))
	w.WriteString(line)
	w.Flush()
}
func jsonStringChat(v interface{}) string { b, _ := json.Marshal(v); return string(b) }
