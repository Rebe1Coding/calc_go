package ui

import (
	"app/core/interpreter"
	"encoding/json"
	"net/http"
	"path/filepath"
)

type WebInterface struct {
	interpreter *interpreter.Interpreter
}

func NewWebInterface(i *interpreter.Interpreter) *WebInterface {
	return &WebInterface{
		interpreter: i,
	}
}

func (w *WebInterface) Start(addr string) error {
	// API routes
	http.HandleFunc("/api/execute", w.handleExecute)
	http.HandleFunc("/api/vars", w.handleVars)
	http.HandleFunc("/api/history", w.handleHistory)
	http.HandleFunc("/api/clear-history", w.handleClearHistory)
	http.HandleFunc("/api/clear-vars", w.handleClearVars)

	// Serve static files from frontend directory
	fs := http.FileServer(http.Dir(filepath.Join(".", "static")))
	http.Handle("/", fs)

	return http.ListenAndServe(addr, nil)
}

func (w *WebInterface) handleExecute(wr http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(wr, "only POST", 400)
		return
	}

	var req struct {
		Input string `json:"input"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result, err := w.interpreter.Execute(req.Input)
	if err != nil {
		wr.WriteHeader(400)
		json.NewEncoder(wr).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(wr).Encode(map[string]interface{}{"result": result})
}

func (w *WebInterface) handleVars(wr http.ResponseWriter, _ *http.Request) {
	vars := w.interpreter.GetVariables()
	json.NewEncoder(wr).Encode(vars)
}

func (w *WebInterface) handleHistory(wr http.ResponseWriter, _ *http.Request) {
	commands := w.interpreter.GetHistoryCommands(10)
	if len(commands) == 0 {
		json.NewEncoder(wr).Encode([]string{})
		return
	}
	json.NewEncoder(wr).Encode(commands)
}

func (w *WebInterface) handleClearHistory(wr http.ResponseWriter, _ *http.Request) {
	w.interpreter.ClearHistory()
	wr.WriteHeader(200)
}

func (w *WebInterface) handleClearVars(wr http.ResponseWriter, _ *http.Request) {
	// interpreter.ClearVariables() — если есть
	wr.WriteHeader(200)
}
