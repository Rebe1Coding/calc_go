package handler

import (
	"app/models"
	"app/service/storage"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type AssignmentHandler struct {
	BaseHandler
	Storage *storage.Storage
}

func NewAssignmentHandler(variables map[string]models.Variable, storage *storage.Storage) *AssignmentHandler {
	return &AssignmentHandler{
		BaseHandler: BaseHandler{Variables: variables},
		Storage:     storage,
	}
}

func (h *AssignmentHandler) CanHandle(input string) bool {
	return strings.Contains(input, "=")
}

func (h *AssignmentHandler) Handle(input string) (string, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid assignment syntax")
	}

	varName := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if !isValidVariableName(varName) {
		return "", fmt.Errorf("invalid variable name")
	}

	varType := "string"
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		varType = "number"
	} else if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		value = value[1 : len(value)-1]
	}

	h.Variables[varName] = models.Variable{
		Name:  varName,
		Value: value,
		Type:  varType,
	}

	if h.Storage != nil {
		h.Storage.SetVariable(varName, value, varType)
	}

	return fmt.Sprintf("Variable '%s' set to '%s'", varName, value), nil
}

func isValidVariableName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}
