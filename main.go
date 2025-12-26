package main

import (
	"app/core/interpreter"
	"app/ui"
	"fmt"
	"log"
	"time"

	"github.com/skratchdot/open-golang/open"
)

func main() {
	// Инициализация интерпретатора
	i := interpreter.NewInterpreter()

	web := ui.NewWebInterface(i)

	calcAddr := ":8080"
	calcURL := "http://localhost" + calcAddr

	go func() {
		time.Sleep(500 * time.Millisecond)
		err := open.Run(calcURL)
		if err != nil {
			log.Printf("❌ Не удалось открыть браузер: %v", err)
		}
	}()

	fmt.Printf("🌐 Калькулятор открывается в браузере: %s\n", calcURL)
	fmt.Println("Нажмите Ctrl+C для выхода.")

	err := web.Start(calcAddr)
	if err != nil {
		log.Fatal(err)
	}
}
