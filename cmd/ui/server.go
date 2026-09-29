package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rafapasa/deepseek-autocode/internal/config"
)

var (
	//go:embed all:templates
	templatesFS embed.FS

	//go:embed all:static
	staticFS embed.FS

	indexHTML []byte
	staticSub fs.FS
)

func init() {
	var err error
	// Descasca a pasta raiz "static" do FS embutido
	staticSub, err = fs.Sub(staticFS, "static")
	if err != nil {
		panic("falha ao inicializar sub-filesystem estatico: " + err.Error())
	}

	data, err := fs.ReadFile(templatesFS, "templates/index.html")
	if err != nil {
		indexHTML = []byte("<h1>erro ao carregar template</h1>")
	} else {
		indexHTML = data
	}
}

func Start(cfg *config.Config) error {
	// Configuração resiliente do Fiber HTTP Server
	app := fiber.New(fiber.Config{
		AppName:      "eTools-Code",
		BodyLimit:    10 * 1024 * 1024,  // Limite máximo de 10 MB para uploads e JSONs
		ReadTimeout:  30 * time.Second,  // Timeout para leitura dos cabeçalhos/body da requisição
		WriteTimeout: 0,                 // 0 = Sem timeout de escrita (indispensável para SSE / Stream de LLM)
		IdleTimeout:  120 * time.Second, // Timeout de conexões ociosas
	})

	app.Use(logger.New())

	// CORS configurado explicitamente
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	h := NewHandler(cfg)
	ch := NewChatHandler(cfg)

	// Rota estática manual garantindo Content-Type correto e caminho saneado
	app.Get("/static/*", func(c *fiber.Ctx) error {
		rawPath := strings.TrimPrefix(c.Params("*"), "/")
		cleanPath := path.Clean(rawPath)

		data, err := fs.ReadFile(staticSub, cleanPath)
		if err != nil {
			return c.Status(404).SendString("static not found: " + cleanPath)
		}

		// Força os tipos MIME explicitamente
		switch {
		case strings.HasSuffix(cleanPath, ".css"):
			c.Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".js"):
			c.Set("Content-Type", "application/javascript; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".png"):
			c.Set("Content-Type", "image/png")
		case strings.HasSuffix(cleanPath, ".ico"):
			c.Set("Content-Type", "image/x-icon")
		default:
			if ct := mime.TypeByExtension(filepath.Ext(cleanPath)); ct != "" {
				c.Set("Content-Type", ct)
			}
		}

		c.Set("Cache-Control", "no-cache")
		return c.Send(data)
	})

	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		data, err := fs.ReadFile(staticSub, "favicon.ico")
		if err == nil {
			c.Set("Content-Type", "image/x-icon")
			c.Set("Cache-Control", "public, max-age=86400")
			return c.Send(data)
		}
		data, err = fs.ReadFile(staticSub, "logo.png")
		if err == nil {
			c.Set("Content-Type", "image/png")
			return c.Send(data)
		}
		return c.SendStatus(204)
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
