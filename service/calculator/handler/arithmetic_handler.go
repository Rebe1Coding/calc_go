package handler

import (
	"app/models"
	"fmt"
	"strconv"
	"strings"
)

type ArithmeticHandler struct {
	BaseHandler
}

func NewArithmeticHandler(variables map[string]models.Variable) *ArithmeticHandler {
	return &ArithmeticHandler{
		BaseHandler: BaseHandler{Variables: variables},
	}
}

func (h *ArithmeticHandler) CanHandle(input string) bool {
	operators := []string{"+", "-", "*", "/"}
	for _, op := range operators {
		if strings.Contains(input, op) {
			return true
		}
	}

	_, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
	return err == nil
}

func (h *ArithmeticHandler) Handle(input string) (string, error) {
	expression := h.ReplaceVariables(input)
	expression = strings.ReplaceAll(expression, "\"", "")
	return h.evaluateExpression(expression)
}

func (h *ArithmeticHandler) evaluateExpression(expression string) (string, error) {
	tokens := strings.Fields(expression)
	if len(tokens) == 1 {
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
