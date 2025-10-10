package storage

import (
	"app/models"
	"encoding/json"
	"os"
	"time"
)

type Storage struct {
	commands  []models.Command
	variables map[string]models.Variable
	filePath  string
}

func NewStorage(filePath string) *Storage {
	storage := &Storage{
		commands:  make([]models.Command, 0),
		variables: make(map[string]models.Variable),
		filePath:  filePath,
	}
	storage.loadFromFile()
	return storage
}

func (s *Storage) SaveCommand(command, result string) {
	cmd := models.Command{
		ID:        len(s.commands) + 1,
		Command:   command,
		Result:    result,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	s.commands = append(s.commands, cmd)
	s.saveToFile()
}

func (s *Storage) GetLastCommands(limit int) []models.Command {
	if len(s.commands) == 0 {
		return []models.Command{}
	}

	start := len(s.commands) - limit
	if start < 0 {
		start = 0
	}
	return s.commands[start:]
}

func (s *Storage) SetVariable(name, value, varType string) {
	s.variables[name] = models.Variable{
		Name:  name,
		Value: value,
		Type:  varType,
	}
	s.saveToFile()
}

func (s *Storage) GetVariable(name string) (models.Variable, bool) {
	variable, exists := s.variables[name]
	return variable, exists
}

func (s *Storage) GetAllVariables() map[string]models.Variable {
	return s.variables
}

func (s *Storage) loadFromFile() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}

	var state struct {
		Commands  []models.Command           `json:"commands"`
		Variables map[string]models.Variable `json:"variables"`
	}

	if err := json.Unmarshal(data, &state); err == nil {
		s.commands = state.Commands
		s.variables = state.Variables
	}
}

func (s *Storage) saveToFile() {
	state := struct {
		Commands  []models.Command           `json:"commands"`
		Variables map[string]models.Variable `json:"variables"`
	}{
		Commands:  s.commands,
		Variables: s.variables,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(s.filePath, data, 0644)
}
