// hashpw gera o hash bcrypt para DSAC_UI_PASSWORD_HASH.
// Uso: make hash-password   (ou: go run ./cmd/hashpw) e digite a senha.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rafapasa/deepseek-autocode/cmd/ui"
)

func main() {
	fmt.Fprint(os.Stderr, "Senha da UI: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stderr, "erro ao ler senha:", err)
		os.Exit(1)
	}
	pw := strings.TrimRight(line, "\r\n")
	if len(pw) < 8 {
		fmt.Fprintln(os.Stderr, "use pelo menos 8 caracteres")
		os.Exit(1)
	}
	h, err := ui.HashPassword(pw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "\nAdicione ao .env (aspas simples por causa do $):")
	fmt.Printf("DSAC_UI_PASSWORD_HASH='%s'\n", h)
}
