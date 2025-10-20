package history

import (
	. "app/core/persistence"
	"strings"
	"time"
)

type HistoryManager struct {
	persistence *PersistenceManager
	maxHistory  int
}

func NewHistoryManager(persistence *PersistenceManager) *HistoryManager {
	return &HistoryManager{
		persistence: persistence,
		maxHistory:  100,
	}
}


// AddCommand - добавление команды в историю с сохранением в JSON
func (hm *HistoryManager) AddCommand(command string) {
	data := hm.persistence.LoadData()
	if data == nil {
		data = &CalculatorData{
			Variables: make(map[string]interface{}),
			History:   make([]HistoryEntry, 0),
		}
	}

	// Создаем запись команды
	commandEntry := HistoryEntry{
		Command:   command,
		Timestamp: time.Now().Format(time.RFC3339),
		ID:        len(data.History) + 1,
	}

	data.History = append(data.History, commandEntry)

	// Ограничиваем размер истории
	if len(data.History) > hm.maxHistory {
		data.History = data.History[len(data.History)-hm.maxHistory:]
	}

	hm.persistence.SaveData(data)
}

// GetHistory - получение истории команд
func (hm *HistoryManager) GetHistory(limit int) []HistoryEntry {
	return hm.persistence.GetRecentHistory(limit)
}

// DetailedHistoryEntry - детализированная запись истории
type DetailedHistoryEntry struct {
	ID        int    `json:"id"`
	Command   string `json:"command"`
	Timestamp string `json:"timestamp"`
	Time      string `json:"time"` // Форматированное время
}

// GetDetailedHistory - получение подробной истории с timestamp
func (hm *HistoryManager) GetDetailedHistory(limit int) []DetailedHistoryEntry {
	history := hm.GetHistory(limit)
	detailed := make([]DetailedHistoryEntry, len(history))

	for i, entry := range history {
		// Парсим timestamp и форматируем
		formattedTime := "unknown"
		if entry.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
				formattedTime = t.Format("2006-01-02 15:04:05")
			}
		}

		detailed[i] = DetailedHistoryEntry{
			ID:        entry.ID,
			Command:   entry.Command,
			Timestamp: entry.Timestamp,
			Time:      formattedTime,
		}
	}

	return detailed
}

// SetHistory - установка истории (для загрузки при старте)
func (hm *HistoryManager) SetHistory(historyList []HistoryEntry) {
	data := hm.persistence.LoadData()
	if data == nil {
		data = &CalculatorData{
			Variables: make(map[string]interface{}),
			History:   make([]HistoryEntry, 0),
		}
	}

	// Форматируем историю
	formattedHistory := make([]HistoryEntry, len(historyList))
	for i, item := range historyList {
		formattedHistory[i] = HistoryEntry{
			Command:   item.Command,
			Timestamp: item.Timestamp,
			ID:        i + 1,
		}
		// Если timestamp пустой, устанавливаем текущее время
		if formattedHistory[i].Timestamp == "" {
			formattedHistory[i].Timestamp = time.Now().Format(time.RFC3339)
		}
	}

	data.History = formattedHistory
	hm.persistence.SaveData(data)
}

// ClearHistory - очистка всей истории
func (hm *HistoryManager) ClearHistory() bool {
	return hm.persistence.ClearHistory()
}

// SearchHistory - поиск по истории команд
func (hm *HistoryManager) SearchHistory(keyword string) []HistoryEntry {
	history := hm.GetHistory(hm.maxHistory)
	results := make([]HistoryEntry, 0)

	for _, entry := range history {
		if strings.Contains(strings.ToLower(entry.Command), strings.ToLower(keyword)) {
			results = append(results, entry)
		}
	}

	return results
}

// GetHistoryCount - получение количества записей в истории
func (hm *HistoryManager) GetHistoryCount() int {
	data := hm.persistence.LoadData()
	if data == nil {
		return 0
	}
	return len(data.History)
}

// GetLastCommand - получение последней команды
func (hm *HistoryManager) GetLastCommand() string {
	history := hm.GetHistory(1)
	if len(history) == 0 {
		return ""
	}
	return history[0].Command
}

