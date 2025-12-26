package ui

import (
	"app/core/interpreter"
	"app/metrics"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type WebInterface struct {
	interpreter *interpreter.Interpreter
}

func NewWebInterface(i *interpreter.Interpreter) *WebInterface {
	return &WebInterface{
		interpreter: i,
	}
}

// Middleware для метрик
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrapper для захвата статус кода
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next(wrapped, r)

		duration := time.Since(start).Seconds()

		metrics.HttpRequestsTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapped.statusCode),
		).Inc()

		metrics.HttpRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
		).Observe(duration)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (w *WebInterface) Start(addr string) error {
	// API routes с метриками
	http.HandleFunc("/api/execute", metricsMiddleware(w.handleExecute))
	http.HandleFunc("/api/vars", metricsMiddleware(w.handleVars))
	http.HandleFunc("/api/history", metricsMiddleware(w.handleHistory))
	http.HandleFunc("/api/clear-history", metricsMiddleware(w.handleClearHistory))

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static files
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
		metrics.CalculatorOperations.WithLabelValues("error").Inc()
		wr.WriteHeader(400)
		json.NewEncoder(wr).Encode(map[string]string{"error": err.Error()})
		return
	}

	metrics.CalculatorOperations.WithLabelValues("success").Inc()

	// Обновляем метрики
	vars := w.interpreter.GetVariables()
	history := w.interpreter.GetHistoryCommands(1000)
	metrics.UpdateCalculatorMetrics(len(vars), len(history))

	json.NewEncoder(wr).Encode(map[string]interface{}{"result": result})
}

func (w *WebInterface) handleVars(wr http.ResponseWriter, _ *http.Request) {
	vars := w.interpreter.GetVariables()
	metrics.UpdateCalculatorMetrics(len(vars), 0)
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
	history := w.interpreter.GetHistoryCommands(1000)
	metrics.UpdateCalculatorMetrics(0, len(history))
	wr.WriteHeader(200)
}
