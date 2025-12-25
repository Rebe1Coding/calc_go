# PowerShell скрипт для запуска тестов
# Использование: .\test.ps1 [команда]

param(
    [Parameter(Position=0)]
    [string]$Command = "help"
)

# Функции для цветного вывода
function Write-Success {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Green
}

function Write-Info {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Cyan
}

function Write-Warning {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Yellow
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Red
}

function Show-Help {
    Write-Host ""
    Write-Warning "Доступные команды:"
    Write-Host ""
    Write-Success "  test              " -NoNewline
    Write-Host "- Запустить все тесты"
    Write-Success "  test-v            " -NoNewline
    Write-Host "- Тесты с подробным выводом"
    Write-Success "  test-cover        " -NoNewline
    Write-Host "- Тесты с покрытием кода"
    Write-Success "  test-race         " -NoNewline
    Write-Host "- Тесты с проверкой гонки данных"
    Write-Success "  test-bench        " -NoNewline
    Write-Host "- Запустить бенчмарки"
    Write-Success "  test-evaluator    " -NoNewline
    Write-Host "- Тесты калькулятора"
    Write-Success "  test-curl         " -NoNewline
    Write-Host "- Тесты HTTP клиента"
    Write-Success "  test-ai           " -NoNewline
    Write-Host "- Тесты AI агента"
    Write-Success "  test-applauncher  " -NoNewline
    Write-Host "- Тесты запуска приложений"
    Write-Success "  test-interpreter  " -NoNewline
    Write-Host "- Интеграционные тесты"
    Write-Success "  test-short        " -NoNewline
    Write-Host "- Быстрые тесты"
    Write-Success "  clean             " -NoNewline
    Write-Host "- Удалить временные файлы"
    Write-Success "  build             " -NoNewline
    Write-Host "- Собрать проект"
    Write-Success "  run               " -NoNewline
    Write-Host "- Запустить приложение"
    Write-Success "  fmt               " -NoNewline
    Write-Host "- Форматировать код"
    Write-Success "  vet               " -NoNewline
    Write-Host "- Проверить код с go vet"
    Write-Success "  install-deps      " -NoNewline
    Write-Host "- Установить зависимости"
    Write-Success "  all               " -NoNewline
    Write-Host "- Запустить все проверки"
    Write-Host ""
}

function Run-Test {
    Write-Success "Запуск всех тестов..."
    go test ./... -timeout 30s
}

function Run-Test-Verbose {
    Write-Success "Запуск тестов с подробным выводом..."
    go test -v ./... -timeout 30s
}

function Run-Test-Cover {
    Write-Success "Запуск тестов с покрытием..."
    go test ./... -coverprofile=coverage.out -timeout 30s
    
    if ($LASTEXITCODE -eq 0) {
        go tool cover -html=coverage.out -o coverage.html
        Write-Success "Отчет о покрытии создан: coverage.html"
        
        # Открываем в браузере
        Start-Process "coverage.html"
    }
}

function Run-Test-Race {
    Write-Success "Запуск тестов с проверкой гонки данных..."
    go test -race ./... -timeout 30s
}

function Run-Test-Bench {
    Write-Success "Запуск бенчмарков..."
    go test -bench=. -benchmem ./...
}

function Run-Test-Evaluator {
    Write-Success "Тестирование evaluator..."
    go test -v ./core/evaluator/... -timeout 10s
}

function Run-Test-Curl {
    Write-Success "Тестирование curl..."
    go test -v ./core/curl/... -timeout 10s
}

function Run-Test-AI {
    Write-Success "Тестирование AI агента..."
    $env:CI = "true"
    go test -v ./core/ai/... -timeout 10s
    Remove-Item Env:\CI
}

function Run-Test-AppLauncher {
    Write-Success "Тестирование applauncher..."
    $env:CI = "true"
    go test -v ./core/applauncher/... -timeout 10s
    Remove-Item Env:\CI
}

function Run-Test-Interpreter {
    Write-Success "Тестирование interpreter..."
    $env:CI = "true"
    go test -v ./core/interpreter/... -timeout 10s
    Remove-Item Env:\CI
}

function Run-Test-Short {
    Write-Success "Запуск быстрых тестов..."
    go test -short ./... -timeout 10s
}

function Clean {
    Write-Success "Очистка временных файлов..."
    go clean
    
    if (Test-Path "coverage.out") { Remove-Item "coverage.out" }
    if (Test-Path "coverage.html") { Remove-Item "coverage.html" }
    if (Test-Path "calculator_data.json") { Remove-Item "calculator_data.json" }
    
    Write-Success "Очистка завершена"
}

function Build {
    Write-Success "Сборка проекта..."
    
    if (-not (Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin" | Out-Null
    }
    
    go build -o bin\calculator.exe .
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Сборка завершена: bin\calculator.exe"
    } else {
        Write-Error-Custom "Ошибка сборки"
    }
}

function Run-App {
    Write-Success "Запуск приложения..."
    go run .
}

function Format-Code {
    Write-Success "Форматирование кода..."
    go fmt ./...
    Write-Success "Форматирование завершено"
}

function Vet-Code {
    Write-Success "Проверка с помощью go vet..."
    go vet ./...
    Write-Success "Проверка завершена"
}

function Install-Dependencies {
    Write-Success "Установка зависимостей..."
    go mod download
    go mod verify
    Write-Success "Зависимости установлены"
}

function Run-All {
    Write-Success "Запуск всех проверок..."
    
    Format-Code
    Vet-Code
    Run-Test
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Success "✓ Все проверки пройдены успешно!"
    } else {
        Write-Host ""
        Write-Error-Custom "✗ Некоторые проверки не пройдены"
    }
}

# Главная логика
switch ($Command.ToLower()) {
    "help" { Show-Help }
    "test" { Run-Test }
    "test-v" { Run-Test-Verbose }
    "test-cover" { Run-Test-Cover }
    "test-race" { Run-Test-Race }
    "test-bench" { Run-Test-Bench }
    "test-evaluator" { Run-Test-Evaluator }
    "test-curl" { Run-Test-Curl }
    "test-ai" { Run-Test-AI }
    "test-applauncher" { Run-Test-AppLauncher }
    "test-interpreter" { Run-Test-Interpreter }
    "test-short" { Run-Test-Short }
    "clean" { Clean }
    "build" { Build }
    "run" { Run-App }
    "fmt" { Format-Code }
    "vet" { Vet-Code }
    "install-deps" { Install-Dependencies }
    "all" { Run-All }
    default {
        Write-Error-Custom "Неизвестная команда: $Command"
        Show-Help
    }
}