package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupTestClient() (*DeepSeekClient, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock DeepSeek API responses
		if strings.HasSuffix(r.URL.Path, "/status") {
			response := TokenStatus{
				DailyLimit:    100,
				Remaining:     50,
				RequestsToday: 50,
				Username:      "testuser",
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/completions") {
			// Читаем запрос для определения типа ответа
			var request DeepSeekRequest
			json.NewDecoder(r.Body).Decode(&request)

			response := DeepSeekResponse{
				ID: "test-id",
				Choices: []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				}{
					{
						Message: struct {
							Content string `json:"content"`
						}{
							Content: getMockResponse(request.Messages),
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	// Временно устанавливаем env переменные для теста
	os.Setenv("USER", "testuser")
	os.Setenv("PASSWORD", "testpass")
	os.Setenv("DEEPSEEK_URL", server.URL)

	client := NewDeepSeekClient()
	return client, server
}

func getMockResponse(messages []Message) string {
	if len(messages) == 0 {
		return "General response"
	}

	userMsg := messages[len(messages)-1].Content

	// Классификация
	if strings.Contains(messages[0].Content, "классификатор") {
		if strings.Contains(strings.ToLower(userMsg), "открой") || strings.Contains(strings.ToLower(userMsg), "google") {
			return `{"type":"browser","url":"https://google.com"}`
		}
		if strings.Contains(strings.ToLower(userMsg), "curl") {
			return `{"type":"curl","url":"https://api.example.com"}`
		}
		if strings.Contains(strings.ToLower(userMsg), "позвонить") || strings.Contains(strings.ToLower(userMsg), "call") {
			return `{"type":"call","url":"username","file_path":"audio"}`
		}
		return `{"type":"general"}`
	}

	// Сводка контента
	if strings.Contains(messages[0].Content, "краткую содержательную сводку") {
		return "Это краткая сводка запрошенного контента."
	}

	// Общий ответ
	return "Это ответ AI на ваш запрос."
}

func TestCheckTokenStatus(t *testing.T) {
	client, server := setupTestClient()
	defer server.Close()

	status, err := client.CheckTokenStatus()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.DailyLimit != 100 {
		t.Errorf("Expected DailyLimit=100, got %d", status.DailyLimit)
	}

	if status.Remaining != 50 {
		t.Errorf("Expected Remaining=50, got %d", status.Remaining)
	}

	if status.Username != "testuser" {
		t.Errorf("Expected Username=testuser, got %s", status.Username)
	}
}

func TestClassifyRequest(t *testing.T) {
	client, server := setupTestClient()
	defer server.Close()

	tests := []struct {
		name         string
		input        string
		expectedType string
	}{
		{"Browser request", "открой гугл", "browser"},
		{"Curl request", "curl https://api.example.com", "curl"},
		{"Call request", "позвонить антону", "call"},
		{"General request", "привет как дела", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.ClassifyRequest(tt.input)

			if result.Type != tt.expectedType {
				t.Errorf("Expected type=%s, got type=%s", tt.expectedType, result.Type)
			}
		})
	}
}

func TestGetAIResponse(t *testing.T) {
	client, server := setupTestClient()
	defer server.Close()

	response, err := client.GetAIResponse("Привет, как дела?")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
}

func TestLongContentTruncation(t *testing.T) {
	client, server := setupTestClient()
	defer server.Close()

	// Создаем контент длиннее 2000 символов
	longContent := strings.Repeat("a", 3000)

	summary, err := client.GetContentSummary("test", longContent)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Проверяем что запрос не упал (контент был обрезан до 2000)
	if summary == "" {
		t.Error("Expected non-empty summary for long content")
	}
}

func TestClassificationJSONParsing(t *testing.T) {
	client, server := setupTestClient()
	defer server.Close()

	result := client.ClassifyRequest("открой браузер")

	// Проверяем что JSON успешно распарсился
	if result.Type == "" {
		t.Error("Expected non-empty classification type")
	}
}

func TestAPIErrorHandling(t *testing.T) {
	// Создаем сервер который возвращает ошибку
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/completions") {
			response := DeepSeekResponse{
				Error: &struct {
					Message string `json:"message"`
				}{
					Message: "API Error: Rate limit exceeded",
				},
			}
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer errorServer.Close()

	os.Setenv("DEEPSEEK_URL", errorServer.URL)
	client := NewDeepSeekClient()

	_, err := client.GetAIResponse("test")
	if err == nil {
		t.Error("Expected error from API but got none")
	}

	if !strings.Contains(err.Error(), "недоступны") {
		t.Errorf("Expected 'недоступны' in error, got: %v", err)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/completions") {
			// Возвращаем невалидный JSON с markdown обертками
			w.Write([]byte("```json\n{\"type\":\"browser\"}\n```"))
			return
		}
	}))
	defer invalidServer.Close()

	os.Setenv("DEEPSEEK_URL", invalidServer.URL)
	client := NewDeepSeekClient()

	result := client.ClassifyRequest("test")

	// Должен вернуться general тип при ошибке парсинга
	if result.Type != "general" {
		t.Errorf("Expected 'general' type on JSON parse error, got: %s", result.Type)
	}
}

func BenchmarkClassifyRequest(b *testing.B) {
	client, server := setupTestClient()
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ClassifyRequest("открой браузер")
	}
}

func BenchmarkGetAIResponse(b *testing.B) {
	client, server := setupTestClient()
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetAIResponse("тестовый запрос")
	}
}
