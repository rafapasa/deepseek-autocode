package dto

// ===== Planejamento (legado — mantido pra compatibilidade) =====

type Part struct {
	ID      int      `json:"id"`
	Demanda string   `json:"demanda"`
	Files   []string `json:"files"`
	Summary string   `json:"summary,omitempty"`
}

type PlanResponse struct {
	Status      string       `json:"status"`
	Parts       []Part       `json:"parts,omitempty"`
	Reason      string       `json:"reason,omitempty"`
	FilesNeeded []FileNeeded `json:"files_needed,omitempty"`
}

type FileNeeded struct {
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty"`
}
