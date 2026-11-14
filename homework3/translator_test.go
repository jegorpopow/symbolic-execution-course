package main

import (
	"testing"

	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"

	"github.com/ebukreev/go-z3/z3"
)

func intVar(name string) *symbolic.SymbolicVariable {
	return symbolic.NewSymbolicVariable(name, symbolic.IntType)
}

func boolVar(name string) *symbolic.SymbolicVariable {
	return symbolic.NewSymbolicVariable(name, symbolic.BoolType)
}

func intLit(n int64) *symbolic.IntConstant {
	return symbolic.NewIntConstant(n)
}

func binop(l, r symbolic.SymbolicExpression, op symbolic.BinaryOperator) *symbolic.BinaryOperation {
	return symbolic.NewBinaryOperation(l, r, op)
}

func logic(ops []symbolic.SymbolicExpression, op symbolic.LogicalOperator) *symbolic.LogicalOperation {
	return symbolic.NewLogicalOperation(ops, op)
}

func checkSat(t *testing.T, tr *translator.Z3Translator, expr symbolic.SymbolicExpression) bool {
	t.Helper()
	translated, err := tr.TranslateExpression(expr)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	ctx := tr.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(translated.(z3.Bool))
	sat, err := solver.Check()
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	return sat
}

func TestVariableTypes(t *testing.T) {
	tr := translator.NewZ3Translator()
	defer tr.Close()

	v, err := tr.TranslateExpression(intVar("x"))
	if err != nil {
		t.Fatalf("translate int var: %v", err)
	}
	if _, ok := v.(z3.Int); !ok {
		t.Fatalf("expected z3.Int for int variable, got %T", v)
	}

	v, err = tr.TranslateExpression(boolVar("b"))
	if err != nil {
		t.Fatalf("translate bool var: %v", err)
	}
	if _, ok := v.(z3.Bool); !ok {
		t.Fatalf("expected z3.Bool for bool variable, got %T", v)
	}
}

func TestConstantContradiction(t *testing.T) {
	tr := translator.NewZ3Translator()
	defer tr.Close()

	x := intVar("x")
	constraint := logic([]symbolic.SymbolicExpression{
		binop(x, intLit(42), symbolic.EQ),
		binop(x, intLit(43), symbolic.EQ),
	}, symbolic.AND)

	if checkSat(t, tr, constraint) {
		t.Fatal("x == 42 && x == 43 should be unsatisfiable")
	}
}

func TestArithmeticSystem(t *testing.T) {
	tr := translator.NewZ3Translator()
	defer tr.Close()

	x, y := intVar("x"), intVar("y")
	system := logic([]symbolic.SymbolicExpression{
		binop(binop(x, y, symbolic.ADD), intLit(10), symbolic.EQ),
		binop(binop(x, y, symbolic.SUB), intLit(4), symbolic.EQ),
	}, symbolic.AND)

	if !checkSat(t, tr, system) {
		t.Fatal("x + y == 10 && x - y == 4 should be satisfiable")
	}
}

func TestUnsatisfiableComparison(t *testing.T) {
	tr := translator.NewZ3Translator()
	defer tr.Close()

	x := intVar("x")
	bad := logic([]symbolic.SymbolicExpression{
		binop(x, intLit(5), symbolic.GT),
		binop(x, intLit(3), symbolic.LT),
	}, symbolic.AND)

	if checkSat(t, tr, bad) {
		t.Fatal("x > 5 && x < 3 should be unsatisfiable")
	}
}

func TestLogicalOperations(t *testing.T) {
	tr := translator.NewZ3Translator()
	defer tr.Close()

	a := boolVar("a")
	notA := logic([]symbolic.SymbolicExpression{a}, symbolic.NOT)

	if !checkSat(t, tr, logic([]symbolic.SymbolicExpression{a, notA}, symbolic.OR)) {
		t.Fatal("a || !a should be satisfiable")
	}
	if checkSat(t, tr, logic([]symbolic.SymbolicExpression{a, notA}, symbolic.AND)) {
		t.Fatal("a && !a should be unsatisfiable")
	}
}

func TestImplication(t *testing.T) {
	tr := translator.NewZ3Translator()
	defer tr.Close()

	a := boolVar("a")
	taut := logic([]symbolic.SymbolicExpression{a, a}, symbolic.IMPLIES)
	negated := logic([]symbolic.SymbolicExpression{taut}, symbolic.NOT)

	if checkSat(t, tr, negated) {
		t.Fatal("!(a => a) should be unsatisfiable")
	}
}
