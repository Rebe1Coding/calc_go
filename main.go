package main

import (
	"fmt"
	"log"
	"time"

	"app/core/interpreter"
	"app/core/webrtc"
	"app/ui"

	"github.com/skratchdot/open-golang/open"
)

func main() {
	// Инициализация интерпретатора
	i := interpreter.NewInterpreter()

	// Инициализация WebRTC сервера
	webrtcServer := webrtc.NewServer(":8000")

	// Устанавливаем ссылку на WebRTC сервер в интерпретаторе
	i.SetWebRTCServer(webrtcServer)

	// Инициализация веб-интерфейса калькулятора
	web := ui.NewWebInterface(i)

	calcAddr := ":8080"
	calcURL := "http://localhost" + calcAddr

	// Запускаем WebRTC сервер в отдельной горутине
	go func() {
		if err := webrtcServer.Start(); err != nil {
			log.Fatalf("❌ WebRTC Server error: %v", err)
		}
	}()

	// Открываем браузер с калькулятором
	go func() {
		time.Sleep(500 * time.Millisecond)
		err := open.Run(calcURL)
		if err != nil {
			log.Printf("❌ Не удалось открыть браузер: %v", err)
		}
	}()

	fmt.Printf("🌐 Калькулятор открывается в браузере: %s\n", calcURL)
	fmt.Printf("📞 WebRTC доступен по адресу: http://localhost:8000/webrtc/\n")
	fmt.Println("Нажмите Ctrl+C для выхода.")

	// Запускаем веб-сервер калькулятора (блокирующий вызов)
	err := web.Start(calcAddr)
	if err != nil {
		log.Fatal(err)
	}
}
