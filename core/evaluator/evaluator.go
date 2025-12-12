package evaluator

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type Evaluator struct {
	operators map[string]func(float64, float64) (float64, error)
}

func NewEvaluator() *Evaluator {
	calc := &Evaluator{
		operators: make(map[string]func(float64, float64) (float64, error)),
	}

	// Инициализация операторов
	calc.operators["+"] = func(a, b float64) (float64, error) { return a + b, nil }
	calc.operators["-"] = func(a, b float64) (float64, error) { return a - b, nil }
	calc.operators["*"] = func(a, b float64) (float64, error) { return a * b, nil }
	calc.operators["/"] = func(a, b float64) (float64, error) {
		if b == 0 {
			return 0, errors.New("деление на ноль")
		}
		return a / b, nil
	}
	calc.operators["**"] = func(a, b float64) (float64, error) { return math.Pow(a, b), nil }
	calc.operators["^"] = func(a, b float64) (float64, error) { return math.Pow(a, b), nil }
	calc.operators["%"] = func(a, b float64) (float64, error) {
		if b == 0 {
			return 0, errors.New("деление по модулю на ноль")
		}
		return math.Mod(a, b), nil
	}

	return calc
}

// Evaluate - вычисление математического выражения
func (c *Evaluator) Evaluate(expression string) (interface{}, error) {
	// Упрощенный парсер выражений
	result, err := c.safeEval(expression)
	if err != nil {
		// Если безопасное вычисление не сработало, пробуем парсить
		return c.parseExpression(expression)
	}
	return result, nil
}

// safeEval - безопасное вычисление выражения с ограниченным набором операций
func (c *Evaluator) safeEval(expression string) (float64, error) {
	allowedChars := "0123456789+-*/.()%^ "
	for _, char := range expression {
		if !strings.ContainsRune(allowedChars, char) {
			return 0, fmt.Errorf("выражение содержит недопустимые символы")
		}
	}

	// Простая реализация eval для базовых выражений
	return c.evalSimpleExpression(expression)
}

// evalSimpleExpression - вычисление простых выражений без использования eval
func (c *Evaluator) evalSimpleExpression(expr string) (float64, error) {
	// Упрощенная реализация для демонстрации
	// В реальном проекте нужно реализовать полноценный парсер
	tokens := c.tokenize(expr)
	return c.evaluateTokens(tokens)
}

// parseExpression - парсинг и вычисление выражения
func (c *Evaluator) parseExpression(expression string) (float64, error) {
	tokens := c.tokenize(expression)
	return c.evaluateTokens(tokens)
}

// tokenize - разбиение выражения на токены
func (c *Evaluator) tokenize(expression string) []string {
	// Регулярное выражение для токенов: числа, операторы, скобки
	re := regexp.MustCompile(`\d+\.?\d*|[+\-*/^%()]|\w+`)
	return re.FindAllString(expression, -1)
}

// evaluateTokens - вычисление токенизированного выражения
func (c *Evaluator) evaluateTokens(tokens []string) (float64, error) {
	// Конвертируем токены в обратную польскую нотацию и вычисляем
	rpn, err := c.shuntingYard(tokens)
	if err != nil {
		return 0, err
	}
	return c.evaluateRPN(rpn)
}

// shuntingYard - алгоритм сортировочной станции (Dijkstra) для преобразования в ОПН
func (c *Evaluator) shuntingYard(tokens []string) ([]string, error) {
	var output []string
	var stack []string

	precedence := map[string]int{
		"+": 1, "-": 1,
		"*": 2, "/": 2, "%": 2,
		"^": 3, "**": 3,
	}

	for _, token := range tokens {
		switch token {
		case "(":
			stack = append(stack, token)
		case ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, fmt.Errorf("несогласованные скобки")
			}
			stack = stack[:len(stack)-1] // удаляем "("
		default:
			if prec, isOp := precedence[token]; isOp {
				for len(stack) > 0 {
					top := stack[len(stack)-1]
					if topPrec, exists := precedence[top]; exists && prec <= topPrec {
						output = append(output, top)
						stack = stack[:len(stack)-1]
					} else {
						break
					}
				}
				stack = append(stack, token)
			} else {
				// Число или переменная
				output = append(output, token)
			}
		}
	}

	// Выталкиваем оставшиеся операторы из стека
	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, fmt.Errorf("несогласованные скобки")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

// evaluateRPN - вычисление выражения в обратной польской нотации
func (c *Evaluator) evaluateRPN(rpn []string) (float64, error) {
	var stack []float64

	for _, token := range rpn {
		if op, exists := c.operators[token]; exists {
			if len(stack) < 2 {
				return 0, fmt.Errorf("недостаточно операндов для оператора %s", token)
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// Вызываем оператор и получаем результат ИЛИ ошибку
			result, err := op(a, b)
			if err != nil {
				return 0, err // например: "деление на ноль"
			}

			stack = append(stack, result)
		} else {
			// Пробуем парсить число
			val, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0, fmt.Errorf("некорректный токен: %s", token)
			}
			stack = append(stack, val)
		}
	}

	if len(stack) != 1 {
		return 0, fmt.Errorf("некорректное выражение")
	}

	return stack[0], nil
}
