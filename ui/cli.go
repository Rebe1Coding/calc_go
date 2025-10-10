package cli

import (
	"app/service/calculator"
	"bufio"
	"fmt"
	"os"
)

type CLI struct {
	calculator *calculator.Calculator
}

func NewCLI(calculator *calculator.Calculator) *CLI {
	return &CLI{
		calculator: calculator,
	}
}

func (c *CLI) Start() {
	fmt.Println("┌──────────────────────────────────────────────────┐")
	fmt.Println("│              Калькулятор CLI v2.0                │")
	fmt.Println("│        Введите 'help' для списка команд          │")
	fmt.Println("│           Введите 'exit' для выхода              │")
	fmt.Println("└──────────────────────────────────────────────────┘")
	fmt.Println("🚀 Калькулятор CLI запущен. Для выхода введите 'q'.")
	fmt.Println("📝 Доступные операции: +, -, *, /, присваивание переменных, curl")

	c.showHistory()
	c.showVariables()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "q" {
			break
		}

		if input == "history" {
			c.showHistory()
			continue
		}

		if input == "variables" {
			c.showVariables()
			continue
		}

		if input == "help" {
			c.showHelp()
			continue
		}

		result := c.calculator.Execute(input)
		if result != "" {
			fmt.Println(result)
		}
	}

	fmt.Println("Goodbye!")
}

func (c *CLI) showHistory() {
	commands := c.calculator.GetLastCommands()
	if len(commands) == 0 {
		fmt.Println("No command history.")
		return
	}

	fmt.Println("\nLast 10 commands:")
	for _, cmd := range commands {
		fmt.Printf("[%s] %s -> %s\n", cmd.Timestamp, cmd.Command, cmd.Result)
	}
	fmt.Println()
}

func (c *CLI) showVariables() {
	variables := c.calculator.GetVariables()
	if len(variables) == 0 {
		fmt.Println("No variables defined.")
		return
	}

	fmt.Println("\nVariables:")
	for name, variable := range variables {
		fmt.Printf("%s = %s (%s)\n", name, variable.Value, variable.Type)
	}
	fmt.Println()
}

func (c *CLI) showHelp() {
	helpText := `
╔══════════════════════════════════════════════════╗
║                 🆘 СПРАВКА                      ║
╠══════════════════════════════════════════════════╣
║ 🧮 АРИФМЕТИКА                                   ║
║   5 + 3, 10 * 2, 15 / 3, 8 - 4                  ║
║                                                 ║
║ 💾 ПЕРЕМЕННЫЕ                                   ║
║   x = 10, name = "Иван", price = 99.99          ║
║                                                 ║
║ 🌐 HTTP ЗАПРОСЫ                                 ║
║   curl https://httpbin.org/get                  ║
║   curl -H "Key: Value" https://api.example.com  ║
║                                                  ║
║ 📊 КОМАНДЫ СИСТЕМЫ                               ║
║   history    - история команд                    ║
║   variables  - список переменных                 ║
║   q       - выход                                ║
╚══════════════════════════════════════════════════╝

📋 ПРИМЕРЫ:
  ▶  (5 + 3) * 2
  ▶  total = 100
  ▶  discount = total * 0.1
  ▶  curl https://jsonplaceholder.typicode.com/posts/1
  ▶  curl -H "Accept: application/json" https://api.github.com
`
	fmt.Println(helpText)
}
