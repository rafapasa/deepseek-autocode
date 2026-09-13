package dto

// ===== Issue montada (base + projeto + tarefa) =====

type Request struct {
	Demanda   string      `json:"demanda"`
	Estrutura interface{} `json:"estrutura,omitempty"`
	Raiz      string      `json:"raiz"`
	Rules     []string    `json:"rules"`
	Arquivos  []string    `json:"arquivos"`
	Tarefas   []Tarefa    `json:"tarefas"`
}

type Tarefa struct {
	ID        string `json:"id"`
	Arquivo   string `json:"arquivo"`
	Tipo      string `json:"tipo"`
	Descricao string `json:"descricao"`
}
