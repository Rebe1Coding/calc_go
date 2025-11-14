package interpreter

import (
	agent "app/core/ai"
	"app/core/applauncher"
	"app/core/curl"
	"app/core/evaluator"
	"app/core/history"
	"app/core/persistence"
	"app/core/variables"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Interpreter struct {
	evaluator      *evaluator.Evaluator
	variables      *variables.VariableStore
	persistence    *persistence.PersistenceManager
	history        *history.HistoryManager
	curlClient     *curl.CurlClient
	deepseekClient *agent.DeepSeekClient
	appLauncher    *applauncher.AppLauncher
}

func NewInterpreter() *Interpreter {
	interpreter := &Interpreter{
		evaluator:      evaluator.NewEvaluator(),
		variables:      variables.NewVariableStore(),
		persistence:    persistence.NewPersistenceManager(),
		curlClient:     curl.NewCurlClient(),
		deepseekClient: agent.NewDeepSeekClient(),
		appLauncher:    applauncher.NewAppLauncher(),
	}

	interpreter.history = history.NewHistoryManager(interpreter.persistence)

	interpreter.loadState()

	return interpreter
}

// loadState - загрузка предыдущего состояния из JSON
func (i *Interpreter) loadState() {
	data := i.persistence.LoadData()
	if data != nil {
		// Загружаем историю
		if len(data.History) > 0 {
			i.history.SetHistory(data.History)
		}

		// Загружаем переменные
		if len(data.Variables) > 0 {
			i.variables.SetVariables(data.Variables)
		}

		// Показываем последние 10 команд при старте
		recentHistory := i.history.GetDetailedHistory(10)
		if len(recentHistory) > 0 {
			fmt.Println("Последние 10 команд из истории:")
			fmt.Println(strings.Repeat("-", 50))
			for _, entry := range recentHistory {
				fmt.Printf("%3d. [%s] %s\n", entry.ID, entry.Time, entry.Command)
			}
			fmt.Println(strings.Repeat("-", 50))
		}
	}
}

// saveState - сохранение текущего состояния в JSON
func (i *Interpreter) saveState() {
	data := &persistence.CalculatorData{
		History:   i.history.GetHistory(1000), // Сохраняем до 1000 последних записей
		Variables: i.variables.GetVariables(),
	}
	i.persistence.SaveData(data)
}

// Execute - выполнение введенной команды
func (i *Interpreter) Execute(inputStr string) (interface{}, error) {
	// Обработка curl команд
	if strings.HasPrefix(strings.TrimSpace(inputStr), "curl ") {
		urlArgs := strings.TrimSpace(inputStr[4:])
		return i.handleCurl(urlArgs), nil
	}

	if strings.TrimSpace(inputStr) == "deepseek status" {
		status, err := i.deepseekClient.CheckTokenStatus()
		var result strings.Builder
		if err != nil {
			return nil, fmt.Errorf("ошибка проверки статуса DeepSeek: %v", err)
		}

		result.WriteString(fmt.Sprintf("Токен действителен: %v\n", status))
		return result.String(), nil
	}

	if i.isFreeFormInput(inputStr) {
		return i.handleFreeFormInput(inputStr), nil
	}

	i.history.AddCommand(inputStr)

	// Обработка присваивания переменных
	if assignmentMatch, varName, expression := i.parseAssignment(inputStr); assignmentMatch {
		result, err := i.evaluateExpression(expression)
		if err != nil {
			return nil, err
		}
		i.variables.SetVariable(varName, result)
		i.saveState()
		return fmt.Sprintf("%s = %v", varName, result), nil
	}

	// Обработка математических выражений
	return i.evaluateExpression(inputStr)
}

func (i *Interpreter) ClearHistory() int {
	count := i.history.GetHistoryCount()
	i.history.ClearHistory()
	i.saveState()
	return count
}

// parseAssignment - парсинг присваивания переменных
func (i *Interpreter) parseAssignment(inputStr string) (bool, string, string) {
	re := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(.+)$`)
	matches := re.FindStringSubmatch(inputStr)
	if len(matches) == 3 {
		return true, matches[1], matches[2]
	}
	return false, "", ""
}

// GetHistoryCommands — получить только команды в виде массива строк
func (i *Interpreter) GetHistoryCommands(limit int) []string {
	history := i.history.GetDetailedHistory(limit)
	commands := make([]string, len(history))
	for idx, entry := range history {
		commands[idx] = entry.Command
	}
	return commands
}

// / isFreeFormInput - определяет, является ли ввод свободной формой

func (i *Interpreter) isFreeFormInput(inputStr string) bool {
	// Убираем лишние пробелы
	trimmed := strings.TrimSpace(inputStr)

	// Исключаем чистые математические выражения
	mathPattern := regexp.MustCompile(`^[\d\s+\-*/().^%]+$`)
	cleanInput := strings.ReplaceAll(trimmed, " ", "")
	if mathPattern.MatchString(cleanInput) {
		return false
	}

	// Исключаем присваивания переменных
	if strings.Contains(trimmed, "=") {
		assignmentPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`)
		if assignmentPattern.MatchString(trimmed) {
			return false
		}
	}

	// Исключаем команды истории
	if trimmed == "history" || trimmed == "history clear" || strings.HasPrefix(trimmed, "history search") {
		return false
	}

	// Исключаем простой вывод переменных (только имя переменной)
	varPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if varPattern.MatchString(trimmed) {
		return false
	}

	// Исключаем математические выражения с переменными
	// Например: x + 10, x-5, y * 2 и т.д.
	exprWithVarsPattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*|\d+)(\s*[\+\-\*/\^%]\s*([a-zA-Z_][a-zA-Z0-9_]*|\d+))*$`)
	cleanExpr := strings.ReplaceAll(trimmed, " ", "")

	return !exprWithVarsPattern.MatchString(cleanExpr)
}

func (i *Interpreter) handleFreeFormInput(inputStr string) string {

	// Классификация запроса
	classification := i.deepseekClient.ClassifyRequest(inputStr)
	fmt.Printf("Классификация: type=%s, url=%s, file_path=%s\n",
		classification.Type, classification.URL, classification.FilePath)

	switch classification.Type {
	case "browser":
		if classification.URL != "" {
			success := i.appLauncher.OpenBrowser(classification.URL)
			if success {
				summary, err := i.deepseekClient.GetContentSummary(inputStr,
					fmt.Sprintf("Открыт сайт %s", classification.URL))
				if err != nil {
					return fmt.Sprintf("Открыт браузер: %s", classification.URL)
				}
				return fmt.Sprintf("Открыт браузер: %s\n\nСводка:\n%s", classification.URL, summary)
			}
			return fmt.Sprintf("Не удалось открыть браузер: %s", classification.URL)
		}

	case "media":
		if classification.FilePath != "" {
			success := i.appLauncher.OpenMedia(classification.FilePath)
			if success {
				return fmt.Sprintf("Открыт медиафайл: %s", classification.FilePath)
			}
			return fmt.Sprintf("Не удалось открыть файл: %s", classification.FilePath)
		}

	case "curl":
		if classification.URL != "" {
			result, err := i.curlClient.Execute(classification.URL, nil)
			if err != nil {
				return fmt.Sprintf("Ошибка curl запроса: %v", err)
			}

			summary, err := i.deepseekClient.GetContentSummary(inputStr, result)
			if err != nil {
				return fmt.Sprintf("Результат запроса:\n%s", result)
			}
			return fmt.Sprintf("Результат запроса:\n%s\n\nСводка:\n%s", result, summary)
		}
	}

	// Общий AI ответ для general типа или если классификация не сработала
	response, err := i.deepseekClient.GetAIResponse(inputStr)
	if err != nil {
		return fmt.Sprintf("Ошибка получения AI ответа: %v", err)
	}

	return response
}

// evaluateExpression - вычисление математического выражения
func (i *Interpreter) evaluateExpression(expression string) (interface{}, error) {
	// Замена переменных на их значения
	exprWithVars, err := i.substituteVariables(expression)
	if err != nil {
		return nil, fmt.Errorf("ошибка подстановки переменных: %v", err)
	}

	result, err := i.evaluator.Evaluate(exprWithVars)
	if err != nil {
		return nil, fmt.Errorf("ошибка вычисления: %v", err)
	}

	return result, nil
}

// substituteVariables - подстановка значений переменных в выражение
func (i *Interpreter) substituteVariables(expression string) (string, error) {
	// Паттерн для поиска переменных: {var} или plain var
	pattern := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}|([a-zA-Z_][a-zA-Z0-9_]*)`)

	result := pattern.ReplaceAllStringFunc(expression, func(match string) string {
		// Извлекаем имя переменной
		var varName string
		if strings.HasPrefix(match, "{") && strings.HasSuffix(match, "}") {
			varName = match[1 : len(match)-1]
		} else {
			varName = match
		}

		// Проверяем, является ли это частью большего слова
		start := strings.Index(expression, match)
		if start > 0 {
			prevChar := expression[start-1]
			if isAlphaNumeric(prevChar) {
				return match // Возвращаем как есть, если это часть слова
			}
		}
		if start+len(match) < len(expression) {
			nextChar := expression[start+len(match)]
			if isAlphaNumeric(nextChar) {
				return match // Возвращаем как есть, если это часть слова
			}
		}

		// Получаем значение переменной
		value := i.variables.GetVariable(varName)
		if value == nil {
			return match // Оставляем как есть, если переменная не найдена
		}

		// Конвертируем значение в строку
		switch v := value.(type) {
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			return strconv.Itoa(v)
		case string:
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	})

	return result, nil
}

func (i *Interpreter) GetVariables() map[string]interface{} {
	return i.variables.GetVariables()
}

// isAlphaNumeric - проверка, является ли символ буквенно-цифровым
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// Дополнение к Interpreter для обработки curl команд
func (i *Interpreter) handleCurl(urlArgs string) string {
	if strings.TrimSpace(urlArgs) == "" {
		return "Использование: curl <url> [-H заголовок] [-X метод] [-d данные]"
	}

	// Парсинг аргументов curl
	parts := strings.Fields(urlArgs)
	if len(parts) == 0 {
		return "Ошибка: не указан URL"
	}

	url := parts[0]
	method := "GET"
	headers := make(map[string]string)
	var body string

	// Обработка дополнительных аргументов
	for j := 1; j < len(parts); j++ {
		switch parts[j] {
		case "-H":
			if j+1 < len(parts) {
				header := parts[j+1]
				if idx := strings.Index(header, ":"); idx != -1 {
					key := strings.TrimSpace(header[:idx])
					value := strings.TrimSpace(header[idx+1:])
					headers[key] = value
				}
				j++
			}
		case "-X":
			if j+1 < len(parts) {
				method = strings.ToUpper(parts[j+1])
				j++
			}
		case "-d":
			if j+1 < len(parts) {
				body = parts[j+1]
				j++
			}
		case "--help", "-h":
			return `Доступные опции curl:
  -H заголовок   Установить заголовок (например: "Content-Type: application/json")
  -X метод       Указать HTTP метод (GET, POST, PUT, DELETE)
  -d данные      Тело запроса для POST/PUT
  --help         Показать эту справку`
		}
	}

	var result string
	var err error

	if method == "GET" && body == "" {
		result, err = i.curlClient.Execute(url, headers)
	} else {
		result, err = i.curlClient.ExecuteWithMethod(method, url, headers, body)
	}

	if err != nil {
		return fmt.Sprintf("Ошибка curl: %v", err)
	}

	// Ограничиваем вывод для очень длинных ответов
	if len(result) > 100 {
		result = result[:100] + "\n\n... (вывод ограничен 1000 символов)"
	}

	return result
}
