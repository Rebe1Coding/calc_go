package main

import (
	"calculator/service/calculator"
	"calculator/service/storage"
	"calculator/ui"
)

func main() {
	// Инициализация хранилища
	storage := storage.NewStorage("calculator_state.json")

	// Инициализация бизнес-логики
	calculator := calculator.NewCalculator(storage)

	// Инициализация CLI
	cli := cli.NewCLI(calculator)

	// Запуск приложения
	cli.Start()
}
