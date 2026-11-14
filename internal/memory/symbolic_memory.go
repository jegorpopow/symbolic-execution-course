package memory

import (
	"fmt"
	"symbolic-execution-course/internal/symbolic"
)

type Memory interface {
	Allocate(tpe symbolic.ExpressionType) *symbolic.Ref
	AllocateOf(shape *symbolic.DataShape) *symbolic.Ref
	AbstractRef(name string, target *symbolic.DataShape) *symbolic.Ref
	AssignField(ref *symbolic.Ref, fieldIdx int, value symbolic.SymbolicExpression)
	GetFieldValue(ref *symbolic.Ref, fieldIdx int) symbolic.SymbolicExpression
	AssignToArray(ref *symbolic.Ref, index int, value symbolic.SymbolicExpression)
	GetFromArray(ref *symbolic.Ref, index int) symbolic.SymbolicExpression
	Constraints() symbolic.SymbolicExpression
}

type cellKey struct {
	shape *symbolic.DataShape
	field int
}

type heapKey struct{ shape *symbolic.DataShape }

type SymbolicMemory struct {
	arenaID uint64
	cursor  *int64
	cells   map[cellKey]symbolic.SymbolicExpression
	heaps   map[heapKey]symbolic.SymbolicExpression
	fresh   []*symbolic.Ref
	inputs  []*symbolic.Ref
}

var arenaSeq uint64

func NewSymbolicMemory() SymbolicMemory {
	arenaSeq++
	var next int64
	return SymbolicMemory{
		arenaID: arenaSeq,
		cursor:  &next,
		cells:   make(map[cellKey]symbolic.SymbolicExpression),
		heaps:   make(map[heapKey]symbolic.SymbolicExpression),
	}
}

func (mem *SymbolicMemory) Allocate(tpe symbolic.ExpressionType) *symbolic.Ref {
	if tpe == symbolic.ArrayType {
		return mem.AllocateOf(symbolic.ArrayShape(symbolic.PrimitiveShape(symbolic.IntType)))
	}
	return mem.AllocateOf(symbolic.PrimitiveShape(tpe))
}

func (mem *SymbolicMemory) AllocateOf(shape *symbolic.DataShape) *symbolic.Ref {
	*mem.cursor++
	addr := symbolic.NewSymbolicVariable(
		fmt.Sprintf("allocated@%d#%d", mem.arenaID, *mem.cursor),
		symbolic.IntType)
	ref := symbolic.NewRef(addr, shape)

	if shape.SymbolicType == symbolic.ArrayType {
		key := heapKey{shape: shape}
		heap := mem.heapCell(key, shape)
		mem.heaps[key] = symbolic.NewArrayStore(heap, addr, shape.DefaultInitialiser())
	} else {
		for i, field := range shape.Fields {
			key := cellKey{shape: shape, field: i}
			mem.cells[key] = symbolic.NewArrayStore(
				mem.cell(key, field), addr, field.DefaultInitialiser())
		}
	}
	mem.fresh = append(mem.fresh, ref)
	return ref
}

func (mem *SymbolicMemory) AbstractRef(name string, target *symbolic.DataShape) *symbolic.Ref {
	ref := symbolic.NewRef(symbolic.NewSymbolicVariable(name, symbolic.IntType), target)
	mem.inputs = append(mem.inputs, ref)
	return ref
}

func (mem *SymbolicMemory) AssignField(ref *symbolic.Ref, i int, v symbolic.SymbolicExpression) {
	ft := ref.Referenced.Fields[i]
	key := cellKey{shape: ref.Referenced, field: i}
	mem.cells[key] = symbolic.NewArrayStore(mem.cell(key, ft), ref.Address, v)
}

func (mem *SymbolicMemory) GetFieldValue(ref *symbolic.Ref, i int) symbolic.SymbolicExpression {
	ft := ref.Referenced.Fields[i]
	key := cellKey{shape: ref.Referenced, field: i}
	return symbolic.NewArraySelect(mem.cell(key, ft), ref.Address)
}

func (mem *SymbolicMemory) AssignToArray(ref *symbolic.Ref, i int, v symbolic.SymbolicExpression) {
	idx := symbolic.NewIntConstant(int64(i))
	arr := ref.Referenced
	key := heapKey{shape: arr}
	heap := mem.heapCell(key, arr)
	elems := symbolic.NewArraySelect(heap, ref.Address)
	mem.heaps[key] = symbolic.NewArrayStore(heap, ref.Address,
		symbolic.NewArrayStore(elems, idx, v))
}

func (mem *SymbolicMemory) GetFromArray(ref *symbolic.Ref, i int) symbolic.SymbolicExpression {
	idx := symbolic.NewIntConstant(int64(i))
	arr := ref.Referenced
	elems := symbolic.NewArraySelect(mem.heapCell(heapKey{shape: arr}, arr), ref.Address)
	return symbolic.NewArraySelect(elems, idx)
}

func (mem *SymbolicMemory) Constraints() symbolic.SymbolicExpression {
	var acc symbolic.SymbolicExpression = symbolic.NewBoolConstant(true)
	zero := symbolic.NewIntConstant(0)

	for _, allocated := range mem.fresh {
		nonNil := symbolic.NewBinaryOperation(allocated.Address, zero, symbolic.NE)
		acc = symbolic.NewLogicalOperation(
			[]symbolic.SymbolicExpression{acc, nonNil}, symbolic.AND)

		for _, incoming := range mem.inputs {
			ne := symbolic.NewBinaryOperation(allocated.Address, incoming.Address, symbolic.NE)
			acc = symbolic.NewLogicalOperation(
				[]symbolic.SymbolicExpression{acc, ne}, symbolic.AND)
		}
	}

	for i := 0; i < len(mem.fresh); i++ {
		for j := i + 1; j < len(mem.fresh); j++ {
			ne := symbolic.NewBinaryOperation(
				mem.fresh[i].Address, mem.fresh[j].Address, symbolic.NE)
			acc = symbolic.NewLogicalOperation(
				[]symbolic.SymbolicExpression{acc, ne}, symbolic.AND)
		}
	}

	return acc
}

func (mem *SymbolicMemory) cell(key cellKey, valueType *symbolic.DataShape) symbolic.SymbolicExpression {
	if v, ok := mem.cells[key]; ok {
		return v
	}
	av := symbolic.NewArrayVariable(
		fmt.Sprintf("field@%d[%d]::%s", mem.arenaID, key.field, key.shape),
		valueType)
	mem.cells[key] = av
	return av
}

func (mem *SymbolicMemory) heapCell(key heapKey, shape *symbolic.DataShape) symbolic.SymbolicExpression {
	if v, ok := mem.heaps[key]; ok {
		return v
	}
	av := symbolic.NewArrayVariable(
		fmt.Sprintf("obj@%d::%s", mem.arenaID, shape),
		shape)
	mem.heaps[key] = av
	return av
}
