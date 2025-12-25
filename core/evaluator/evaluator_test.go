package evaluator

import (
	"math"
	"testing"
)

func TestBasicOperations(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name     string
		expr     string
		expected float64
		wantErr  bool
	}{
		{"Addition", "2+3", 5, false},
		{"Subtraction", "10-7", 3, false},
		{"Multiplication", "4*5", 20, false},
		{"Division", "15/3", 5, false},
		{"Complex expression", "2+3*4", 14, false},
		{"Parentheses", "(2+3)*4", 20, false},
		{"Power", "2^3", 8, false},
		{"Modulo", "10%3", 1, false},
		{"Floating point", "3.5+2.5", 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			resultFloat, ok := result.(float64)
			if !ok {
				t.Errorf("Result is not float64: %T", result)
				return
			}

			if math.Abs(resultFloat-tt.expected) > 0.0001 {
				t.Errorf("Expected %v, got %v", tt.expected, resultFloat)
			}
		})
	}
}

func TestOperatorPrecedence(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		expr     string
		expected float64
	}{
		{"2+3*4", 14},   // multiplication first
		{"10-2*3", 4},   // multiplication first
		{"20/4+3", 8},   // division first
		{"2^3*2", 16},   // power first
		{"(2+3)*4", 20}, // parentheses override
		{"2*(3+4)", 14}, // parentheses override
		{"10-2-3", 5},   // left to right
		{"20/4/2", 2.5}, // left to right
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expr)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			resultFloat := result.(float64)
			if math.Abs(resultFloat-tt.expected) > 0.0001 {
				t.Errorf("Expression %s: expected %v, got %v", tt.expr, tt.expected, resultFloat)
			}
		})
	}
}

func TestErrorCases(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name string
		expr string
	}{
		{"Division by zero", "5/0"},
		{"Modulo by zero", "10%0"},
		{"Unmatched parentheses", "(2+3"},
		{"Empty parentheses content", "()"},
		{"Invalid characters", "2 + abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := eval.Evaluate(tt.expr)
			if err == nil {
				t.Errorf("Expected error for expression: %s", tt.expr)
			}
		})
	}
}

func TestVerySmallNumbers(t *testing.T) {
	eval := NewEvaluator()

	result, err := eval.Evaluate("0.0000001*0.000000000001")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := 0.0000001 * 0.000000000001
	resultFloat := result.(float64)

	if math.Abs(resultFloat-expected) > 1e-20 {
		t.Errorf("Expected %e, got %e", expected, resultFloat)
	}
}

func BenchmarkSimpleAddition(b *testing.B) {
	eval := NewEvaluator()
	for i := 0; i < b.N; i++ {
		eval.Evaluate("2+3")
	}
}

func BenchmarkComplexExpression(b *testing.B) {
	eval := NewEvaluator()
	for i := 0; i < b.N; i++ {
		eval.Evaluate("(10+5)*2-3/3+2^3")
	}
}
