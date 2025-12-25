package applauncher

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewAppLauncher(t *testing.T) {
	launcher := NewAppLauncher()

	if launcher == nil {
		t.Fatal("Expected non-nil launcher")
	}

	if launcher.system != runtime.GOOS {
		t.Errorf("Expected system=%s, got %s", runtime.GOOS, launcher.system)
	}

	if len(launcher.safeDirectories) == 0 {
		t.Error("Expected non-empty safe directories")
	}
}

func TestGetSafeDirectories(t *testing.T) {
	launcher := NewAppLauncher()
	dirs := launcher.GetSafeDirectories()

	if len(dirs) == 0 {
		t.Error("Expected at least one safe directory")
	}

	// Проверяем что текущая директория всегда в списке
	currentDir, _ := os.Getwd()
	found := false
	for _, dir := range dirs {
		if dir == currentDir {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected current directory to be in safe directories")
	}
}

func TestIsSafePath(t *testing.T) {
	launcher := NewAppLauncher()
	currentDir, _ := os.Getwd()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Current directory file", filepath.Join(currentDir, "test.txt"), true},
		{"Relative path", "./test.txt", true},
		{"Relative path in subdir", "./subdir/test.txt", true},
		{"System file", getSystemPath(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := launcher.isSafePath(tt.path)
			if result != tt.expected {
				t.Errorf("isSafePath(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func getSystemPath() string {
	switch runtime.GOOS {
	case "windows":
		return "C:\\Windows\\System32\\config\\SAM"
	case "darwin":
		return "/etc/master.passwd"
	default:
		return "/etc/shadow"
	}
}

func TestOpenBrowser(t *testing.T) {
	launcher := NewAppLauncher()

	tests := []struct {
		name string
		url  string
	}{
		{"HTTP URL", "http://example.com"},
		{"HTTPS URL", "https://google.com"},
		{"URL without protocol", "google.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Замечание: реальное открытие браузера может быть проблематично в CI
			// Этот тест просто проверяет что функция не падает
			// В реальной CI среде нужно мокировать exec.Command

			// Пропускаем реальное выполнение в CI
			if os.Getenv("CI") != "" {
				t.Skip("Skipping browser test in CI environment")
			}

			// На Windows проверяем что команда корректно формируется
			if runtime.GOOS == "windows" {
				// Тест пройдет если функция не паникует
				launcher.OpenBrowser(tt.url)
			}
		})
	}
}

func TestOpenMediaSafety(t *testing.T) {
	launcher := NewAppLauncher()

	// Пытаемся открыть файл вне безопасных директорий
	unsafePath := getSystemPath()

	result := launcher.OpenMedia(unsafePath)
	if result {
		t.Error("Expected OpenMedia to fail for unsafe path")
	}
}

func TestOpenMediaNonExistentFile(t *testing.T) {
	launcher := NewAppLauncher()
	currentDir, _ := os.Getwd()

	// Файл в безопасной директории но не существует
	nonExistent := filepath.Join(currentDir, "this_file_does_not_exist_12345.txt")

	result := launcher.OpenMedia(nonExistent)
	if result {
		t.Error("Expected OpenMedia to fail for non-existent file")
	}
}

func TestOpenFile(t *testing.T) {
	launcher := NewAppLauncher()

	// Создаем временный файл для теста
	currentDir, _ := os.Getwd()
	testFile := filepath.Join(currentDir, "test_file_temp.txt")

	// Создаем файл
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.WriteString("test content")
	f.Close()
	defer os.Remove(testFile)

	// Пропускаем реальное выполнение в CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping file opening test in CI environment")
	}

	// На Windows просто проверяем что функция не паникует
	if runtime.GOOS == "windows" {
		launcher.OpenFile(testFile)
	}
}

func TestWindowsSpecificPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	launcher := NewAppLauncher()

	// Проверяем что Windows системные пути заблокированы
	systemPaths := []string{
		"C:\\Windows\\System32\\config\\SAM",
		"C:\\Windows\\System32\\drivers\\etc\\hosts",
		"C:\\Program Files\\",
	}

	for _, path := range systemPaths {
		if launcher.isSafePath(path) {
			t.Errorf("System path %s should not be safe", path)
		}
	}

	// Проверяем что пути пользователя разрешены
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		userPaths := []string{
			filepath.Join(homeDir, "Downloads", "test.txt"),
			filepath.Join(homeDir, "Documents", "test.doc"),
		}

		for _, path := range userPaths {
			if !launcher.isSafePath(path) {
				t.Errorf("User path %s should be safe", path)
			}
		}
	}
}

func TestRelativePathHandling(t *testing.T) {
	launcher := NewAppLauncher()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Current dir", ".", true},
		{"Parent dir", "..", true},
		{"Relative file", "./test.txt", true},
		{"Relative subdir", "./subdir/file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := launcher.isSafePath(tt.path)
			if result != tt.expected {
				t.Errorf("isSafePath(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestAbsolutePathNormalization(t *testing.T) {
	launcher := NewAppLauncher()
	currentDir, _ := os.Getwd()

	// Относительный и абсолютный путь к одному файлу
	relPath := "./test.txt"
	absPath := filepath.Join(currentDir, "test.txt")

	relResult := launcher.isSafePath(relPath)
	absResult := launcher.isSafePath(absPath)

	if relResult != absResult {
		t.Errorf("Relative and absolute paths to same file should have same safety: rel=%v, abs=%v", relResult, absResult)
	}
}

func BenchmarkIsSafePath(b *testing.B) {
	launcher := NewAppLauncher()
	currentDir, _ := os.Getwd()
	testPath := filepath.Join(currentDir, "test.txt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		launcher.isSafePath(testPath)
	}
}

func BenchmarkOpenBrowserURLNormalization(b *testing.B) {
	for i := 0; i < b.N; i++ {
		url := "google.com"
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}
	}
}
