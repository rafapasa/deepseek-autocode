package dto

type FilesNeededResponse struct {
	Status      string        `json:"status"`
	FilesNeeded []FileRequest `json:"files_needed"`
}

type FileRequest struct {
	Path        string `json:"path"`
	Description string `json:"description"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type FinalResponse struct {
	Status string       `json:"status"`
	Files  []FileOutput `json:"files"`
}

type FileOutput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
