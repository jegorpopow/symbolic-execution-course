package internal

import (
	"container/heap"
	"fmt"
	"go/types"
	"symbolic-execution-course/internal/symbolic"
	"symbolic-execution-course/internal/translator"

	internalssa "symbolic-execution-course/internal/ssa"

	"golang.org/x/tools/go/ssa"
)

const stepsQuota = 5000

type Analyser struct {
	Package      *ssa.Package
	StatesQueue  PriorityQueue
	PathSelector PathSelector
	Results      []Interpreter
	Z3Translator *translator.Z3Translator
}

func Analyse(source, functionName string) []Interpreter {
	function, err := internalssa.NewBuilder().ParseAndBuildSSA(source, functionName)
	if err != nil {
		panic(fmt.Sprintf("analysis: %v", err))
	}
	if len(function.Blocks) == 0 {
		panic("analysis: function has no body")
	}

	a := &Analyser{
		Package:      function.Pkg,
		PathSelector: &DfsPathSelector{},
		Z3Translator: translator.NewZ3Translator(),
	}

	initial := Interpreter{
		Analyser:      a,
		PathCondition: symbolic.NewBoolConstant(true),
		CallStack: []CallStackFrame{{
			Function:    function,
			LocalMemory: make(map[string]symbolic.SymbolicExpression),
			IP:          InstructionPointer{CurrentBlock: function.Blocks[0]},
		}},
	}

	for _, p := range function.Params {
		initial.CallStack[0].LocalMemory[getStringKey(p)] =
			symbolic.NewSymbolicVariable(p.Name(), goTypeToSymbolicType(p.Type()))
	}

	heap.Init(&a.StatesQueue)
	heap.Push(&a.StatesQueue, &Item{value: initial, priority: a.PathSelector.CalculatePriority(initial)})

	stepsPerformed := 0
	for a.StatesQueue.Len() > 0 && stepsPerformed < stepsQuota {
		currentInterpreter := heap.Pop(&a.StatesQueue).(*Item).value
		currentInstruction := currentInterpreter.currentStackframe().IP.CurrentInstruction()
		for _, next := range currentInterpreter.interpretDynamically(currentInstruction) {
			stepsPerformed++
			if next.Terminal {
				a.Results = append(a.Results, next)
				continue
			}
			heap.Push(&a.StatesQueue, &Item{value: next, priority: a.PathSelector.CalculatePriority(next)})
		}
	}

	for a.StatesQueue.Len() > 0 {
		state := heap.Pop(&a.StatesQueue).(*Item).value
		state.terminateWith()
		a.Results = append(a.Results, state)
	}

	return a.Results
}

func goTypeToSymbolicType(t types.Type) symbolic.ExpressionType {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		panic(fmt.Sprintf("UnsupportedType %s", t))
	}
	switch {
	case basic.Info()&types.IsBoolean != 0:
		return symbolic.BoolType
	case basic.Info()&types.IsInteger != 0:
		return symbolic.IntType
	}
	panic(fmt.Sprintf("analysis: unsupported basic type %s", t))
}
