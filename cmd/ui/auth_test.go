package ui

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newTestApp(t *testing.T) *fiber.App {
	t.Setenv("DSAC_UI_AUTH", "")
	t.Setenv("DSAC_UI_PASSWORD_HASH", "")
	t.Setenv("DSAC_UI_PASSWORD", "segredo123")
	a, err := NewAuthFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	a.Register(app)
	app.Use(a.Middleware())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/api/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func login(t *testing.T, app *fiber.App, pw string) (int, string) {
	req := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"password": {pw}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookie {
			return resp.StatusCode, c.Value
		}
	}
	return resp.StatusCode, ""
}

func TestAuthMiddleware(t *testing.T) {
	app := newTestApp(t)

	resp, _ := app.Test(httptest.NewRequest("GET", "/api/x", nil))
	if resp.StatusCode != 401 {
		t.Fatalf("api sem sessão: esperado 401, veio %d", resp.StatusCode)
	}
	resp, _ = app.Test(httptest.NewRequest("GET", "/", nil))
	if resp.StatusCode != 303 {
		t.Fatalf("página sem sessão: esperado redirect, veio %d", resp.StatusCode)
	}

	if code, tok := login(t, app, "errada"); code != 401 || tok != "" {
		t.Fatalf("senha errada deveria falhar: %d", code)
	}
	code, tok := login(t, app, "segredo123")
	if code != 303 || tok == "" {
		t.Fatalf("login válido falhou: %d", code)
	}
	req := httptest.NewRequest("GET", "/api/x", nil)
	req.Header.Set("Cookie", sessionCookie+"="+tok)
	resp, _ = app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("com sessão: esperado 200, veio %d", resp.StatusCode)
	}
}

func TestAuthRateLimit(t *testing.T) {
	app := newTestApp(t)
	for i := 0; i < loginMaxFails; i++ {
		login(t, app, "errada")
	}
	if code, _ := login(t, app, "segredo123"); code != 429 {
		t.Fatalf("esperado 429 após %d falhas, veio %d", loginMaxFails, code)
	}
}

func TestAuthFailClosed(t *testing.T) {
	t.Setenv("DSAC_UI_AUTH", "")
	t.Setenv("DSAC_UI_PASSWORD_HASH", "")
	t.Setenv("DSAC_UI_PASSWORD", "")
	if _, err := NewAuthFromEnv(); err == nil {
		t.Fatal("sem senha deveria recusar iniciar")
	}
	h, _ := HashPassword("segredo123")
	t.Setenv("DSAC_UI_PASSWORD_HASH", h)
	a, err := NewAuthFromEnv()
	if err != nil || !a.checkPassword("segredo123") || a.checkPassword("x") {
		t.Fatal("hash bcrypt não validou corretamente")
	}
}
