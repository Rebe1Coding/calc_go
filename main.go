package main

import (
	"fmt"
	"log"
	"time"

	"app/core/interpreter"
	"app/ui"

	"github.com/skratchdot/open-golang/open"
)

func main() {
	i := interpreter.NewInterpreter()
	web := ui.NewWebInterface(i)

	addr := ":8080"
	url := "http://localhost" + addr

	go func() {
		time.Sleep(500 * time.Millisecond) // ждём запуска сервера
		err := open.Run(url)
		if err != nil {
			log.Printf("❌ Не удалось открыть браузер: %v", err)
		}
	}()

	fmt.Printf("🌐 Открывается в браузере: %s\n", url)
	fmt.Println("Нажмите Ctrl+C для выхода.")

	err := web.Start(addr)
	if err != nil {
		log.Fatal(err)
	}
}
