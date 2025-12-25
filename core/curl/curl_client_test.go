package curl

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExecuteGET(t *testing.T) {
	// Создаем mock сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","data":"test"}`))
	}))
	defer server.Close()

	client := NewCurlClient()
	result, err := client.Execute(server.URL, nil)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status=success, got %v", response["status"])
	}
}

func TestExecuteWithCustomHeaders(t *testing.T) {
	expectedUserAgent := "CustomAgent/1.0"
	expectedAuth := "Bearer token123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != expectedUserAgent {
			t.Errorf("Expected User-Agent: %s, got: %s", expectedUserAgent, ua)
		}
		if auth := r.Header.Get("Authorization"); auth != expectedAuth {
			t.Errorf("Expected Authorization: %s, got: %s", expectedAuth, auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewCurlClient()
	headers := map[string]string{
		"User-Agent":    expectedUserAgent,
		"Authorization": expectedAuth,
	}

	result, err := client.Execute(server.URL, headers)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "OK" {
		t.Errorf("Expected 'OK', got '%s'", result)
	}
}

func TestExecuteWithMethod(t *testing.T) {
	tests := []struct {
		method       string
		body         string
		expectedBody string
	}{
		{"POST", `{"key":"value"}`, `{"key":"value"}`},
		{"PUT", `{"updated":"data"}`, `{"updated":"data"}`},
		{"DELETE", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Errorf("Expected %s method, got %s", tt.method, r.Method)
				}

				if tt.expectedBody != "" {
					body := make([]byte, len(tt.expectedBody))
					r.Body.Read(body)
					if string(body) != tt.expectedBody {
						t.Errorf("Expected body: %s, got: %s", tt.expectedBody, string(body))
					}
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Success"))
			}))
			defer server.Close()

			client := NewCurlClient()
			result, err := client.ExecuteWithMethod(tt.method, server.URL, nil, tt.body)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != "Success" {
				t.Errorf("Expected 'Success', got '%s'", result)
			}
		})
	}
}

func TestGetStatusCode(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{"200 OK", http.StatusOK, http.StatusOK},
		{"404 Not Found", http.StatusNotFound, http.StatusNotFound},
		{"500 Server Error", http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewCurlClient()
			status, err := client.GetStatusCode(server.URL, nil)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}

func TestHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewCurlClient()
	_, err := client.Execute(server.URL, nil)

	if err == nil {
		t.Error("Expected error for 500 status code")
	}

	if !strings.Contains(err.Error(), "HTTP ошибка") {
		t.Errorf("Expected HTTP error message, got: %v", err)
	}
}

func TestInvalidURL(t *testing.T) {
	client := NewCurlClient()
	_, err := client.Execute("invalid://url", nil)

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func BenchmarkSimpleGET(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := NewCurlClient()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Execute(server.URL, nil)
	}
}
