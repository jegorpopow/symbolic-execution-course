// Package symbolic содержит конкретные реализации символьных выражений
package symbolic

import (
	"fmt"
	"strings"
)

// SymbolicExpression - базовый интерфейс для всех символьных выражений
type SymbolicExpression interface {
	// Type возвращает тип выражения
	Type() ExpressionType

	// String возвращает строковое представление выражения
	String() string

	// Accept принимает visitor для обхода дерева выражений
	Accept(visitor Visitor) interface{}
}

// SymbolicVariable представляет символьную переменную
type SymbolicVariable struct {
	Name     string
	ExprType ExpressionType
}

// NewSymbolicVariable создаёт новую символьную переменную
func NewSymbolicVariable(name string, exprType ExpressionType) *SymbolicVariable {
	return &SymbolicVariable{
		Name:     name,
		ExprType: exprType,
	}
}

// Type возвращает тип переменной
func (sv *SymbolicVariable) Type() ExpressionType {
	return sv.ExprType
}

// String возвращает строковое представление переменной
func (sv *SymbolicVariable) String() string {
	return sv.Name
}

// Accept реализует Visitor pattern
func (sv *SymbolicVariable) Accept(visitor Visitor) interface{} {
	return visitor.VisitVariable(sv)
}

// IntConstant представляет целочисленную константу
type IntConstant struct {
	Value int64
}

// NewIntConstant создаёт новую целочисленную константу
func NewIntConstant(value int64) *IntConstant {
	return &IntConstant{Value: value}
}

// Type возвращает тип константы
func (ic *IntConstant) Type() ExpressionType {
	return IntType
}

// String возвращает строковое представление константы
func (ic *IntConstant) String() string {
	return fmt.Sprintf("%d", ic.Value)
}

// Accept реализует Visitor pattern
func (ic *IntConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitIntConstant(ic)
}

// BoolConstant представляет булеву константу
type BoolConstant struct {
	Value bool
}

// NewBoolConstant создаёт новую булеву константу
func NewBoolConstant(value bool) *BoolConstant {
	return &BoolConstant{Value: value}
}

// Type возвращает тип константы
func (bc *BoolConstant) Type() ExpressionType {
	return BoolType
}

// String возвращает строковое представление константы
func (bc *BoolConstant) String() string {
	return fmt.Sprintf("%t", bc.Value)
}

// Accept реализует Visitor pattern
func (bc *BoolConstant) Accept(visitor Visitor) interface{} {
	return visitor.VisitBoolConstant(bc)
}

// BinaryOperation представляет бинарную операцию
type BinaryOperation struct {
	Left     SymbolicExpression
	Right    SymbolicExpression
	Operator BinaryOperator
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// NewBinaryOperation создаёт новую бинарную операцию
func (bo *BinaryOperation) validate() error {
	lt, rt := bo.Left.Type(), bo.Right.Type()

	switch bo.Operator {
	case ADD, SUB, MUL, DIV, MOD:
		if lt != IntType || rt != IntType {
			return fmt.Errorf(
				"operator %s requires int operands, got %s and %s",
				bo.Operator, lt, rt,
			)
		}

	case EQ, NE:
		if lt != rt {
			return fmt.Errorf(
				"operator %s requires operands of the same type, got %s and %s",
				bo.Operator, lt, rt,
			)
		}

	case LT, LE, GT, GE:
		if lt != IntType || rt != IntType {
			return fmt.Errorf(
				"operator %s requires int operands, got %s and %s",
				bo.Operator, lt, rt,
			)
		}

	default:
		return fmt.Errorf("unknown binary operator %d", bo.Operator)
	}

	return nil
}

func NewBinaryOperation(left, right SymbolicExpression, op BinaryOperator) *BinaryOperation {
	bo := &BinaryOperation{
		Left:     left,
		Right:    right,
		Operator: op,
	}
	if err := bo.validate(); err != nil {
		panic(fmt.Sprintf("invalid binary operation: %v", err))
	}
	return bo
}

// Type возвращает результирующий тип операции
func (bo *BinaryOperation) Type() ExpressionType {
	switch bo.Operator {
	case ADD, SUB, MUL, DIV, MOD:
		return IntType
	case EQ, NE, LT, LE, GT, GE:
		return BoolType
	default:
		panic(fmt.Sprintf("unknown binary operator: %d", bo.Operator))
	}
}

// String возвращает строковое представление операции
func (bo *BinaryOperation) String() string {
	return fmt.Sprintf(
		"(%s %s %s)",
		bo.Left.String(),
		bo.Operator.String(),
		bo.Right.String(),
	)
}

// Accept реализует Visitor pattern
func (bo *BinaryOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitBinaryOperation(bo)
}

// LogicalOperation представляет логическую операцию
type LogicalOperation struct {
	Operands []SymbolicExpression
	Operator LogicalOperator
}

// TODO: Реализуйте следующие методы в рамках домашнего задания

// NewLogicalOperation создаёт новую логическую операцию
func NewLogicalOperation(operands []SymbolicExpression, op LogicalOperator) *LogicalOperation {
	lo := &LogicalOperation{
		Operands: operands,
		Operator: op,
	}
	if err := lo.validate(); err != nil {
		panic(fmt.Sprintf("invalid logical operation: %v", err))
	}
	return lo
}

func (lo *LogicalOperation) validate() error {
	switch lo.Operator {
	case NOT:
		if len(lo.Operands) != 1 {
			return fmt.Errorf("NOT requires exactly 1 operand, got %d", len(lo.Operands))
		}
		if lo.Operands[0].Type() != BoolType {
			return fmt.Errorf(
				"NOT requires bool operand, got %s",
				lo.Operands[0].Type(),
			)
		}

	case AND, OR:
		if len(lo.Operands) < 2 {
			return fmt.Errorf(
				"%s requires at least 2 operands, got %d",
				lo.Operator, len(lo.Operands),
			)
		}
		for i, operand := range lo.Operands {
			if operand.Type() != BoolType {
				return fmt.Errorf(
					"%s operand %d must be bool, got %s",
					lo.Operator, i, operand.Type(),
				)
			}
		}

	case IMPLIES:
		if len(lo.Operands) != 2 {
			return fmt.Errorf(
				"IMPLIES requires exactly 2 operands, got %d",
				len(lo.Operands),
			)
		}
		for i, operand := range lo.Operands {
			if operand.Type() != BoolType {
				return fmt.Errorf(
					"IMPLIES operand %d must be bool, got %s",
					i, operand.Type(),
				)
			}
		}

	default:
		return fmt.Errorf("unknown logical operator %d", lo.Operator)
	}

	return nil
}

// Type возвращает тип логической операции (всегда bool)
func (lo *LogicalOperation) Type() ExpressionType {
	return BoolType
}

// String возвращает строковое представление логической операции
func (lo *LogicalOperation) String() string {
	switch lo.Operator {
	case NOT:
		return "!" + lo.Operands[0].String()

	case AND, OR:
		parts := make([]string, len(lo.Operands))
		for i, operand := range lo.Operands {
			parts[i] = operand.String()
		}
		return "(" + strings.Join(parts, " "+lo.Operator.String()+" ") + ")"

	case IMPLIES:
		return fmt.Sprintf(
			"(%s => %s)",
			lo.Operands[0].String(),
			lo.Operands[1].String(),
		)

	default:
		panic(fmt.Sprintf("unknown logical operator: %d", lo.Operator))
	}
}

// Accept реализует Visitor pattern
func (lo *LogicalOperation) Accept(visitor Visitor) interface{} {
	return visitor.VisitLogicalOperation(lo)
}

// Операторы для бинарных выражений
type BinaryOperator int

const (
	// Арифметические операторы
	ADD BinaryOperator = iota
	SUB
	MUL
	DIV
	MOD

	// Операторы сравнения
	EQ // равно
	NE // не равно
	LT // меньше
	LE // меньше или равно
	GT // больше
	GE // больше или равно
)

// String возвращает строковое представление оператора
func (op BinaryOperator) String() string {
	switch op {
	case ADD:
		return "+"
	case SUB:
		return "-"
	case MUL:
		return "*"
	case DIV:
		return "/"
	case MOD:
		return "%"
	case EQ:
		return "=="
	case NE:
		return "!="
	case LT:
		return "<"
	case LE:
		return "<="
	case GT:
		return ">"
	case GE:
		return ">="
	default:
		return "unknown"
	}
}

// Логические операторы
type LogicalOperator int

const (
	AND LogicalOperator = iota
	OR
	NOT
	IMPLIES
)

// String возвращает строковое представление логического оператора
func (op LogicalOperator) String() string {
	switch op {
	case AND:
		return "&&"
	case OR:
		return "||"
	case NOT:
		return "!"
	case IMPLIES:
		return "=>"
	default:
		return "unknown"
	}
}

type Ref struct {
	// TODO: Выбрать и написать внутреннее представление символьной ссылки
}

func (ref *Ref) Type() ExpressionType {
	panic("не реализовано")
}

func (ref *Ref) String() string {
	panic("не реализовано")
}

func (ref *Ref) Accept(visitor Visitor) interface{} {
	panic("не реализовано")
}

// TODO: Добавьте дополнительные типы выражений по необходимости:
// - UnaryOperation (унарные операции: -x, !x)
// - ArrayAccess (доступ к элементам массива: arr[index])
// - FunctionCall (вызовы функций: f(x, y))
// - ConditionalExpression (тернарный оператор: condition ? true_expr : false_expr)
