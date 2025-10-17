package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// HistoryEntry - структура для записи истории
type HistoryEntry struct {
	Command   string `json:"command"`
	Timestamp string `json:"timestamp"`
	ID        int    `json:"id"`
}

// CalculatorData - структура для хранения всех данных
type CalculatorData struct {
	Variables map[string]interface{} `json:"variables"`
	History   []HistoryEntry         `json:"history"`
}

type PersistenceManager struct {
	dataFile string
}

func NewPersistenceManager() *PersistenceManager {
	return &PersistenceManager{
		dataFile: "calculator_data.json",
	}
}

func NewPersistenceManagerWithFile(dataFile string) *PersistenceManager {
	return &PersistenceManager{
		dataFile: dataFile,
	}
}

// SaveData - сохранение данных в JSON файл
func (pm *PersistenceManager) SaveData(data *CalculatorData) bool {
	// Добавляем timestamp к каждой команде в истории
	timestampedHistory := make([]HistoryEntry, len(data.History))
	for i, entry := range data.History {
		if entry.Timestamp == "" {
			timestampedHistory[i] = HistoryEntry{
				Command:   entry.Command,
				Timestamp: time.Now().Format(time.RFC3339),
				ID:        i + 1,
			}
		} else {
			timestampedHistory[i] = entry
		}
	}
	data.History = timestampedHistory

	file, err := os.Create(pm.dataFile)
	if err != nil {
		fmt.Printf("Ошибка сохранения: %v\n", err)
		return false
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(data); err != nil {
		fmt.Printf("Ошибка кодирования JSON: %v\n", err)
		return false
	}

	return true
}

// LoadData - загрузка данных из JSON файла
func (pm *PersistenceManager) LoadData() *CalculatorData {
	file, err := os.Open(pm.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует - возвращаем пустые данные
			return &CalculatorData{
				Variables: make(map[string]interface{}),
				History:   make([]HistoryEntry, 0),
			}
		}
		fmt.Printf("Ошибка загрузки: %v\n", err)
		return nil
	}
	defer file.Close()

	var data CalculatorData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		fmt.Printf("Ошибка декодирования JSON: %v\n", err)
		return nil
	}

	// Конвертируем старый формат истории в новый (если нужно)
	pm.migrateHistoryFormat(&data)

	return &data
}

// migrateHistoryFormat - конвертация старого формата истории
func (pm *PersistenceManager) migrateHistoryFormat(data *CalculatorData) {
	if len(data.History) == 0 {
		return
	}

	// Проверяем, есть ли записи в старом формате (просто строки)
	// В Go это сложнее определить, так как типы строгие
	// Вместо этого полагаемся на структуру HistoryEntry

	// Создаем новую историю с правильным форматом
	newHistory := make([]HistoryEntry, 0, len(data.History))
	for i, entry := range data.History {
		// Если запись уже в правильном формате, оставляем как есть
		if entry.Command != "" {
			if entry.Timestamp == "" {
				entry.Timestamp = time.Now().Format(time.RFC3339)
			}
			if entry.ID == 0 {
				entry.ID = i + 1
			}
			newHistory = append(newHistory, entry)
		}
	}
	data.History = newHistory
}

// GetRecentHistory - получение последних N команд из истории
func (pm *PersistenceManager) GetRecentHistory(limit int) []HistoryEntry {
	data := pm.LoadData()
	if data == nil || len(data.History) == 0 {
		return []HistoryEntry{}
	}

	if limit <= 0 || limit > len(data.History) {
		// Возвращаем всю историю (копию)
		historyCopy := make([]HistoryEntry, len(data.History))
		copy(historyCopy, data.History)
		return historyCopy
	}

	// Возвращаем последние limit записей
	start := len(data.History) - limit
	return data.History[start:]
}

// ClearHistory - очистка истории
func (pm *PersistenceManager) ClearHistory() bool {
	data := pm.LoadData()
	if data == nil {
		data = &CalculatorData{
			Variables: make(map[string]interface{}),
			History:   make([]HistoryEntry, 0),
		}
	}

	data.History = []HistoryEntry{}
	return pm.SaveData(data)
}

// SaveVariables - сохранение только переменных
func (pm *PersistenceManager) SaveVariables(variables map[string]interface{}) bool {
	data := pm.LoadData()
	if data == nil {
		data = &CalculatorData{
			Variables: make(map[string]interface{}),
			History:   make([]HistoryEntry, 0),
		}
	}

	data.Variables = variables
	return pm.SaveData(data)
}

// LoadVariables - загрузка только переменных
func (pm *PersistenceManager) LoadVariables() map[string]interface{} {
	data := pm.LoadData()
	if data == nil {
		return make(map[string]interface{})
	}
	return data.Variables
}

// // Пример использования
// func main() {
// 	pm := NewPersistenceManager()

// 	// Тестовые данные
// 	testData := &CalculatorData{
// 		Variables: map[string]interface{}{
// 			"x": 42,
// 			"y": 3.14,
// 			"name": "test",
// 		},
// 		History: []HistoryEntry{
// 			{Command: "2 + 2", Timestamp: time.Now().Format(time.RFC3339), ID: 1},
// 			{Command: "x = 10", Timestamp: time.Now().Format(time.RFC3339), ID: 2},
// 		},
// 	}

// 	// Сохраняем данные
// 	if pm.SaveData(testData) {
// 		fmt.Println("Данные успешно сохранены")
// 	}

// 	// Загружаем данные
// 	loadedData := pm.LoadData()
// 	if loadedData != nil {
// 		fmt.Printf("Загружено переменных: %d\n", len(loadedData.Variables))
// 		fmt.Printf("Загружено записей истории: %d\n", len(loadedData.History))
// 	}

// 	// Получаем последние записи
// 	recent := pm.GetRecentHistory(1)
// 	fmt.Printf("Последняя команда: %s\n", recent[0].Command)
// }
