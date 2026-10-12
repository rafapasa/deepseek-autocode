package ui

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// Autenticação da UI por senha única.
//
//	DSAC_UI_PASSWORD_HASH  hash bcrypt (preferido; gere com: deepseek-autocode --hash-password)
//	DSAC_UI_PASSWORD       senha em texto (fallback)
//	DSAC_UI_AUTH=off       desativa (só para desenvolvimento local)
const (
	sessionCookie   = "dsac_session"
	sessionTTL      = 12 * time.Hour
	loginMaxFails   = 5
	loginFailWindow = time.Minute
)

type Auth struct {
	enabled  bool
	hash     []byte
	plain    []byte
	mu       sync.Mutex
	sessions map[string]time.Time
	fails    map[string][]time.Time
}

// NewAuthFromEnv falha (fail closed) se nenhuma senha estiver configurada.
func NewAuthFromEnv() (*Auth, error) {
	a := &Auth{sessions: map[string]time.Time{}, fails: map[string][]time.Time{}}
	if strings.EqualFold(os.Getenv("DSAC_UI_AUTH"), "off") {
		log.Printf("[auth] ATENÇÃO: autenticação da UI desativada (DSAC_UI_AUTH=off)")
		return a, nil
	}
	if h := strings.TrimSpace(os.Getenv("DSAC_UI_PASSWORD_HASH")); h != "" {
		if _, err := bcrypt.Cost([]byte(h)); err != nil {
			return nil, fmt.Errorf("DSAC_UI_PASSWORD_HASH inválido (esperado bcrypt): %v", err)
		}
		a.hash = []byte(h)
	} else if p := os.Getenv("DSAC_UI_PASSWORD"); p != "" {
		a.plain = []byte(p)
	} else {
		return nil, fmt.Errorf("senha da UI não configurada: defina DSAC_UI_PASSWORD_HASH (gere com --hash-password) ou DSAC_UI_PASSWORD; ou DSAC_UI_AUTH=off para desativar")
	}
	a.enabled = true
	return a, nil
}

func (a *Auth) checkPassword(pw string) bool {
	if a.hash != nil {
		return bcrypt.CompareHashAndPassword(a.hash, []byte(pw)) == nil
	}
	return subtle.ConstantTimeCompare(a.plain, []byte(pw)) == 1
}

func (a *Auth) newSession() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, exp := range a.sessions {
		if now.After(exp) {
			delete(a.sessions, k)
		}
	}
	a.sessions[tok] = now.Add(sessionTTL)
	return tok, nil
}

func (a *Auth) validSession(tok string) bool {
	if tok == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[tok]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(a.sessions, tok)
		return false
	}
	return true
}

// rateLimited informa se o IP excedeu as falhas na janela.
func (a *Auth) rateLimited(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	cut := time.Now().Add(-loginFailWindow)
	var recent []time.Time
	for _, t := range a.fails[ip] {
		if t.After(cut) {
			recent = append(recent, t)
		}
	}
	a.fails[ip] = recent
	return len(recent) >= loginMaxFails
}

func (a *Auth) registerFail(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.fails[ip] = append(a.fails[ip], time.Now())
}

// clientIP usa X-Forwarded-For só quando a conexão vem de loopback (proxy local, ex. Caddy).
func clientIP(c *fiber.Ctx) string {
	ip := c.IP()
	if parsed := net.ParseIP(ip); parsed != nil && parsed.IsLoopback() {
		if ips := c.IPs(); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return ip
}

func isSecure(c *fiber.Ctx) bool {
	return c.Protocol() == "https" || strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
}

// Middleware protege todas as rotas exceto /login.
func (a *Auth) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !a.enabled || c.Path() == "/login" {
			return c.Next()
		}
		if a.validSession(c.Cookies(sessionCookie)) {
			return c.Next()
		}
		if strings.HasPrefix(c.Path(), "/api") || c.Method() != fiber.MethodGet {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "não autenticado"})
		}
		return c.Redirect("/login", fiber.StatusSeeOther)
	}
}

func (a *Auth) Register(app *fiber.App) {
	app.Get("/login", func(c *fiber.Ctx) error {
		return sendLoginPage(c, "")
	})
	app.Post("/login", func(c *fiber.Ctx) error {
		if !a.enabled {
			return c.Redirect("/", fiber.StatusSeeOther)
		}
		ip := clientIP(c)
		if a.rateLimited(ip) {
			log.Printf("[auth] login bloqueado por excesso de tentativas ip=%s", ip)
			c.Status(fiber.StatusTooManyRequests)
			return sendLoginPage(c, "Muitas tentativas. Aguarde 1 minuto.")
		}
		if !a.checkPassword(c.FormValue("password")) {
			a.registerFail(ip)
			log.Printf("[auth] senha incorreta ip=%s", ip)
			c.Status(fiber.StatusUnauthorized)
			return sendLoginPage(c, "Senha incorreta.")
		}
		tok, err := a.newSession()
		if err != nil {
			return c.Status(500).SendString("erro ao criar sessão")
		}
		c.Cookie(&fiber.Cookie{
			Name: sessionCookie, Value: tok, Path: "/",
			Expires: time.Now().Add(sessionTTL), HTTPOnly: true,
			SameSite: fiber.CookieSameSiteLaxMode, Secure: isSecure(c),
		})
		log.Printf("[auth] login ok ip=%s", ip)
		return c.Redirect("/", fiber.StatusSeeOther)
	})
	logout := func(c *fiber.Ctx) error {
		a.mu.Lock()
		delete(a.sessions, c.Cookies(sessionCookie))
		a.mu.Unlock()
		c.Cookie(&fiber.Cookie{Name: sessionCookie, Value: "", Path: "/", Expires: time.Unix(0, 0), HTTPOnly: true})
		return c.Redirect("/login", fiber.StatusSeeOther)
	}
	app.Get("/logout", logout)
	app.Post("/logout", logout)
}

func sendLoginPage(c *fiber.Ctx, msg string) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	errHTML := ""
	if msg != "" {
		errHTML = `<p class="err">` + html.EscapeString(msg) + `</p>`
	}
	return c.SendString(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>eTools-Code - Login</title>
<style>body{font-family:system-ui,sans-serif;background:#111;color:#eee;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
form{background:#1e1e1e;padding:2rem;border-radius:8px;min-width:280px}input,button{width:100%;padding:.6rem;margin-top:.8rem;box-sizing:border-box;border-radius:4px;border:1px solid #444}
input{background:#2a2a2a;color:#eee}button{background:#3b82f6;color:#fff;border:0;cursor:pointer}.err{color:#f87171}</style></head>
<body><form method="POST" action="/login"><h2>eTools-Code</h2>` + errHTML + `
<input type="password" name="password" placeholder="Senha" autofocus required>
<button type="submit">Entrar</button></form></body></html>`)
}

// HashPassword gera hash bcrypt para DSAC_UI_PASSWORD_HASH.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}
