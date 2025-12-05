package interpreter

import (
	agent "app/core/ai"
	"app/core/applauncher"
	"app/core/curl"
	"app/core/evaluator"
	"app/core/history"
	"app/core/persistence"
	"app/core/variables"
	"app/core/webrtc"
	"fmt"
	"regexp"
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
	webrtcServer   WebRTCServer // Добавляем интерфейс WebRTC сервера
	currentUser    string       // Текущий пользователь для WebRTC
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

func (i *Interpreter) Execute(inputStr string) (interface{}, error) {
	// Обработка команды звонка
	if match, target, callType := i.parseCallCommand(inputStr); match {
		return i.handleCall(target, callType)
	}

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
	trimmed := strings.TrimSpace(inputStr)

	// Исключаем команды звонков
	if match, _, _ := i.parseCallCommand(inputStr); match {
		return false
	}

	// Исключаем команду входа
	loginPattern := regexp.MustCompile(`^(login|войти)\s+`)
	if loginPattern.MatchString(strings.ToLower(trimmed)) {
		return false
	}

	// Убираем лишние пробелы
	mathPattern := regexp.MustCompile(`^[\d\s+\-*/().^%]+$`)
	cleanInput := strings.ReplaceAll(trimmed, " ", "")
	if mathPattern.MatchString(cleanInput) {
		return false
	}

	// Остальные проверки как раньше...
	if strings.Contains(trimmed, "=") {
		assignmentPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`)
		if assignmentPattern.MatchString(trimmed) {
			return false
		}
	}

	if trimmed == "history" || trimmed == "history clear" || strings.HasPrefix(trimmed, "history search") {
		return false
	}

	varPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if varPattern.MatchString(trimmed) {
		return false
	}

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
		return i.handleBrowserClassification(classification, inputStr)

	case "media":
		return i.handleMediaClassification(classification)

	case "curl":
		return i.handleCurlClassification(classification, inputStr)

	case "call":
		return i.handleCallClassification(inputStr)

	case "login":
		return i.handleLoginClassification(inputStr)

	default:
		// Общий AI ответ для general типа
		response, err := i.deepseekClient.GetAIResponse(inputStr)
		if err != nil {
			return fmt.Sprintf("Ошибка получения AI ответа: %v", err)
		}
		return response
	}
}

// ============================================================================
// ОБРАБОТКА КЛАССИФИКАЦИЙ
// ============================================================================

func (i *Interpreter) handleBrowserClassification(classification interface{}, inputStr string) string {
	// Приведение типа (в зависимости от структуры Classification)
	type ClassificationType interface {
		GetURL() string
		GetType() string
	}

	// Попытка получить URL из классификации
	// Предположим, что classification имеет поле URL
	classif := classification.(map[string]interface{})
	url, ok := classif["URL"].(string)
	if !ok || url == "" {
		return "Ошибка: не удалось определить URL для браузера"
	}

	success := i.appLauncher.OpenBrowser(url)
	if !success {
		return fmt.Sprintf("Не удалось открыть браузер: %s", url)
	}

	// Получаем сводку содержимого
	summary, err := i.deepseekClient.GetContentSummary(inputStr,
		fmt.Sprintf("Открыт сайт %s", url))
	if err != nil {
		return fmt.Sprintf("✅ Открыт браузер: %s", url)
	}

	return fmt.Sprintf("✅ Открыт браузер: %s\n\nСводка:\n%s", url, summary)
}

func (i *Interpreter) handleMediaClassification(classification interface{}) string {
	classif := classification.(map[string]interface{})
	filePath, ok := classif["FilePath"].(string)
	if !ok || filePath == "" {
		return "Ошибка: не удалось определить путь к медиафайлу"
	}

	success := i.appLauncher.OpenMedia(filePath)
	if !success {
		return fmt.Sprintf("❌ Не удалось открыть файл: %s", filePath)
	}

	return fmt.Sprintf("✅ Открыт медиафайл: %s", filePath)
}

func (i *Interpreter) handleCurlClassification(classification interface{}, inputStr string) string {
	classif := classification.(map[string]interface{})
	url, ok := classif["URL"].(string)
	if !ok || url == "" {
		return "Ошибка: не удалось определить URL для curl запроса"
	}

	result, err := i.curlClient.Execute(url, nil)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка curl запроса: %v", err)
	}

	// Ограничиваем размер результата
	const maxLen = 500
	if len(result) > maxLen {
		result = result[:maxLen] + "\n... (вывод ограничен)"
	}

	// Получаем сводку
	summary, err := i.deepseekClient.GetContentSummary(inputStr, result)
	if err != nil {
		return fmt.Sprintf("✅ Результат запроса:\n%s", result)
	}

	return fmt.Sprintf("✅ Результат запроса:\n%s\n\nСводка:\n%s", result, summary)
}

func (i *Interpreter) handleCallClassification(inputStr string) string {
	// Парсим команду звонка из исходного текста
	match, target, callType := i.parseCallCommand(inputStr)
	if !match {
		return "❌ Не удалось распознать команду звонка. Используйте: call <username> [video|audio]"
	}

	// Вызываем реальный обработчик звонка
	result, err := i.handleCall(target, callType)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка при инициации звонка: %v", err)
	}

	return result
}

func (i *Interpreter) handleLoginClassification(inputStr string) string {
	// Парсим команду логина
	result, err := i.HandleWebRTCLogin(inputStr)
	if err != nil {
		return fmt.Sprintf("❌ %v", err)
	}

	return result
}

// ============================================================================
// УЛУЧШЕННАЯ ОБРАБОТКА ЗВОНКОВ В handleCall
// ============================================================================

func (i *Interpreter) handleCall(target, callType string) (string, error) {
	// Валидация
	if i.webrtcServer == nil {
		return "", fmt.Errorf("WebRTC сервер не инициализирован")
	}

	if i.currentUser == "" {
		return "", fmt.Errorf("не установлен текущий пользователь. Используйте команду: login <username>")
	}

	if target == i.currentUser {
		return "", fmt.Errorf("невозможно позвонить самому себе")
	}

	if target == "" {
		return "", fmt.Errorf("не указан адресат звонка")
	}

	// Инициируем звонок
	session, err := i.webrtcServer.InitiateCall(i.currentUser, target, callType)
	if err != nil {
		return "", fmt.Errorf("ошибка при инициации звонка: %v", err)
	}

	// Логируем в историю
	i.history.AddCommand(fmt.Sprintf("позвонить %s %s", target, callType))
	i.saveState()

	callTypeDisplay := strings.ToUpper(callType[:1]) + callType[1:]
	return fmt.Sprintf("✅ %s звонок пользователю '%s' инициирован.\n"+
		"Откройте WebRTC интерфейс по адресу: http://localhost:8000/webrtc/\n"+
		"Сессия: %v", callTypeDisplay, target, session), nil
}

func (i *Interpreter) HandleWebRTCLogin(inputStr string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(inputStr))

	// Паттерн "login alice" или "войти alice"
	loginPattern := regexp.MustCompile(`^(login|войти)\s+([a-zA-Z0-9_]+)$`)
	matches := loginPattern.FindStringSubmatch(trimmed)

	if len(matches) != 3 {
		return "", fmt.Errorf("неверный формат команды. Используйте: login <username> или войти <username>")
	}

	username := matches[2]

	// Валидация имени пользователя
	if len(username) < 3 {
		return "", fmt.Errorf("имя пользователя должно быть минимум 3 символа")
	}

	i.SetCurrentUser(username)
	i.history.AddCommand(fmt.Sprintf("login %s", username))
	i.saveState()

	return fmt.Sprintf("✅ Вы вошли как '%s'. Теперь вы можете совершать звонки.", username), nil
}

// ============================================================================
// ПАРСИНГ КОМАНД
// ============================================================================

// parseCallCommand - парсит команды звонков
func (i *Interpreter) parseCallCommand(inputStr string) (bool, string, string) {
	trimmed := strings.ToLower(strings.TrimSpace(inputStr))

	patterns := []struct {
		regex    *regexp.Regexp
		callType string
	}{
		{regexp.MustCompile(`^позвонить\s+([a-zA-Z0-9_]+)(?:\s+(video|audio))?$`), "video"},
		{regexp.MustCompile(`^call\s+([a-zA-Z0-9_]+)(?:\s+(video|audio))?$`), "video"},
		{regexp.MustCompile(`^видеозвонок\s+([a-zA-Z0-9_]+)$`), "video"},
		{regexp.MustCompile(`^аудиозвонок\s+([a-zA-Z0-9_]+)$`), "audio"},
		{regexp.MustCompile(`^video\s+call\s+([a-zA-Z0-9_]+)$`), "video"},
		{regexp.MustCompile(`^audio\s+call\s+([a-zA-Z0-9_]+)$`), "audio"},
	}

	for _, p := range patterns {
		matches := p.regex.FindStringSubmatch(trimmed)
		if len(matches) >= 2 {
			target := matches[1]
			callType := p.callType

			if len(matches) == 3 && matches[2] != "" {
				callType = matches[2]
			}

			return true, target, callType
		}
	}

	return false, "", ""
}

// ============================================================================
// ВЫЧИСЛЕНИЯ
// ============================================================================

// evaluateExpression - вычисляет математическое выражение с переменными
func (i *Interpreter) evaluateExpression(expression string) (interface{}, error) {
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

// substituteVariables - подставляет значения переменных в выражение
func (i *Interpreter) substituteVariables(expression string) (string, error) {
	pattern := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}|([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := pattern.FindAllStringSubmatchIndex(expression, -1)

	if len(matches) == 0 {
		return expression, nil
	}

	var result strings.Builder
	lastIndex := 0

	for _, m := range matches {
		start := m[0]
		end := m[1]

		if lastIndex < start {
			result.WriteString(expression[lastIndex:start])
		}

		varName := ""
		if len(m) >= 4 && m[2] != -1 {
			varName = expression[m[2]:m[3]]
		} else if len(m) >= 6 && m[4] != -1 {
			varName = expression[m[4]:m[5]]
		}

		// Проверка на часть слова
		partOfWord := false
		if start > 0 && isAlphaNumeric(expression[start-1]) {
			partOfWord = true
		}
		if end < len(expression) && isAlphaNumeric(expression[end]) {
			partOfWord = true
		}

		if partOfWord || varName == "" {
			result.WriteString(expression[start:end])
			lastIndex = end
			continue
		}

		value := i.variables.GetVariable(varName)
		if value == nil {
			result.WriteString(expression[start:end])
			lastIndex = end
			continue
		}

		// Конвертирование значения
		switch v := value.(type) {
		case float64:
			result.WriteString(fmt.Sprintf("%v", v))
		case int:
			result.WriteString(fmt.Sprintf("%d", v))
		case string:
			result.WriteString(v)
		default:
			result.WriteString(fmt.Sprintf("%v", v))
		}

		lastIndex = end
	}

	if lastIndex < len(expression) {
		result.WriteString(expression[lastIndex:])
	}

	return result.String(), nil
}

// ============================================================================
// ОБРАБОТКА CURL
// ============================================================================

// handleCurl - обрабатывает curl команды
func (i *Interpreter) handleCurl(urlArgs string) string {
	if strings.TrimSpace(urlArgs) == "" {
		return "Использование: curl <url> [-H заголовок] [-X метод] [-d данные]"
	}

	parts := strings.Fields(urlArgs)
	if len(parts) == 0 {
		return "Ошибка: не указан URL"
	}

	url := parts[0]
	method := "GET"
	headers := make(map[string]string)
	var body string

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
		return fmt.Sprintf("❌ Ошибка curl: %v", err)
	}

	const maxLen = 1000
	if len(result) > maxLen {
		result = result[:maxLen] + "\n\n... (вывод ограничен 1000 символов)"
	}

	return result
}

// ============================================================================
// УПРАВЛЕНИЕ WEBRTC
// ============================================================================

// SetWebRTCServer - устанавливает WebRTC сервер

type WebRTCServer interface {
	InitiateCall(caller, target, callType string) (*webrtc.Session, error)
}

func (i *Interpreter) SetWebRTCServer(server WebRTCServer) {
	i.webrtcServer = server
}

// SetCurrentUser - устанавливает текущего пользователя
func (i *Interpreter) SetCurrentUser(username string) {
	i.currentUser = username
}

// GetCurrentUser - получает текущего пользователя
func (i *Interpreter) GetCurrentUser() string {
	return i.currentUser
}

// ============================================================================
// УТИЛИТЫ
// ============================================================================

// isAlphaNumeric - проверяет является ли символ буквой, цифрой или подчеркиванием
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// GetVariables - получает все переменные
func (i *Interpreter) GetVariables() map[string]interface{} {
	return i.variables.GetVariables()
}
