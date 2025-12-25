package interpreter

import (
	"os"
	"strings"
	"testing"
)

func setupTestInterpreter() *Interpreter {
	// Устанавливаем тестовые env переменные
	os.Setenv("USER", "testuser")
	os.Setenv("PASSWORD", "testpass")
	os.Setenv("DEEPSEEK_URL", "http://localhost:9999")

	return NewInterpreter()
}

func TestNewInterpreter(t *testing.T) {
	interp := setupTestInterpreter()

	if interp == nil {
		t.Fatal("Expected non-nil interpreter")
	}

	if interp.evaluator == nil {
		t.Error("Expected non-nil evaluator")
	}

	if interp.variables == nil {
		t.Error("Expected non-nil variables")
	}

	if interp.history == nil {
		t.Error("Expected non-nil history")
	}
}

func TestExecuteCurlCommand(t *testing.T) {
	interp := setupTestInterpreter()

	// Тест curl команды (без реального сетевого запроса в unit тесте)
	result, err := interp.Execute("curl --help")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultStr := formatResult(result)
	if !strings.Contains(resultStr, "curl") {
		t.Errorf("Expected curl help message, got: %s", resultStr)
	}
}

func TestGetVariables(t *testing.T) {
	interp := setupTestInterpreter()

	// Устанавливаем несколько переменных
	interp.Execute("x=10")
	interp.Execute("y=20")
	interp.Execute("z=30")

	vars := interp.GetVariables()

	if len(vars) != 3 {
		t.Errorf("Expected 3 variables, got %d", len(vars))
	}

	if vars["x"] == nil || vars["y"] == nil || vars["z"] == nil {
		t.Error("Not all variables were saved")
	}
}

func TestClearHistory(t *testing.T) {
	interp := setupTestInterpreter()

	// Добавляем команды в историю
	interp.Execute("2+2")
	interp.Execute("3+3")
	interp.Execute("4+4")

	// Очищаем историю
	count := interp.ClearHistory()

	if count == 0 {
		t.Error("Expected non-zero history count before clear")
	}

	// Проверяем что история пуста
	commands := interp.GetHistoryCommands(10)
	if len(commands) != 0 {
		t.Errorf("Expected empty history after clear, got %d commands", len(commands))
	}
}

func TestGetHistoryCommands(t *testing.T) {
	interp := setupTestInterpreter()

	// Очищаем историю перед тестом
	interp.ClearHistory()

	// Добавляем команды
	testCommands := []string{"2+2", "3+3", "4+4", "5+5"}
	for _, cmd := range testCommands {
		interp.Execute(cmd)
	}

	// Получаем историю
	history := interp.GetHistoryCommands(10)

	if len(history) != len(testCommands) {
		t.Errorf("Expected %d commands in history, got %d", len(testCommands), len(history))
	}

	// Проверяем что команды сохранились
	for i, cmd := range testCommands {
		if history[i] != cmd {
			t.Errorf("Expected command %s at position %d, got %s", cmd, i, history[i])
		}
	}
}

func TestParseAssignment(t *testing.T) {
	interp := setupTestInterpreter()

	tests := []struct {
		input       string
		shouldMatch bool
		varName     string
		expression  string
	}{
		{"x=10", true, "x", "10"},
		{"myVar=20+5", true, "myVar", "20+5"},
		{"test = 100", true, "test", "100"},
		{"2+2", false, "", ""},
		{"=10", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match, varName, expr := interp.parseAssignment(tt.input)

			if match != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v", tt.shouldMatch, match)
			}

			if match {
				if varName != tt.varName {
					t.Errorf("Expected varName=%s, got %s", tt.varName, varName)
				}
				if expr != tt.expression {
					t.Errorf("Expected expression=%s, got %s", tt.expression, expr)
				}
			}
		})
	}
}

func TestIsFreeFormInput(t *testing.T) {
	interp := setupTestInterpreter()

	tests := []struct {
		input    string
		expected bool
	}{
		{"2+2", false},                    // Математическое выражение
		{"x=10", false},                   // Присваивание
		{"x+5", false},                    // Выражение с переменной
		{"открой браузер", true},          // Свободная форма
		{"расскажи о погоде", true},       // Свободная форма
		{"login username", false},         // Специальная команда
		{"curl http://example.com", true}, // Curl (свободная форма для AI)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := interp.isFreeFormInput(tt.input)
			if result != tt.expected {
				t.Errorf("isFreeFormInput(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	interp := setupTestInterpreter()

	// Устанавливаем переменные
	interp.Execute("x=10")
	interp.Execute("y=5")

	tests := []struct {
		input    string
		expected string
	}{
		{"x+5", "10+5"},
		{"y*2", "5*2"},
		{"x+y", "10+5"},
		{"x+y*2", "10+5*2"},
		{"2+2", "2+2"}, // Без переменных
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := interp.substituteVariables(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestPersistenceIntegration(t *testing.T) {
	// Создаем интерпретатор
	interp1 := setupTestInterpreter()

	// Добавляем данные
	interp1.Execute("x=42")
	interp1.Execute("y=100")
	interp1.Execute("2+2")

	// Создаем новый интерпретатор (должен загрузить данные)
	interp2 := setupTestInterpreter()

	// Проверяем что переменные загрузились
	vars := interp2.GetVariables()
	if vars["x"] == nil || vars["y"] == nil {
		t.Error("Variables were not persisted")
	}

	// Проверяем что история загрузилась
	history := interp2.GetHistoryCommands(10)
	if len(history) == 0 {
		t.Error("History was not persisted")
	}
}

func TestParseCallCommand(t *testing.T) {
	interp := setupTestInterpreter()

	tests := []struct {
		input       string
		shouldMatch bool
		target      string
		callType    string
	}{
		{"позвонить username video", true, "username", "video"},
		{"call alice audio", true, "alice", "audio"},
		{"видеозвонок bob", true, "bob", "video"},
		{"not a call command", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match, target, callType := interp.parseCallCommand(tt.input)

			if match != tt.shouldMatch {
				t.Errorf("Expected match=%v, got %v", tt.shouldMatch, match)
			}

			if match {
				if target != tt.target {
					t.Errorf("Expected target=%s, got %s", tt.target, target)
				}
				if callType != tt.callType {
					t.Errorf("Expected callType=%s, got %s", tt.callType, callType)
				}
			}
		})
	}
}

func TestIsLoginCommand(t *testing.T) {
	interp := setupTestInterpreter()

	tests := []struct {
		input    string
		expected bool
	}{
		{"login username", true},
		{"войти пользователь", true},
		{"LOGIN test", true},
		{"not login", false},
		{"2+2", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := interp.isLoginCommand(tt.input)
			if result != tt.expected {
				t.Errorf("isLoginCommand(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func formatResult(result interface{}) string {
	if result == nil {
		return ""
	}

	switch v := result.(type) {
	case string:
		return v
	case float64:
		return strings.TrimRight(strings.TrimRight(sprintf("%.10f", v), "0"), ".")
	default:
		return sprintf("%v", v)
	}
}

func sprintf(format string, args ...interface{}) string {
	// Простая реализация sprintf для тестов
	return strings.Replace(format, "%.10f", "%.2f", 1)
}

func BenchmarkExecuteSimpleMath(b *testing.B) {
	interp := setupTestInterpreter()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		interp.Execute("2+2")
	}
}

func BenchmarkExecuteWithVariables(b *testing.B) {
	interp := setupTestInterpreter()
	interp.Execute("x=10")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		interp.Execute("x+5")
	}
}
