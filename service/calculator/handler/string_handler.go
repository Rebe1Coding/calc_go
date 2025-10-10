package handler

import (
	"app/models"
	"regexp"
	"strings"
)

type StringHandler struct {
	variables map[string]models.Variable
}

func NewStringHandler(variables map[string]models.Variable) *StringHandler {
	return &StringHandler{variables: variables}
}

func (h *StringHandler) CanHandle(input string) bool {
	// Обрабатываем строковые операции, например конкатенацию
	return strings.Contains(input, "++") ||
		(strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\""))
}

func (h *StringHandler) Handle(input string) (string, error) {
	// Заменяем переменные
	expression := h.replaceVariables(input)

	// Обрабатываем конкатенацию
	if strings.Contains(expression, "++") {
		parts := strings.Split(expression, "++")
		for i := range parts {
			parts[i] = strings.Trim(parts[i], "\" ")
		}
		return strings.Join(parts, ""), nil
	}

	// Возвращаем строку без кавычек
	return strings.Trim(expression, "\""), nil
}

func (h *StringHandler) replaceVariables(expression string) string {
	// Та же логика замены переменных
	re := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	return re.ReplaceAllStringFunc(expression, func(match string) string {
		if variable, exists := h.variables[match]; exists {
			return variable.Value
		}
		return match
	})
}
