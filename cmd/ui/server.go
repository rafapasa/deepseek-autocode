package ui

import (
	_ "embed"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rafapasa/deepseek-autocode/internal/config"
)

//go:embed index.html
var indexHTML []byte

func Start(cfg *config.Config) error {
	app := fiber.New(fiber.Config{AppName: "eTools-Code"})
	app.Use(logger.New())
	app.Use(cors.New())
	h := NewHandler(cfg)
	ch := NewChatHandler(cfg)

	app.Get("/", h.Index)
	api := app.Group("/api")
	api.Get("/config", h.GetConfig)
	api.Post("/config", h.SaveConfig)
	api.Get("/env", h.CheckEnv)
	api.Get("/issues", h.ListIssues)
	api.Post("/issues/create", h.CreateIssue)
	api.Post("/issues/upload", h.UploadIssue)
	api.Post("/run", h.Run)
	api.Get("/stream/:id", h.Stream)
	api.Post("/stop", h.Stop)

	chatApi := api.Group("/chat")
	chatApi.Post("/new", ch.CreateChat)
	chatApi.Get("/", ch.ListChats)
	chatApi.Get("/:id", ch.GetChat)
	chatApi.Post("/:id/message", ch.PostMessage)
	chatApi.Post("/:id/export", ch.ExportChat)

	fmt.Printf("[eTools-Code] http://localhost:%d\n", cfg.HttpPort)
	return app.Listen(fmt.Sprintf(":%d", cfg.HttpPort))
}
