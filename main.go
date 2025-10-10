package main

import (
	"app/service/calculator"
	"app/service/storage"
	cli "app/ui"
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
