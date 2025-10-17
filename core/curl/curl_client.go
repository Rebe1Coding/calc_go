package curl

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CurlClient - HTTP клиент для выполнения запросов
type CurlClient struct {
	client *http.Client
}

func NewCurlClient() *CurlClient {
	return &CurlClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Execute - выполнение HTTP запроса
func (c *CurlClient) Execute(url string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Устанавливаем заголовки
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Устанавливаем User-Agent по умолчанию если не задан
	if _, exists := headers["User-Agent"]; !exists {
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Calculator-Curl/1.0)")
	}

	response, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP ошибка: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	return string(body), nil
}

// ExecuteWithMethod - выполнение запроса с указанием метода
func (c *CurlClient) ExecuteWithMethod(method, url string, headers map[string]string, body string) (string, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Устанавливаем заголовки
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Устанавливаем Content-Type для POST/PUT запросов
	if (method == "POST" || method == "PUT") && body != "" {
		if _, exists := headers["Content-Type"]; !exists {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	response, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP ошибка: %s", response.Status)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	return string(responseBody), nil
}

// GetStatusCode - выполнение запроса с возвратом статус кода
func (c *CurlClient) GetStatusCode(url string, headers map[string]string) (int, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	response, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer response.Body.Close()

	return response.StatusCode, nil
}
