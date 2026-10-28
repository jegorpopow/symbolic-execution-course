package internal

import (
	"fmt"
	"go/constant"
	"go/token"
	"maps"
	"slices"
	"symbolic-execution-course/internal/memory"
	"symbolic-execution-course/internal/symbolic"

	"github.com/ebukreev/go-z3/z3"
	"golang.org/x/tools/go/ssa"
)

type Interpreter struct {
	CallStack     []CallStackFrame
	Analyser      *Analyser
	PathCondition symbolic.SymbolicExpression
	Heap          memory.Memory
	Terminal      bool
}
type InstructionPointer struct {
	CurrentBlock     *ssa.BasicBlock
	PreviousBlock    *ssa.BasicBlock
	InstructionIndex int
}

type CallStackFrame struct {
	Function    *ssa.Function
	LocalMemory map[string]symbolic.SymbolicExpression
	ReturnValue symbolic.SymbolicExpression
	IP          InstructionPointer
}

func (ip *InstructionPointer) Advance() {
	ip.InstructionIndex++
}

func (ip *InstructionPointer) JumpTo(block *ssa.BasicBlock) {
	ip.PreviousBlock = ip.CurrentBlock
	ip.CurrentBlock = block
	ip.InstructionIndex = 0
}

func (ip *InstructionPointer) PredecessorIndex() int {
	for i, pred := range ip.CurrentBlock.Preds {
		if pred == ip.PreviousBlock {
			return i
		}
	}
	return -1
}

func (ip *InstructionPointer) Phi() (*ssa.Phi, bool) {
	if ip.InstructionIndex >= len(ip.CurrentBlock.Instrs) {
		return nil, false
	}
	phi, ok := ip.CurrentBlock.Instrs[ip.InstructionIndex].(*ssa.Phi)
	return phi, ok
}

func (ip *InstructionPointer) CurrentInstruction() ssa.Instruction {
	return ip.CurrentBlock.Instrs[ip.InstructionIndex]
}

func getStringKey(value ssa.Value) string {
	return value.Name()
}

func (s *Interpreter) terminateWith() {
	s.Terminal = true
}

func (interpreter *Interpreter) currentStackframe() *CallStackFrame {
	return &interpreter.CallStack[len(interpreter.CallStack)-1]
}

func (interpreter *Interpreter) Clone() Interpreter {
	result := *interpreter
	result.CallStack = slices.Clone(interpreter.CallStack)
	for i := range result.CallStack {
		result.CallStack[i].LocalMemory = maps.Clone(result.CallStack[i].LocalMemory)
	}
	return result
}

func (interpreter *Interpreter) moveTo(block *ssa.BasicBlock) {
	frame := interpreter.currentStackframe()
	frame.IP.JumpTo(block)

	for phi, ok := frame.IP.Phi(); ok; phi, ok = frame.IP.Phi() {
		frame.LocalMemory[getStringKey(phi)] =
			interpreter.resolveExpression(phi.Edges[frame.IP.PredecessorIndex()])
		frame.IP.Advance()
	}
}

func (interpreter *Interpreter) interpretDynamically(element ssa.Instruction) []Interpreter {
	next := interpreter.Clone()
	currentFrame := next.currentStackframe()

	switch instruction := element.(type) {
	case *ssa.If:
		succs := currentFrame.IP.CurrentBlock.Succs
		if len(succs) != 2 {
			panic(fmt.Sprintf("Conditional jump issues, wrong number of exits %v", instruction))
		}

		condition := next.resolveExpression(instruction.Cond)
		var states []Interpreter

		next.PathCondition = symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{next.PathCondition, condition},
			symbolic.AND,
		)
		next.moveTo(succs[0])
		if next.isReachablePath(next.PathCondition) {
			states = append(states, next)
		}

		elseState := interpreter.Clone()
		elseState.PathCondition = symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{
				elseState.PathCondition,
				symbolic.NewLogicalOperation(
					[]symbolic.SymbolicExpression{condition},
					symbolic.NOT,
				),
			},
			symbolic.AND,
		)
		elseState.moveTo(succs[1])
		if elseState.isReachablePath(elseState.PathCondition) {
			states = append(states, elseState)
		}

		return states
	case *ssa.Jump:
		succs := currentFrame.IP.CurrentBlock.Succs
		if len(succs) != 1 {
			panic(fmt.Sprintf("Goto issues, many exists %v", instruction))
		}
		next.moveTo(succs[0])
	case *ssa.Return:
		results := instruction.Results
		if len(results) > 1 {
			panic(fmt.Sprintf("Return issues, multiple returns %v", instruction))
		}
		if len(results) == 1 {
			currentFrame.ReturnValue = next.resolveExpression(results[0])
		}
		next.terminateWith()
	case *ssa.Panic:
		next.terminateWith()
	case *ssa.MakeInterface:
		currentFrame.IP.Advance()
	case *ssa.BinOp:
		currentFrame.LocalMemory[getStringKey(instruction)] = next.resolveExpression(instruction)
		currentFrame.IP.Advance()
	case *ssa.UnOp:
		currentFrame.LocalMemory[getStringKey(instruction)] = next.resolveExpression(instruction)
		currentFrame.IP.Advance()
	default:
		panic(fmt.Sprintf("Unsupported instruction %v", instruction))
	}
	return []Interpreter{next}
}

func (interpreter *Interpreter) isReachablePath(pathCondition symbolic.SymbolicExpression) bool {
	translated, err := interpreter.Analyser.Z3Translator.TranslateExpression(pathCondition)
	if err != nil {
		panic(fmt.Sprintf("Translation failure: %v", err))
	}
	ctx := interpreter.Analyser.Z3Translator.GetContext().(*z3.Context)
	solver := z3.NewSolver(ctx)
	solver.Assert(translated.(z3.Bool))
	sat, err := solver.Check()
	if err != nil {
		panic(fmt.Sprintf("Solver failure: %v", err))
	}
	return sat
}

func (interpreter *Interpreter) resolveExpression(value ssa.Value) symbolic.SymbolicExpression {
	if literal, ok := value.(*ssa.Const); ok {
		return literalExpression(literal)
	}
	if cached, ok := interpreter.currentStackframe().LocalMemory[getStringKey(value)]; ok {
		return cached
	}
	switch instruction := value.(type) {
	case *ssa.BinOp:
		return interpreter.binaryExpression(instruction)
	case *ssa.UnOp:
		return interpreter.unaryExpression(instruction)
	}
	panic(fmt.Sprintf("Unsupported expression %s", getStringKey(value)))
}

func (interpreter *Interpreter) unaryExpression(instruction *ssa.UnOp) symbolic.SymbolicExpression {
	operand := interpreter.resolveExpression(instruction.X)
	switch instruction.Op {
	case token.SUB:
		return symbolic.NewBinaryOperation(symbolic.NewIntConstant(0), operand, symbolic.SUB)
	case token.NOT:
		return symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{operand},
			symbolic.NOT,
		)
	case token.XOR:
		negated := symbolic.NewBinaryOperation(symbolic.NewIntConstant(0), operand, symbolic.SUB)
		return symbolic.NewBinaryOperation(negated, symbolic.NewIntConstant(1), symbolic.SUB)
	}
	panic(fmt.Sprintf("Unsupported unary %v", instruction.Op))
}

func literalExpression(literal *ssa.Const) symbolic.SymbolicExpression {
	switch literal.Value.Kind() {
	case constant.Bool:
		return symbolic.NewBoolConstant(constant.BoolVal(literal.Value))
	case constant.Int:
		number, _ := constant.Int64Val(literal.Value)
		return symbolic.NewIntConstant(number)
	}
	panic(fmt.Sprintf("Unsupported constant kind %v", literal.Value.Kind()))
}

var binaryOperators = map[token.Token]symbolic.BinaryOperator{
	token.ADD: symbolic.ADD,
	token.SUB: symbolic.SUB,
	token.MUL: symbolic.MUL,
	token.QUO: symbolic.DIV,
	token.REM: symbolic.MOD,
	token.EQL: symbolic.EQ,
	token.NEQ: symbolic.NE,
	token.LSS: symbolic.LT,
	token.LEQ: symbolic.LE,
	token.GTR: symbolic.GT,
	token.GEQ: symbolic.GE,
}

var logicalOperators = map[token.Token]symbolic.LogicalOperator{
	token.LAND: symbolic.AND,
	token.LOR:  symbolic.OR,
}

func (interpreter *Interpreter) binaryExpression(instruction *ssa.BinOp) symbolic.SymbolicExpression {
	lhs := interpreter.resolveExpression(instruction.X)
	rhs := interpreter.resolveExpression(instruction.Y)
	if op, ok := binaryOperators[instruction.Op]; ok {
		return symbolic.NewBinaryOperation(lhs, rhs, op)
	}
	if op, ok := logicalOperators[instruction.Op]; ok {
		return symbolic.NewLogicalOperation([]symbolic.SymbolicExpression{lhs, rhs}, op)
	}
	panic(fmt.Sprintf("Unsupported binary %v", instruction.Op))
}

func (interpreter Interpreter) String() string {
	status := "unknown"
	if interpreter.Terminal {
		status = "haulted"
	}
	result := status + " if " + interpreter.PathCondition.String()
	if len(interpreter.CallStack) > 0 {
		top := interpreter.CallStack[len(interpreter.CallStack)-1]
		if top.ReturnValue != nil {
			result += ", return value = " + top.ReturnValue.String()
		}
	}
	return result
}
