package models

// Command представляет выполненную команду
type Command struct {
	ID        int    `json:"id"`
	Command   string `json:"command"`
	Result    string `json:"result"`
	Timestamp string `json:"timestamp"`
}

// Variable представляет переменную
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"` // "number", "string"
}

// HTTPRequest представляет HTTP запрос
type HTTPRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}
