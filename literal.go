package di

// LiteralKind indicates the type of a literal value.
type LiteralKind int

const (
	LiteralString LiteralKind = iota
	LiteralInt
	LiteralFloat
	LiteralBool
	LiteralNull
)

// Literal represents a typed literal value, independent of any parsing format.
type Literal struct {
	Kind  LiteralKind
	Value interface{} // string, int64, float64, bool, or nil
}

// String returns the string value, or empty string if not a string literal.
func (l Literal) String() string {
	if l.Kind == LiteralString {
		if s, ok := l.Value.(string); ok {
			return s
		}
	}
	return ""
}

// Int returns the int64 value, or 0 if not an int literal.
func (l Literal) Int() int64 {
	if l.Kind == LiteralInt {
		if v, ok := l.Value.(int64); ok {
			return v
		}
	}
	return 0
}

// Float returns the float64 value, or 0 if not a float literal.
func (l Literal) Float() float64 {
	if l.Kind == LiteralFloat {
		if v, ok := l.Value.(float64); ok {
			return v
		}
	}
	return 0
}

// Bool returns the bool value, or false if not a bool literal.
func (l Literal) Bool() bool {
	if l.Kind == LiteralBool {
		if v, ok := l.Value.(bool); ok {
			return v
		}
	}
	return false
}

// IsNull returns true if this is a null literal.
func (l Literal) IsNull() bool {
	return l.Kind == LiteralNull
}

// NewStringLiteral creates a string literal.
func NewStringLiteral(s string) Literal {
	return Literal{Kind: LiteralString, Value: s}
}

// NewIntLiteral creates an int literal.
func NewIntLiteral(v int64) Literal {
	return Literal{Kind: LiteralInt, Value: v}
}

// NewFloatLiteral creates a float literal.
func NewFloatLiteral(v float64) Literal {
	return Literal{Kind: LiteralFloat, Value: v}
}

// NewBoolLiteral creates a bool literal.
func NewBoolLiteral(v bool) Literal {
	return Literal{Kind: LiteralBool, Value: v}
}

// NewNullLiteral creates a null literal.
func NewNullLiteral() Literal {
	return Literal{Kind: LiteralNull, Value: nil}
}
