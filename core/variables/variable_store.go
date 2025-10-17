package variables

// VariableStore - хранилище переменных

type VariableStore struct {
	variables map[string]interface{}
}

func NewVariableStore() *VariableStore {
	return &VariableStore{
		variables: make(map[string]interface{}),
	}
}

// SetVariable - установка переменной
func (vs *VariableStore) SetVariable(name string, value interface{}) {
	vs.variables[name] = value
}

// GetVariable - получение переменной
func (vs *VariableStore) GetVariable(name string) interface{} {
	return vs.variables[name] // В Go возвращает nil если ключа нет
}

// GetVariables - получение копии всех переменных
func (vs *VariableStore) GetVariables() map[string]interface{} {
	// Создаем новую мапу и копируем значения
	copyMap := make(map[string]interface{})
	for k, v := range vs.variables {
		copyMap[k] = v
	}
	return copyMap
}

// SetVariables - установка всех переменных
func (vs *VariableStore) SetVariables(variablesDict map[string]interface{}) {
	// Создаем новую мапу и копируем значения
	vs.variables = make(map[string]interface{})
	for k, v := range variablesDict {
		vs.variables[k] = v
	}
}
