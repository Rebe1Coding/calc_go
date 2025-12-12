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
	"strings"
)

// ============================================================================
// КОНСТАНТЫ
// ============================================================================

const (
	MaxOutputLength     = 1000
	MaxSummaryLength    = 500
	MaxHistoryEntries   = 1000
	MinUsernameLength   = 3
	RecentCommandsCount = 10
)

// ============================================================================
// ТИПЫ И ИНТЕРФЕЙСЫ
// ============================================================================

// Classification представляет результат классификации запроса
type Classification struct {
	Type     string
	URL      string
	FilePath string
}

type Interpreter struct {
	evaluator      *evaluator.Evaluator
	variables      *variables.VariableStore
	persistence    *persistence.PersistenceManager
	history        *history.HistoryManager
	curlClient     *curl.CurlClient
	deepseekClient *agent.DeepSeekClient
	appLauncher    *applauncher.AppLauncher
}

// ============================================================================
// ИНИЦИАЛИЗАЦИЯ
// ============================================================================

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

func (i *Interpreter) loadState() {
	data := i.persistence.LoadData()
	if data == nil {
		return
	}

	if len(data.History) > 0 {
		i.history.SetHistory(data.History)
	}

	if len(data.Variables) > 0 {
		i.variables.SetVariables(data.Variables)
	}

	i.displayRecentHistory()
}

func (i *Interpreter) displayRecentHistory() {
	recentHistory := i.history.GetDetailedHistory(RecentCommandsCount)
	if len(recentHistory) == 0 {
		return
	}

	fmt.Printf("Последние %d команд из истории:\n", RecentCommandsCount)
	fmt.Println(strings.Repeat("-", 50))
	for _, entry := range recentHistory {
		fmt.Printf("%3d. [%s] %s\n", entry.ID, entry.Time, entry.Command)
	}
	fmt.Println(strings.Repeat("-", 50))
}

func (i *Interpreter) saveState() {
	data := &persistence.CalculatorData{
		History:   i.history.GetHistory(MaxHistoryEntries),
		Variables: i.variables.GetVariables(),
	}
	i.persistence.SaveData(data)
}

// ============================================================================
// ОСНОВНОЕ ВЫПОЛНЕНИЕ
// ============================================================================

func (i *Interpreter) Execute(inputStr string) (interface{}, error) {

	// Обработка curl команд
	if strings.HasPrefix(strings.TrimSpace(inputStr), "curl ") {
		urlArgs := strings.TrimSpace(inputStr[5:])
		return i.handleCurl(urlArgs), nil
	}

	// Обработка свободной формы (AI)
	if i.isFreeFormInput(inputStr) {
		return i.handleFreeFormInput(inputStr), nil
	}

	// Добавление в историю для математических выражений
	i.history.AddCommand(inputStr)

	// Обработка присваивания переменных
	if match, varName, expression := i.parseAssignment(inputStr); match {
		return i.handleAssignment(varName, expression)
	}

	// Обработка математических выражений
	return i.evaluateExpression(inputStr)
}

// ============================================================================
// ОБРАБОТКА ТИПОВ КОМАНД
// ============================================================================

func (i *Interpreter) handleAssignment(varName, expression string) (interface{}, error) {
	result, err := i.evaluateExpression(expression)
	if err != nil {
		return nil, err
	}
	i.variables.SetVariable(varName, result)
	i.saveState()
	return fmt.Sprintf("%s = %v", varName, result), nil
}

func (i *Interpreter) handleFreeFormInput(inputStr string) string {
	classification := i.classifyAndParseRequest(inputStr)

	switch classification.Type {
	case "browser":
		return i.handleBrowser(classification, inputStr)
	case "media":
		return i.handleMedia(classification)
	case "curl":
		return i.handleCurlRequest(classification, inputStr)
	default:
		return i.getAIResponse(inputStr)
	}
}

func (i *Interpreter) classifyAndParseRequest(inputStr string) Classification {
	rawClassification := i.deepseekClient.ClassifyRequest(inputStr)

	fmt.Printf("Классификация: type=%s, url=%s, file_path=%s\n",
		rawClassification.Type, rawClassification.URL, rawClassification.FilePath)

	// Безопасное преобразование в структуру Classification
	return Classification{
		Type:     getStringField(rawClassification, "Type"),
		URL:      getStringField(rawClassification, "URL"),
		FilePath: getStringField(rawClassification, "FilePath"),
	}
}

// ============================================================================
// ОБРАБОТЧИКИ КЛАССИФИКАЦИЙ
// ============================================================================

func (i *Interpreter) handleBrowser(classification Classification, inputStr string) string {
	if classification.URL == "" {
		return "❌ Не удалось определить URL для браузера"
	}

	if !i.appLauncher.OpenBrowser(classification.URL) {
		return fmt.Sprintf("❌ Не удалось открыть браузер: %s", classification.URL)
	}

	summary := i.getContentSummary(inputStr, fmt.Sprintf("Открыт сайт %s", classification.URL))
	if summary != "" {
		return fmt.Sprintf("✅ Открыт браузер: %s\n\nСводка:\n%s", classification.URL, summary)
	}

	return fmt.Sprintf("✅ Открыт браузер: %s", classification.URL)
}

func (i *Interpreter) handleMedia(classification Classification) string {
	if classification.FilePath == "" {
		return "❌ Не удалось определить путь к медиафайлу"
	}

	if !i.appLauncher.OpenMedia(classification.FilePath) {
		return fmt.Sprintf("❌ Не удалось открыть файл: %s", classification.FilePath)
	}

	return fmt.Sprintf("✅ Открыт медиафайл: %s", classification.FilePath)
}

func (i *Interpreter) handleCurlRequest(classification Classification, inputStr string) string {
	if classification.URL == "" {
		return "❌ Не удалось определить URL для curl запроса"
	}

	result, err := i.curlClient.Execute(classification.URL, nil)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка curl запроса: %v", err)
	}

	result = limitOutput(result, MaxSummaryLength)
	summary := i.getContentSummary(inputStr, result)

	if summary != "" {
		return fmt.Sprintf("✅ Результат запроса:\n%s\n\nСводка:\n%s", result, summary)
	}

	return fmt.Sprintf("✅ Результат запроса:\n%s", result)
}

// ============================================================================
// ПАРСИНГ КОМАНД
// ============================================================================

func (i *Interpreter) parseCallCommand(inputStr string) (bool, string, string) {
	trimmed := strings.ToLower(strings.TrimSpace(inputStr))

	patterns := []struct {
		pattern  string
		callType string
	}{
		{`^позвонить\s+([a-zA-Z0-9_]+)(?:\s+(video|audio))?$`, "video"},
		{`^call\s+([a-zA-Z0-9_]+)(?:\s+(video|audio))?$`, "video"},
		{`^видеозвонок\s+([a-zA-Z0-9_]+)$`, "video"},
		{`^аудиозвонок\s+([a-zA-Z0-9_]+)$`, "audio"},
		{`^video\s+call\s+([a-zA-Z0-9_]+)$`, "video"},
		{`^audio\s+call\s+([a-zA-Z0-9_]+)$`, "audio"},
	}

	for _, p := range patterns {
		re := regexp.MustCompile(p.pattern)
		matches := re.FindStringSubmatch(trimmed)

		if len(matches) >= 2 {
			target := matches[1]
			callType := p.callType

			// Переопределение типа звонка, если указан явно
			if len(matches) == 3 && matches[2] != "" {
				callType = matches[2]
			}

			return true, target, callType
		}
	}

	return false, "", ""
}

func (i *Interpreter) parseLoginCommand(inputStr string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(inputStr))
	loginPattern := regexp.MustCompile(`^(login|войти)\s+([a-zA-Z0-9_]+)$`)
	matches := loginPattern.FindStringSubmatch(trimmed)

	if len(matches) != 3 {
		return "", fmt.Errorf("неверный формат команды. Используйте: login <username> или войти <username>")
	}

	return matches[2], nil
}

func (i *Interpreter) parseAssignment(inputStr string) (bool, string, string) {
	re := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(.+)$`)
	matches := re.FindStringSubmatch(inputStr)

	if len(matches) == 3 {
		return true, matches[1], matches[2]
	}

	return false, "", ""
}

func (i *Interpreter) isLoginCommand(inputStr string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(inputStr))
	loginPattern := regexp.MustCompile(`^(login|войти)\s+`)
	return loginPattern.MatchString(trimmed)
}

// ============================================================================
// ОПРЕДЕЛЕНИЕ ТИПА ВВОДА
// ============================================================================

func (i *Interpreter) isFreeFormInput(inputStr string) bool {
	trimmed := strings.TrimSpace(inputStr)

	// Проверки на специальные команды
	if i.isSpecialCommand(trimmed) {
		return false
	}

	// Проверка на математическое выражение
	if i.isMathExpression(trimmed) {
		return false
	}

	// Проверка на присваивание переменной
	if i.isVariableAssignment(trimmed) {
		return false
	}

	// Проверка на переменную или выражение с переменными
	if i.isVariableOrExpression(trimmed) {
		return false
	}

	return true
}

func (i *Interpreter) isSpecialCommand(trimmed string) bool {
	// Проверка на команды звонков
	if match, _, _ := i.parseCallCommand(trimmed); match {
		return true
	}

	// Проверка на команду логина
	if i.isLoginCommand(trimmed) {
		return true
	}

	// Проверка на команды истории
	if trimmed == "history" || trimmed == "history clear" || strings.HasPrefix(trimmed, "history search") {
		return true
	}

	return false
}

func (i *Interpreter) isMathExpression(trimmed string) bool {
	mathPattern := regexp.MustCompile(`^[\d\s+\-*/().^%]+$`)
	cleanInput := strings.ReplaceAll(trimmed, " ", "")
	return mathPattern.MatchString(cleanInput)
}

func (i *Interpreter) isVariableAssignment(trimmed string) bool {
	if !strings.Contains(trimmed, "=") {
		return false
	}

	assignmentPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`)
	return assignmentPattern.MatchString(trimmed)
}

func (i *Interpreter) isVariableOrExpression(trimmed string) bool {
	// Простая переменная
	varPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if varPattern.MatchString(trimmed) {
		return true
	}

	// Выражение с переменными
	exprWithVarsPattern := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*|\d+)(\s*[\+\-\*/\^%]\s*([a-zA-Z_][a-zA-Z0-9_]*|\d+))*$`)
	cleanExpr := strings.ReplaceAll(trimmed, " ", "")
	return exprWithVarsPattern.MatchString(cleanExpr)
}

// ============================================================================
// ОБРАБОТКА CURL
// ============================================================================

func (i *Interpreter) handleCurl(urlArgs string) string {
	if strings.TrimSpace(urlArgs) == "" {
		return "Использование: curl <url> [-H заголовок] [-X метод] [-d данные]"
	}

	url, method, headers, body, help := i.parseCurlArgs(urlArgs)

	if help {
		return i.getCurlHelp()
	}

	if url == "" {
		return "❌ Ошибка: не указан URL"
	}

	result, err := i.executeCurlRequest(method, url, headers, body)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка curl: %v", err)
	}

	return limitOutput(result, MaxOutputLength)
}

func (i *Interpreter) parseCurlArgs(urlArgs string) (url, method string, headers map[string]string, body string, help bool) {
	parts := strings.Fields(urlArgs)
	if len(parts) == 0 {
		return
	}

	url = parts[0]
	method = "GET"
	headers = make(map[string]string)

	for j := 1; j < len(parts); j++ {
		switch parts[j] {
		case "-H":
			if j+1 < len(parts) {
				if key, value, ok := parseHeader(parts[j+1]); ok {
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
			help = true
			return
		}
	}

	return
}

func (i *Interpreter) executeCurlRequest(method, url string, headers map[string]string, body string) (string, error) {
	if method == "GET" && body == "" {
		return i.curlClient.Execute(url, headers)
	}
	return i.curlClient.ExecuteWithMethod(method, url, headers, body)
}

func (i *Interpreter) getCurlHelp() string {
	return `Доступные опции curl:
  -H заголовок   Установить заголовок (например: "Content-Type: application/json")
  -X метод       Указать HTTP метод (GET, POST, PUT, DELETE)
  -d данные      Тело запроса для POST/PUT
  --help         Показать эту справку`
}

// ============================================================================
// ВЫЧИСЛЕНИЯ И ПЕРЕМЕННЫЕ
// ============================================================================

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

func (i *Interpreter) substituteVariables(expression string) (string, error) {
	pattern := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}|([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := pattern.FindAllStringSubmatchIndex(expression, -1)

	if len(matches) == 0 {
		return expression, nil
	}

	var result strings.Builder
	lastIndex := 0

	for _, m := range matches {
		start, end := m[0], m[1]

		// Добавляем текст до совпадения
		if lastIndex < start {
			result.WriteString(expression[lastIndex:start])
		}

		varName := extractVariableName(expression, m)

		// Проверка, является ли совпадение частью слова
		if isPartOfWord(expression, start, end) || varName == "" {
			result.WriteString(expression[start:end])
			lastIndex = end
			continue
		}

		// Подстановка значения переменной
		value := i.variables.GetVariable(varName)
		if value == nil {
			result.WriteString(expression[start:end])
		} else {
			result.WriteString(formatVariableValue(value))
		}

		lastIndex = end
	}

	if lastIndex < len(expression) {
		result.WriteString(expression[lastIndex:])
	}

	return result.String(), nil
}

// ============================================================================
// ПУБЛИЧНЫЕ МЕТОДЫ
// ============================================================================

func (i *Interpreter) ClearHistory() int {
	count := i.history.GetHistoryCount()
	i.history.ClearHistory()
	i.saveState()
	return count
}

func (i *Interpreter) GetHistoryCommands(limit int) []string {
	history := i.history.GetDetailedHistory(limit)
	commands := make([]string, len(history))
	for idx, entry := range history {
		commands[idx] = entry.Command
	}
	return commands
}

func (i *Interpreter) GetVariables() map[string]interface{} {
	return i.variables.GetVariables()
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================================

func getStringField(data interface{}, fieldName string) string {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[fieldName]; exists {
			if strVal, ok := val.(string); ok {
				return strVal
			}
		}
	}

	// Попытка использовать рефлексию для структур
	// (если ClassifyRequest возвращает структуру)
	// Здесь можно добавить reflection при необходимости

	return ""
}

func (i *Interpreter) getAIResponse(inputStr string) string {
	response, err := i.deepseekClient.GetAIResponse(inputStr)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка получения AI ответа: %v", err)
	}
	return response
}

func (i *Interpreter) getContentSummary(inputStr, result string) string {
	summary, err := i.deepseekClient.GetContentSummary(inputStr, result)
	if err != nil {
		return ""
	}
	return summary
}

func limitOutput(output string, maxLen int) string {
	if len(output) > maxLen {
		return output[:maxLen] + "\n... (вывод ограничен)"
	}
	return output
}

func parseHeader(header string) (key, value string, ok bool) {
	idx := strings.Index(header, ":")
	if idx == -1 {
		return "", "", false
	}

	key = strings.TrimSpace(header[:idx])
	value = strings.TrimSpace(header[idx+1:])
	return key, value, true
}

func extractVariableName(expression string, matchIndices []int) string {
	// matchIndices: [start, end, group1_start, group1_end, group2_start, group2_end]
	if len(matchIndices) >= 4 && matchIndices[2] != -1 {
		// Совпадение с фигурными скобками {var}
		return expression[matchIndices[2]:matchIndices[3]]
	}
	if len(matchIndices) >= 6 && matchIndices[4] != -1 {
		// Совпадение без скобок var
		return expression[matchIndices[4]:matchIndices[5]]
	}
	return ""
}

func isPartOfWord(expression string, start, end int) bool {
	if start > 0 && isAlphaNumeric(expression[start-1]) {
		return true
	}
	if end < len(expression) && isAlphaNumeric(expression[end]) {
		return true
	}
	return false
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func formatVariableValue(value interface{}) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%v", v)
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
