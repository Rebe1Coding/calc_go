package handler

import (
	"app/models"
	"regexp"
)

type Handler interface {
	CanHandle(input string) bool
	Handle(input string) (string, error)
}

// Базовые методы для всех handler'ов
type BaseHandler struct {
	Variables map[string]models.Variable
}

func (h *BaseHandler) ReplaceVariables(expression string) string {
	// Твоя логика замены переменных
	re := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	return re.ReplaceAllStringFunc(expression, func(match string) string {
		if variable, exists := h.Variables[match]; exists {
			return variable.Value
		}
		return match
	})
}
