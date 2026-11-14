package main

import (
	"testing"

	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"

	"github.com/ebukreev/go-z3/z3"
)

func checkWithSolver(t *testing.T, mem *memory.SymbolicMemory, query symbolic.SymbolicExpression) bool {
	t.Helper()
	tr := translator.NewZ3Translator()
	defer tr.Close()

	combined := symbolic.NewLogicalOperation(
		[]symbolic.SymbolicExpression{mem.Constraints(), query},
		symbolic.AND,
	)
	expr, err := tr.TranslateExpression(combined)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	cond, ok := expr.(z3.Bool)
	if !ok {
		t.Fatalf("expected z3.Bool, got %T", expr)
	}
	ctx := tr.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(cond)
	sat, err := solver.Check()
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	return sat
}

func eq(l, r symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	return symbolic.NewBinaryOperation(l, r, symbolic.EQ)
}

func ne(l, r symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	return symbolic.NewBinaryOperation(l, r, symbolic.NE)
}

func and(parts ...symbolic.SymbolicExpression) symbolic.SymbolicExpression {
	return symbolic.NewLogicalOperation(parts, symbolic.AND)
}

func TestStructAliasingSatisfiable(t *testing.T) {
	mem := memory.NewSymbolicMemory()
	foo := symbolic.StructShape("Foo", symbolic.PrimitiveShape(symbolic.IntType))
	f1 := mem.AbstractRef("foo1", foo)
	f2 := mem.AbstractRef("foo2", foo)

	mem.AssignField(f2, 0, symbolic.NewIntConstant(5))
	mem.AssignField(f1, 0, symbolic.NewIntConstant(2))

	query := and(
		eq(f1.Address, f2.Address),
		eq(mem.GetFieldValue(f2, 0), symbolic.NewIntConstant(2)),
	)
	if !checkWithSolver(t, &mem, query) {
		t.Fatal("expected foo1==foo2 && foo2.a==2 to be satisfiable")
	}
}

func TestStructAliasingUnsatisfiable(t *testing.T) {
	mem := memory.NewSymbolicMemory()
	foo := symbolic.StructShape("Foo", symbolic.PrimitiveShape(symbolic.IntType))
	f1 := mem.AbstractRef("foo1", foo)
	f2 := mem.AbstractRef("foo2", foo)

	mem.AssignField(f2, 0, symbolic.NewIntConstant(5))
	mem.AssignField(f1, 0, symbolic.NewIntConstant(2))

	query := and(
		ne(f1.Address, f2.Address),
		eq(mem.GetFieldValue(f2, 0), symbolic.NewIntConstant(2)),
	)
	if checkWithSolver(t, &mem, query) {
		t.Fatal("expected foo1!=foo2 && foo2.a==2 to be unsatisfiable")
	}
}

func TestArrayAliasingSatisfiable(t *testing.T) {
	mem := memory.NewSymbolicMemory()
	arr := symbolic.ArrayShape(symbolic.PrimitiveShape(symbolic.IntType))
	a1 := mem.AbstractRef("arr1", arr)
	a2 := mem.AbstractRef("arr2", arr)

	mem.AssignToArray(a2, 1, symbolic.NewIntConstant(5))
	mem.AssignToArray(a1, 1, symbolic.NewIntConstant(2))

	query := and(
		eq(a1.Address, a2.Address),
		eq(mem.GetFromArray(a2, 1), symbolic.NewIntConstant(2)),
	)
	if !checkWithSolver(t, &mem, query) {
		t.Fatal("expected arr1==arr2 && arr2[1]==2 to be satisfiable")
	}
}

func TestArrayAliasingUnsatisfiable(t *testing.T) {
	mem := memory.NewSymbolicMemory()
	arr := symbolic.ArrayShape(symbolic.PrimitiveShape(symbolic.IntType))
	a1 := mem.AbstractRef("arr1", arr)
	a2 := mem.AbstractRef("arr2", arr)

	mem.AssignToArray(a2, 1, symbolic.NewIntConstant(5))
	mem.AssignToArray(a1, 1, symbolic.NewIntConstant(2))

	query := and(
		ne(a1.Address, a2.Address),
		eq(mem.GetFromArray(a2, 1), symbolic.NewIntConstant(2)),
	)
	if checkWithSolver(t, &mem, query) {
		t.Fatal("expected arr1!=arr2 && arr2[1]==2 to be unsatisfiable")
	}
}

func TestInputRefsMayAlias(t *testing.T) {
	mem := memory.NewSymbolicMemory()
	foo := symbolic.StructShape("Foo", symbolic.PrimitiveShape(symbolic.IntType))
	f1 := mem.AbstractRef("foo1", foo)
	f2 := mem.AbstractRef("foo2", foo)

	query := eq(f1.Address, f2.Address)
	if !checkWithSolver(t, &mem, query) {
		t.Fatal("expected input refs to be possibly aliased")
	}
}

func TestFreshDoesNotAliasInput(t *testing.T) {
	mem := memory.NewSymbolicMemory()
	foo := symbolic.StructShape("Foo", symbolic.PrimitiveShape(symbolic.IntType))
	input := mem.AbstractRef("foo", foo)
	fresh := mem.AllocateOf(foo)

	query := eq(input.Address, fresh.Address)
	if checkWithSolver(t, &mem, query) {
		t.Fatal("expected fresh allocation to differ from input ref")
	}
}
