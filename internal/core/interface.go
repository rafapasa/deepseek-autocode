package core

type Llminterface interface {
	Chat(prompt string) (string, error)
}
