// Package translator содержит реализацию транслятора в Z3
package translator

import (
	"fmt"
	"math/big"
	"symbolic-execution-course/internal/symbolic"

	"github.com/ebukreev/go-z3/z3"
)

// Z3Translator транслирует символьные выражения в Z3 формулы
type Z3Translator struct {
	ctx    *z3.Context
	config *z3.Config
	vars   map[string]z3.Value // Кэш переменных
}

// NewZ3Translator создаёт новый экземпляр Z3 транслятора
func NewZ3Translator() *Z3Translator {
	config := &z3.Config{}
	ctx := z3.NewContext(config)

	return &Z3Translator{
		ctx:    ctx,
		config: config,
		vars:   make(map[string]z3.Value),
	}
}

// GetContext возвращает Z3 контекст
func (zt *Z3Translator) GetContext() interface{} {
	return zt.ctx
}

// Reset сбрасывает состояние транслятора
func (zt *Z3Translator) Reset() {
	zt.vars = make(map[string]z3.Value)
}

// Close освобождает ресурсы
func (zt *Z3Translator) Close() {
	// Z3 контекст закрывается автоматически
}

// TranslateExpression транслирует символьное выражение в Z3
func (zt *Z3Translator) TranslateExpression(expr symbolic.SymbolicExpression) (interface{}, error) {
	return expr.Accept(zt), nil
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// VisitVariable транслирует символьную переменную в Z3
func (zt *Z3Translator) VisitVariable(expr *symbolic.SymbolicVariable) interface{} {
	if v, exists := zt.vars[expr.Name]; exists {
		return v
	}
	v := zt.createZ3Variable(expr.Name, expr.Type())

	if v == nil {
		panic(fmt.Sprintf(
			"Z3Translator: unsupported variable type %s for %q",
			expr.Type(), expr.Name,
		))
	}
	zt.vars[expr.Name] = v
	return v
}

// VisitIntConstant транслирует целочисленную константу в Z3
func (zt *Z3Translator) VisitIntConstant(expr *symbolic.IntConstant) interface{} {
	return zt.ctx.FromBigInt(
		big.NewInt(expr.Value),
		zt.ctx.IntSort(),
	)
}

// VisitBoolConstant транслирует булеву константу в Z3
func (zt *Z3Translator) VisitBoolConstant(expr *symbolic.BoolConstant) interface{} {
	return zt.ctx.FromBool(expr.Value)
}

// VisitBinaryOperation транслирует бинарную операцию в Z3
func (zt *Z3Translator) VisitBinaryOperation(expr *symbolic.BinaryOperation) interface{} {
	left := expr.Left.Accept(zt)
	right := expr.Right.Accept(zt)

	switch expr.Operator {
	case symbolic.ADD:
		return left.(z3.Int).Add(right.(z3.Int))
	case symbolic.SUB:
		return left.(z3.Int).Sub(right.(z3.Int))
	case symbolic.MUL:
		return left.(z3.Int).Mul(right.(z3.Int))
	case symbolic.DIV:
		return left.(z3.Int).Div(right.(z3.Int))
	case symbolic.MOD:
		return left.(z3.Int).Mod(right.(z3.Int))
	case symbolic.EQ:
		return zt.eq(left, right)
	case symbolic.NE:
		return zt.neq(left, right)
	case symbolic.LT:
		return left.(z3.Int).LT(right.(z3.Int))
	case symbolic.LE:
		return left.(z3.Int).LE(right.(z3.Int))
	case symbolic.GT:
		return left.(z3.Int).GT(right.(z3.Int))
	case symbolic.GE:
		return left.(z3.Int).GE(right.(z3.Int))
	default:
		panic(fmt.Sprintf("Z3Translator: unknown binary operator %d", expr.Operator))
	}
}

func (zt *Z3Translator) eq(left, right interface{}) z3.Bool {
	switch l := left.(type) {
	case z3.Int:
		return l.Eq(right.(z3.Int))
	case z3.Bool:
		return l.Eq(right.(z3.Bool))
	default:
		panic(fmt.Sprintf("Z3Translator: EQ unsupported for %T", left))
	}
}

func (zt *Z3Translator) neq(left, right interface{}) z3.Bool {
	switch l := left.(type) {
	case z3.Int:
		return l.NE(right.(z3.Int))
	case z3.Bool:
		return l.NE(right.(z3.Bool))
	default:
		panic(fmt.Sprintf("Z3Translator: NE unsupported for %T", left))
	}
}

// VisitLogicalOperation транслирует логическую операцию в Z3
func (zt *Z3Translator) VisitLogicalOperation(expr *symbolic.LogicalOperation) interface{} {
	// TODO: Реализовать
	// 1. Транслировать все операнды
	// 2. Применить соответствующую логическую операцию

	// Подсказки:
	// - AND: zt.ctx.And(operands...)
	// - OR: zt.ctx.Or(operands...)
	// - NOT: operand.Not() (для единственного операнда)
	// - IMPLIES: antecedent.Implies(consequent)

	operands := make([]z3.Bool, len(expr.Operands))
	for i, op := range expr.Operands {
		operands[i] = op.Accept(zt).(z3.Bool)
	}

	switch expr.Operator {
	case symbolic.AND:
		return operands[0].And(operands[1:]...)
	case symbolic.OR:
		return operands[0].Or(operands[1:]...)
	case symbolic.NOT:
		return operands[0].Not()
	case symbolic.IMPLIES:
		return operands[0].Implies(operands[1])
	default:
		panic(fmt.Sprintf("Z3Translator: unknown logical operator %d", expr.Operator))
	}
}

// Вспомогательные методы

// createZ3Variable создаёт Z3 переменную соответствующего типа
func (zt *Z3Translator) createZ3Variable(name string, exprType symbolic.ExpressionType) z3.Value {
	// TODO: Реализовать (вспомогательный метод)
	// Создать Z3 переменную на основе типа
	switch exprType {
	case symbolic.IntType:
		return zt.ctx.IntConst(name)
	case symbolic.BoolType:
		return zt.ctx.BoolConst(name)
	default:
		return nil
	}
}

// castToZ3Type приводит значение к нужному Z3 типу
func (zt *Z3Translator) castToZ3Type(value interface{}, targetType symbolic.ExpressionType) (z3.Value, error) {
	// TODO: Реализовать (вспомогательный метод)
	// Безопасно привести interface{} к конкретному Z3 типу
	switch targetType {
	case symbolic.IntType:
		if v, ok := value.(z3.Int); ok {
			return v, nil
		}
		return nil, fmt.Errorf("Z3Translator: cannot cast %T to z3.Int", value)
	case symbolic.BoolType:
		if v, ok := value.(z3.Bool); ok {
			return v, nil
		}
		return nil, fmt.Errorf("Z3Translator: cannot cast %T to z3.Bool", value)
	default:
		return nil, fmt.Errorf(
			"Z3Translator: unsupported target type %s",
			targetType,
		)
	}
}
