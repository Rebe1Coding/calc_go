package calculator

import (
	"app/models"
	"app/service/calculator/handler"
	"app/service/storage"
	"fmt"
	"strings"
)

type Calculator struct {
	storage   *storage.Storage
	variables map[string]models.Variable
	handlers  []handler.Handler
}

func NewCalculator(storage *storage.Storage) *Calculator {
	calc := &Calculator{
		storage:   storage,
		variables: make(map[string]models.Variable),
		handlers:  make([]handler.Handler, 0),
	}

	calc.loadVariables()
	calc.registerHandlers()

	return calc
}

func (c *Calculator) loadVariables() {
	vars := c.storage.GetAllVariables()
	for name, variable := range vars {
		c.variables[name] = variable
	}
}

func (c *Calculator) registerHandlers() {

	c.handlers = append(c.handlers,
		handler.NewCurlHandler(),
		handler.NewAssignmentHandler(c.variables, c.storage),
		handler.NewArithmeticHandler(c.variables),
	)
}

func (c *Calculator) Execute(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	for _, h := range c.handlers {
		if h.CanHandle(input) {
			result, err := h.Handle(input)

			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			if c.storage != nil {
				c.storage.SaveCommand(input, result)
			}

			return result
		}
	}

	errorMsg := "Error: unknown command format"
	if c.storage != nil {
		c.storage.SaveCommand(input, errorMsg)
	}
	return errorMsg
}

func (c *Calculator) GetLastCommands() []models.Command {
	return c.storage.GetLastCommands(10)
}

func (c *Calculator) GetVariables() map[string]models.Variable {
	return c.variables
}
