package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rafapasa/deepseek-autocode/cmd/ui"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
	"github.com/rafapasa/deepseek-autocode/internal/service"
)

func main() {
	uiMode := flag.Bool("ui", false, "modo interface web")
	port := flag.String("port", "8080", "porta do servidor web (modo ui)")
	flag.Parse()

	if *uiMode {
		if err := ui.Start(*port); err != nil {
			fmt.Printf("❌ Erro no servidor: %v\n", err)
			os.Exit(1)
		}
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Uso:")
		fmt.Println("  deepseek-autocode <issue.json>")
		fmt.Println("  deepseek-autocode --ui [--port 8080]")
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Printf("❌ Erro ao ler arquivo: %v\n", err)
		os.Exit(1)
	}

	var req dto.Request
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Printf("❌ Erro ao parsear JSON: %v\n", err)
		os.Exit(1)
	}

	o := service.NewOrchestrator()
	if err := o.Start(req); err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Concluído!")
}
