package symbolic

// Like SymbolicType, but better
type DataShape struct {
	SymbolicType ExpressionType
	Name         string
	Fields       []*DataShape
	Element      *DataShape
}

func PrimitiveShape(kind ExpressionType) *DataShape {
	return &DataShape{SymbolicType: kind}
}

func StructShape(name string, fields ...*DataShape) *DataShape {
	return &DataShape{SymbolicType: StructType, Name: name, Fields: fields}
}

func ArrayShape(element *DataShape) *DataShape {
	return &DataShape{SymbolicType: ArrayType, Element: element}
}

func RefShape(target *DataShape) *DataShape {
	return &DataShape{SymbolicType: RefType, Element: target}
}

func (l *DataShape) DefaultInitialiser() SymbolicExpression {
	switch l.SymbolicType {
	case BoolType:
		return NewBoolConstant(false)
	case RefType:
		return NewRef(NewIntConstant(0), l.Element)
	case FloatType:
		return NewFloatConstant(0)
	case ArrayType:
		return NewConstantArray(l.Element, l.Element.DefaultInitialiser())
	default:
		return NewIntConstant(0)
	}
}
