package ui

import (
	"app/core/interpreter"
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConsoleInterface представляет консольный интерфейс
type ConsoleInterface struct {
	interpreter  *interpreter.Interpreter
	scanner      *bufio.Scanner
	history      []string
	historyIndex int
}

// NewConsoleInterface создает новый консольный интерфейс
func NewConsoleInterface(i *interpreter.Interpreter) *ConsoleInterface {
	return &ConsoleInterface{
		interpreter: interpreter.NewInterpreter(),
		scanner:     bufio.NewScanner(os.Stdin),
	}
}

// Run запускает главный цикл интерфейса
func (c *ConsoleInterface) Run() error {
	c.showWelcome()
	c.loadHistory()

	// Основной цикл
	for {
		fmt.Print("calc> ")

		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		// Проверяем команды выхода
		if input == "/quit" || input == "/exit" {
			break
		}

		// Обрабатываем команду
		c.processCommand(input)
	}

	// Сохраняем состояние при выходе
	// c.interpreter.saveState()

	fmt.Println("👋 До свидания!")
	return nil
}

// showWelcome показывает приветственное сообщение
func (c *ConsoleInterface) showWelcome() {
	fmt.Println("🧮 ═══════════════════════════════════════════")
	fmt.Println("   Калькулятор-интерпретатор с curl поддержкой")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()
	fmt.Println("📚 Команды:")
	fmt.Println("  • Арифметика: 2 + 3 * 4, (x + y) / 2")
	fmt.Println("  • Переменные: x = 10, name = \"hello\"")
	fmt.Println("  • HTTP запросы: curl https://api.github.com")
	fmt.Println("  • С заголовками: curl https://api.com -H \"Auth: token\"")
	fmt.Println("  • Просмотр: /vars, /history, /show <переменная>")
	fmt.Println("  • Удаление: /del <переменная>, /clear-vars, /clear-history")
	fmt.Println("  • Справка: /help")
	fmt.Println("  • Выход: /quit или Ctrl+C")
	fmt.Println()
}

// loadHistory загружает и показывает последние команды
func (c *ConsoleInterface) loadHistory() {
	history := c.interpreter.ShowHistory()
	if len(history) > 0 {
		fmt.Println("📜 Последние команды:")
		for i, cmd := range history {
			if i >= 5 { // показываем только последние 5
				break
			}
			fmt.Printf("   %d. %s\n", i+1, cmd)
		}
		fmt.Println()
	}
}

// processCommand обрабатывает введенную команду
func (c *ConsoleInterface) processCommand(input string) {
	// Добавляем в локальную историю для навигации
	c.history = append(c.history, input)
	if len(c.history) > 50 {
		c.history = c.history[len(c.history)-50:]
	}
	c.historyIndex = -1

	// Специальные команды
	switch {
	case input == "/vars":
		c.showVariables()
		return
	case input == "/history":
		c.showHistory()
		return
	case input == "/help":
		c.showHelp()
		return
	case input == "/clear":
		c.clearScreen()
		return
	case input == "/clear-history" || input == "/clearhist":
		c.interpreter.ClearHistory()
		fmt.Println("🗑️ История команд очищена")
		return

	}

	// Обрабатываем обычную команду
	result, err := c.interpreter.Execute(input)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	// Выводим результат
	switch v := result.(type) {
	case float64:
		fmt.Printf("📊 = %.2f\n", v)
	case string:
		c.displayString(v)
	default:
		fmt.Printf("✅ %v\n", result)
	}
}

// displayString отображает строковый результат с возможностью усечения
func (c *ConsoleInterface) displayString(text string) {
	lines := strings.Split(text, "\n")
	if len(lines) > 5 || len(text) > 300 {
		// Показываем превью
		preview := text
		if len(text) > 300 {
			preview = text[:300] + "..."
		}
		if len(lines) > 5 {
			preview = strings.Join(lines[:5], "\n") + "\n... (еще строк: " + fmt.Sprintf("%d", len(lines)-5) + ")"
		}

		fmt.Printf("📄 %s\n", preview)
		fmt.Println("💡 Сохраните в переменную и используйте /show <переменная> для полного просмотра")
	} else {
		fmt.Printf("📄 %s\n", text)
	}
}

// showVariables показывает все переменные
func (c *ConsoleInterface) showVariables() {
	vars := c.interpreter.GetVariables()
	if len(vars) == 0 {
		fmt.Println("ℹ️ Переменные не определены")
		return
	}

	fmt.Printf("📊 Переменные: %v\n", vars)

}

// showHistory показывает историю команд
func (c *ConsoleInterface) showHistory() {
	history := c.interpreter.ShowHistory()
	if len(history) == 0 {
		fmt.Println("ℹ️ История команд пуста")
		return
	}

	fmt.Println("📜 История команд:")
	for i, cmd := range history {
		fmt.Printf("  %d. %s\n", i+1, cmd)
	}
	fmt.Println("\n💡 Команды: /clear-history - очистить историю")
}

// clearScreen очищает экран (эмуляция)
func (c *ConsoleInterface) clearScreen() {
	// Печатаем много пустых строк для "очистки"
	for i := 0; i < 50; i++ {
		fmt.Println()
	}
	fmt.Println("🧮 Экран очищен (переменные и история сохранены)")
}

// showHelp показывает подробную справку
func (c *ConsoleInterface) showHelp() {
	fmt.Println("📚 ═══════════════ СПРАВКА ═══════════════")
	fmt.Println()
	fmt.Println("🔢 АРИФМЕТИЧЕСКИЕ ОПЕРАЦИИ:")
	fmt.Println("  2 + 3 * 4           - базовые вычисления")
	fmt.Println("  (10 + 5) / 3        - скобки поддерживаются")
	fmt.Println("  x = 10              - создание числовой переменной")
	fmt.Println("  result = x * 2 + 5  - использование переменных")
	fmt.Println()
	fmt.Println("📝 СТРОКОВЫЕ ПЕРЕМЕННЫЕ:")
	fmt.Println("  name = \"John Doe\"    - строка в кавычках")
	fmt.Println("  formula = 'x + y'   - одинарные кавычки")
	fmt.Println()
	fmt.Println("🌐 HTTP ЗАПРОСЫ (CURL):")
	fmt.Println("  curl https://api.github.com")
	fmt.Println("  curl https://httpbin.org/json")
	fmt.Println("  curl https://api.com -H \"Authorization: Bearer token\"")
	fmt.Println("  curl https://api.com -H \"Content-Type: application/json\"")
	fmt.Println("  data = curl https://api.example.com")
	fmt.Println()
	fmt.Println("👀 ПРОСМОТР ДАННЫХ:")
	fmt.Println("  /vars               - показать все переменные")
	fmt.Println("  /history            - показать историю команд")
	fmt.Println("  /show <переменная>  - показать полный текст переменной")
	fmt.Println()
	fmt.Println("🗑️ УПРАВЛЕНИЕ ДАННЫМИ:")
	fmt.Println("  /del <переменная>   - удалить переменную")
	fmt.Println("  /delete <переменная> - удалить переменную (альтернатива)")
	fmt.Println("  /clear-vars         - удалить все переменные")
	fmt.Println("  /clear-history      - очистить историю команд")
	fmt.Println("  /clear              - очистить экран")
	fmt.Println()
	fmt.Println("🔧 СИСТЕМНЫЕ КОМАНДЫ:")
	fmt.Println("  /help               - показать эту справку")
	fmt.Println("  /quit или /exit     - выход из программы")
	fmt.Println()
	fmt.Println("💡 ПРИМЕРЫ ИСПОЛЬЗОВАНИЯ:")
	fmt.Println("  calc> x = 10")
	fmt.Println("  calc> y = x * 2 + 5")
	fmt.Println("  calc> weather = curl https://wttr.in/Moscow?format=3")
	fmt.Println("  calc> /show weather")
	fmt.Println("  calc> /vars")
	fmt.Println("  calc> /del x")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
}
