package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rafapasa/deepseek-autocode/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: deepseek-autocode arquivo.json")
		os.Exit(1)
	}

	jsonFile := os.Args[1]

	data, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		fmt.Printf("❌ Erro ao ler arquivo: %v\n", err)
		os.Exit(1)
	}

	var req service.Request
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
