package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"mime"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rafapasa/deepseek-autocode/internal/config"
)

//go:embed all:templates
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

var indexHTML []byte

func init() {
	data, err := fs.ReadFile(templatesFS, "templates/index.html")
	if err != nil {
		indexHTML = []byte("<h1>erro ao carregar template</h1>")
	} else {
		indexHTML = data
	}
}

func Start(cfg *config.Config) error {
	app := fiber.New(fiber.Config{AppName: "eTools-Code"})
	app.Use(logger.New())
	app.Use(cors.New())
	h := NewHandler(cfg)
	ch := NewChatHandler(cfg)

	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		data, err := fs.ReadFile(staticFS, "static/favicon.ico")
		if err == nil {
			c.Set("Content-Type", "image/x-icon")
			c.Set("Cache-Control", "public, max-age=86400")
			return c.Send(data)
		}
		data, err = fs.ReadFile(staticFS, "static/logo.png")
		if err == nil {
			c.Set("Content-Type", "image/png")
			return c.Send(data)
		}
		return c.SendStatus(204)
	})

	app.Get("/static/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		data, err := fs.ReadFile(staticFS, "static/"+path)
		if err != nil {
			return c.Status(404).SendString("static not found: " + path)
		}
		if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
			c.Set("Content-Type", ct)
		} else if strings.HasSuffix(path, ".css") {
			c.Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(path, ".js") {
			c.Set("Content-Type", "application/javascript; charset=utf-8")
		}
		c.Set("Cache-Control", "no-cache")
		return c.Send(data)
	})

	app.Get("/", h.Index)
	api := app.Group("/api")
	api.Get("/config", h.GetConfig)
	api.Post("/config", h.SaveConfig)
	api.Get("/env", h.CheckEnv)
	api.Get("/issues", h.ListIssues)
	api.Get("/projects", h.ListProjects)
	api.Get("/base", h.GetBase)
	api.Post("/base", h.SaveBase)
	api.Post("/base/upload", h.UploadBase)
	api.Get("/projeto/:project", h.GetProjeto)
	api.Post("/projeto/:project", h.SaveProjeto)
	api.Post("/projeto/:project/upload", h.UploadProjeto)
	api.Post("/issues/create", h.CreateIssue)
	api.Post("/issues/upload", h.UploadIssue)
	api.Post("/run", h.Run)
	api.Get("/stream/:id", h.Stream)
	api.Post("/stop", h.Stop)
	api.Get("/file/:project/*", h.GetIssueFile)
	api.Post("/file/:project/*", h.SaveIssueFile)
	api.Post("/file/:project/*/upload", h.UploadIssueReplace)

	chatApi := api.Group("/chat")
	chatApi.Post("/new", ch.CreateChat)
	chatApi.Get("/", ch.ListChats)
	chatApi.Get("/:id", ch.GetChat)
	chatApi.Post("/:id/message", ch.PostMessage)
	chatApi.Post("/:id/export", ch.ExportChat)

	fmt.Printf("[eTools-Code] http://localhost:%d - ISSUES_DIR=%s\n", cfg.HttpPort, cfg.IssuesDir)
	return app.Listen(fmt.Sprintf(":%d", cfg.HttpPort))
}
