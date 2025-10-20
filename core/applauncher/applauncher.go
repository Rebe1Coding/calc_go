package applauncher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type AppLauncher struct {
	system          string
	safeDirectories []string
}

func NewAppLauncher() *AppLauncher {
	launcher := &AppLauncher{
		system: runtime.GOOS,
	}
	launcher.safeDirectories = launcher.getSafeDirectories()
	return launcher
}

// getSafeDirectories - определение безопасных директорий
func (a *AppLauncher) getSafeDirectories() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Если не удалось получить домашнюю директорию, используем текущую
		currentDir, _ := os.Getwd()
		return []string{currentDir}
	}

	safeDirs := []string{
		filepath.Join(homeDir, "Downloads"),
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Videos"),
		filepath.Join(homeDir, "Music"),
		filepath.Join(homeDir, "Pictures"),
		filepath.Join(homeDir, "Desktop"),
	}

	// Фильтруем только существующие директории
	var existingDirs []string
	for _, dir := range safeDirs {
		if _, err := os.Stat(dir); err == nil {
			existingDirs = append(existingDirs, dir)
		}
	}

	// Добавляем текущую рабочую директорию
	currentDir, _ := os.Getwd()
	existingDirs = append(existingDirs, currentDir)

	return existingDirs
}

// isSafePath - проверка безопасности пути
func (a *AppLauncher) isSafePath(filePath string) bool {
	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return false
	}

	// Разрешаем относительные пути в текущей директории
	if !filepath.IsAbs(filePath) {
		if _, err := os.Stat(absPath); err == nil {
			return true
		}
	}

	// Проверяем, находится ли файл в безопасной директории
	for _, safeDir := range a.safeDirectories {
		safeAbs, _ := filepath.Abs(safeDir)
		if strings.HasPrefix(absPath, safeAbs) {
			return true
		}
	}

	return false
}

// OpenBrowser - открытие URL в браузере
func (a *AppLauncher) OpenBrowser(url string) bool {
	// Базовая проверка URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	fmt.Printf("Открываю браузер: %s\n", url)
	if a == nil {
		fmt.Printf("Ошибка: AppLauncher не инициализирован\n")
		return false
	}
	fmt.Printf("Система: %s\n", a.system)
	switch a.system {
	case "darwin": // macOS
		cmd := exec.Command("open", url)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Ошибка открытия браузера на macOS: %v\n", err)
			return false
		}
		return true

	case "windows":
		cmd := exec.Command("cmd", "/c", "start", url)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Ошибка открытия браузера на Windows: %v\n", err)
			return false
		}
		return true

	default: // linux и другие unix-like системы
		// Пробуем разные браузеры
		browsers := []string{"xdg-open", "google-chrome", "chromium", "firefox"}
		for _, browser := range browsers {
			cmd := exec.Command(browser, url)
			if err := cmd.Run(); err == nil {
				return true
			}
		}
		fmt.Printf("Не удалось найти подходящий браузер\n")
		return false
	}
}

// OpenMedia - открытие медиафайла
func (a *AppLauncher) OpenMedia(filePath string) bool {
	if !a.isSafePath(filePath) {
		fmt.Printf("Доступ к пути запрещен: %s\n", filePath)
		return false
	}

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Файл не найден: %s\n", filePath)
		return false
	}

	fmt.Printf("Открываю медиафайл: %s\n", filePath)

	switch a.system {
	case "darwin": // macOS
		cmd := exec.Command("open", filePath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Ошибка открытия файла на macOS: %v\n", err)
			return false
		}
		return true

	case "windows":
		cmd := exec.Command("cmd", "/c", "start", "", filePath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Ошибка открытия файла на Windows: %v\n", err)
			return false
		}
		return true

	default: // linux и другие unix-like системы
		// Пробуем разные медиаплееры
		players := []string{"xdg-open", "vlc", "mpv", "mplayer"}
		for _, player := range players {
			cmd := exec.Command(player, filePath)
			if err := cmd.Run(); err == nil {
				return true
			}
		}
		fmt.Printf("Не удалось найти подходящий медиаплеер\n")
		return false
	}
}

// OpenFile - универсальное открытие файла (для любых типов)
func (a *AppLauncher) OpenFile(filePath string) bool {
	if !a.isSafePath(filePath) {
		fmt.Printf("Доступ к пути запрещен: %s\n", filePath)
		return false
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Файл не найден: %s\n", filePath)
		return false
	}

	fmt.Printf("Открываю файл: %s\n", filePath)

	switch a.system {
	case "darwin":
		return exec.Command("open", filePath).Run() == nil
	case "windows":
		return exec.Command("cmd", "/c", "start", "", filePath).Run() == nil
	default:
		return exec.Command("xdg-open", filePath).Run() == nil
	}
}

// GetSafeDirectories - получение списка безопасных директорий (для отладки)
func (a *AppLauncher) GetSafeDirectories() []string {
	return a.safeDirectories
}

// // Пример использования
// func main() {
// 	launcher := NewAppLauncher()

// 	fmt.Printf("Система: %s\n", launcher.system)
// 	fmt.Printf("Безопасные директории: %v\n", launcher.GetSafeDirectories())

// 	// Тестируем открытие браузера
// 	fmt.Println("\nТестируем открытие браузера:")
// 	success := launcher.OpenBrowser("https://google.com")
// 	fmt.Printf("Результат: %v\n", success)

// 	// Тестируем открытие файла (если существует тестовый файл)
// 	testFile := "test.txt"
// 	if _, err := os.Stat(testFile); err == nil {
// 		fmt.Printf("\nТестируем открытие файла %s:\n", testFile)
// 		success := launcher.OpenMedia(testFile)
// 		fmt.Printf("Результат: %v\n", success)
// 	} else {
// 		fmt.Printf("\nТестовый файл %s не найден\n", testFile)
// 	}

// 	// Тестируем проверку безопасности
// 	fmt.Println("\nТестируем проверку безопасности:")
// 	testPaths := []string{
// 		"./safe_file.txt",           // относительный путь
// 		"/etc/passwd",               // системный файл (должен быть запрещен)
// 		filepath.Join(os.TempDir(), "test.txt"), // временная директория
// 	}

// 	for _, path := range testPaths {
// 		isSafe := launcher.isSafePath(path)
// 		fmt.Printf("%s: %v\n", path, isSafe)
// 	}
// }
