package calculator

import (
	"calculator/models"
	"calculator/service/storage"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Calculator struct {
	storage   *storage.Storage
	variables map[string]models.Variable
}

func NewCalculator(storage *storage.Storage) *Calculator {
	calc := &Calculator{
		storage:   storage,
		variables: make(map[string]models.Variable),
	}
	calc.loadVariables()
	return calc
}

func (c *Calculator) loadVariables() {
	vars := c.storage.GetAllVariables()
	for name, variable := range vars {
		c.variables[name] = variable
	}
}
func (c *Calculator) Execute(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	var result string
	var err error

	// Обработка curl команды
	if strings.HasPrefix(input, "curl") {
		result = c.handleCurl(input)
	} else if strings.Contains(input, "=") { // ← ELSE IF БЛЯТЬ!
		result = c.handleAssignment(input)
		err = nil
	} else {
		result = c.handleArithmetic(input)
	}

	// Единая обработка ошибок
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}

	if c.storage != nil {
		c.storage.SaveCommand(input, result)
	}
	return result
}

func (c *Calculator) handleCurl(input string) string {
	// Убираем "curl" и парсим аргументы
	args := strings.Fields(input)[1:]
	if len(args) == 0 {
		return "Error: URL required for curl"
	}

	url := args[0]
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}

	// Обработка дополнительных аргументов
	for i := 1; i < len(args); i++ {
		if args[i] == "-H" && i+1 < len(args) {
			header := strings.Split(args[i+1], ":")
			if len(header) == 2 {
				req.Header.Add(strings.TrimSpace(header[0]), strings.TrimSpace(header[1]))
			}
			i++
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}

	// Обрезаем длинный ответ для удобства
	if len(body) > 500 {
		body = append(body[:500], []byte("... [truncated]")...)
	}

	return string(body)
}

func (c *Calculator) handleAssignment(input string) string {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "Error: invalid assignment syntax"
	}

	varName := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Проверяем валидность имени переменной
	if !isValidVariableName(varName) {
		return "Error: invalid variable name"
	}

	// Определяем тип значения
	varType := "string"
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		varType = "number"
	} else if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		value = value[1 : len(value)-1]
	}

	c.variables[varName] = models.Variable{
		Name:  varName,
		Value: value,
		Type:  varType,
	}
	c.storage.SetVariable(varName, value, varType)

	return fmt.Sprintf("Variable '%s' set to '%s'", varName, value)
}

func (c *Calculator) handleArithmetic(input string) string {
	// Заменяем переменные на их значения
	expression := c.replaceVariables(input)

	// Убираем кавычки для вычислений
	expression = strings.ReplaceAll(expression, "\"", "")

	result, err := c.evaluateExpression(expression)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return result
}

func (c *Calculator) replaceVariables(expression string) string {
	re := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	return re.ReplaceAllStringFunc(expression, func(match string) string {
		if variable, exists := c.variables[match]; exists {
			return variable.Value
		}
		return match
	})
}

func (c *Calculator) evaluateExpression(expression string) (string, error) {
	// Простая реализация арифметических операций
	// В реальном приложении стоит использовать парсер выражений

	tokens := strings.Fields(expression)
	if len(tokens) == 1 {
		// Одиночное число или строка
		return tokens[0], nil
	}

	if len(tokens) != 3 {
		return "", fmt.Errorf("invalid expression format")
	}

	a, err := strconv.ParseFloat(tokens[0], 64)
	if err != nil {
		return "", fmt.Errorf("invalid number: %s", tokens[0])
	}

	b, err := strconv.ParseFloat(tokens[2], 64)
	if err != nil {
		return "", fmt.Errorf("invalid number: %s", tokens[2])
	}

	var result float64
	switch tokens[1] {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		if b == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return "", fmt.Errorf("unknown operator: %s", tokens[1])
	}

	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func isValidVariableName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

func (c *Calculator) GetLastCommands() []models.Command {
	return c.storage.GetLastCommands(10)
}

func (c *Calculator) GetVariables() map[string]models.Variable {
	return c.variables
}
