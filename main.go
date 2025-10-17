package main

import (
	"app/core/interpreter"
	"app/ui"
	"fmt"
	"os"
)

func main() {

	// Инициализация калькулятора
	calc := interpreter.NewInterpreter()

	// Создание консольного интерфейса
	console := ui.NewConsoleInterface(calc)

	// Запуск интерфейса
	if err := console.Run(); err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

}
