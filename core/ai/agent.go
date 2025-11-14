package agent

import (
	"app/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TokenCredential - учетные данные для одного токена
type TokenCredential struct {
	Username string
	Password string
}

// DeepSeekClient - клиент для работы с DeepSeek API через прокси
type DeepSeekClient struct {
	baseURL           string
	credentials       TokenCredential
	currentTokenIndex int
	client            *http.Client
}

func NewDeepSeekClient() *DeepSeekClient {
	config := config.Load()

	return &DeepSeekClient{
		baseURL: config.DeepSeekURL,
		credentials: TokenCredential{
			Username: config.Username,
			Password: config.Password,
		},
		currentTokenIndex: 0,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type TokenStatus struct {
	DailyLimit    int    `json:"daily_limit"`
	Remaining     int    `json:"remaining"`
	RequestsToday int    `json:"requests_today"`
	Username      string `json:"username"`
}

func (d *DeepSeekClient) CheckTokenStatus() (*TokenStatus, error) {
	req, err := http.NewRequest("GET", d.baseURL+"/status", nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}
	fmt.Printf("Проверка статуса токена для пользователя %s\n", d.credentials.Username)
	fmt.Printf("Используем DeepSeek URL: %s\n", d.baseURL)
	fmt.Printf("Используем DeepSeek Username: %s\n", d.credentials.Password)
	req.SetBasicAuth(d.credentials.Username, d.credentials.Password)

	response, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var status TokenStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v\nТело ответа: %s", err, string(body))
	}

	return &status, nil
}

// ClassificationResult - результат классификации запроса
type ClassificationResult struct {
	Type     string `json:"type"`      // "browser|media|curl|general"
	URL      string `json:"url"`       // извлеченный URL
	FilePath string `json:"file_path"` // извлеченный путь к файлу
}

// DeepSeekRequest - структура запроса к API
type DeepSeekRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekResponse - структура ответа от API
type DeepSeekResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (d *DeepSeekClient) ClassifyRequest(userInput string) *ClassificationResult {
	systemPrompt := `Ты - классификатор запросов. Определи тип запроса и извлеки relevant информацию.
Возможные типы:
- browser: запрос на открытие сайта если явно не указан URL например "открой гугл" то определи реальный URL
- media: запрос на открытие медиафайла, извлечи путь к файлу
- curl: запрос на получение данных по URL, извлечи URL
- general: общий запрос

Ответ в формате JSON: {"type": "browser|media|curl|general", "url": "...", "file_path": "..."}`

	response, err := d.makeDeepSeekRequest(systemPrompt, userInput)
	if err != nil {
		fmt.Printf("Ошибка классификации запроса: %v\n", err)
		return &ClassificationResult{Type: "general"}
	}

	// Пытаемся распарсить JSON ответ
	var classification ClassificationResult
	if err := json.Unmarshal([]byte(response), &classification); err != nil {
		fmt.Printf("Ошибка парсинга JSON классификации: %v\n", err)
		fmt.Printf("Ответ от AI: %s\n", response)
		return &ClassificationResult{Type: "general"}
	}

	return &classification
}

// GetContentSummary - получение сводки содержимого
func (d *DeepSeekClient) GetContentSummary(originalQuery, content string) (string, error) {
	// Ограничиваем длину контента
	if len(content) > 2000 {
		content = content[:2000]
	}

	prompt := fmt.Sprintf(`Пользователь запросил: "%s"

Вот содержимое:
%s

Дай краткую содержательную сводку.`, originalQuery, content)

	return d.makeDeepSeekRequest(prompt, "")
}

// GetAIResponse - получение AI ответа на произвольный запрос
func (d *DeepSeekClient) GetAIResponse(userInput string) (string, error) {
	systemPrompt := `Ты - помощник в калькуляторе-интерпретаторе. 
Отвечай кратко и по делу. Если запрос требует вычислений - предложи использовать математические операции.
Если запрос про открытие сайтов или файлов - объясни, что это поддерживается.`

	return d.makeDeepSeekRequest(systemPrompt, userInput)
}

// makeDeepSeekRequest - базовый метод для запросов к DeepSeek
func (d *DeepSeekClient) makeDeepSeekRequest(systemPrompt, userPrompt string) (string, error) {
	payload := DeepSeekRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   1000,
		Temperature: 0.3,
		Stream:      false,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("ошибка маршалинга JSON: %v", err)
	}

	req, err := http.NewRequest("POST", d.baseURL+"/completions", strings.NewReader(string(jsonData)))
	if err != nil {

		fmt.Printf("Ошибка создания запроса с токеном %s: %v\n", d.credentials.Username, err)
	}

	req.SetBasicAuth(d.credentials.Username, d.credentials.Password)
	req.Header.Set("Content-Type", "application/json")

	response, err := d.client.Do(req)
	if err != nil {
		fmt.Printf("Ошибка запроса с токеном %s: %v\n", d.credentials.Username, err)

	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Ошибка чтения ответа с токеном %s: %v\n", d.credentials.Username, err)

	}

	// // Отладочная информация для AI запросов
	// fmt.Printf("AI Response Status: %d\n", response.StatusCode)
	// fmt.Printf("AI Response Body: %s\n", string(body))

	var result DeepSeekResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Ошибка парсинга JSON с токеном %s: %v\n", d.credentials.Username, err)

	}

	if result.Error != nil {
		fmt.Printf("API ошибка с токеном %s: %s\n", d.credentials.Username, result.Error.Message)

	}

	if len(result.Choices) > 0 && result.Choices[0].Message.Content != "" {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("все токены недоступны или исчерпали лимиты")
}
