package ir

import (
	"go/types"
	"time"
)

// Container is the fully resolved intermediate representation of a DI container.
type Container struct {
	Services   map[string]*Service
	Parameters map[string]*Parameter
	Tags       map[string]*Tag

	// Computed fields
	ServiceOrder []string // Topologically sorted service IDs
}

// Service is a fully resolved service definition.
type Service struct {
	ID   string
	Type types.Type

	// Construction
	Constructor *Constructor
	Alias       *Service // If this is an alias, points to target service

	// Lifecycle
	Shared bool
	Public bool

	// Decoration
	Decorates  *Service   // Service being decorated (if decorator)
	Decorators []*Service // Services decorating this one
	Priority   int        // Decoration priority

	// Tags
	Tags []*ServiceTag

	// Computed
	Dependencies  []*Service // Direct dependencies (resolved)
	CanError      bool       // Whether getter can return error
	BuildCanError bool       // Whether build function can return error
}

// IsAlias returns true if this service is an alias.
func (s *Service) IsAlias() bool {
	return s.Alias != nil
}

// IsDecorator returns true if this service decorates another.
func (s *Service) IsDecorator() bool {
	return s.Decorates != nil
}

// Constructor defines how a service is constructed.
type Constructor struct {
	Kind ConstructorKind
	Func *types.Func // For FuncConstructor
	Args []*Argument

	// For method constructors
	Receiver *Service // The service whose method is called

	// Signature info
	Params       []types.Type
	ResultType   types.Type
	ReturnsError bool
}

// ConstructorKind indicates the type of constructor.
type ConstructorKind int

const (
	FuncConstructor ConstructorKind = iota
	MethodConstructor
)

// Argument is a resolved constructor argument.
type Argument struct {
	Kind ArgumentKind
	Type types.Type // Expected parameter type

	// Value based on kind
	Service   *Service      // For ServiceRef
	Parameter *Parameter    // For ParamRef
	Tag       *Tag          // For Tagged
	Literal   LiteralValue  // For Literal
	Inner     bool          // For Inner (decorator's inner service)
}

// ArgumentKind indicates the type of argument.
type ArgumentKind int

const (
	LiteralArg ArgumentKind = iota
	ServiceRefArg
	InnerArg
	ParamRefArg
	TaggedArg
)

// LiteralValue holds a typed literal value.
type LiteralValue struct {
	Type  LiteralType
	Value interface{} // string, int64, float64, bool, or nil
}

// LiteralType indicates the type of literal.
type LiteralType int

const (
	StringLiteral LiteralType = iota
	IntLiteral
	FloatLiteral
	BoolLiteral
	NullLiteral
	DurationLiteral
)

// Parameter is a resolved parameter definition.
type Parameter struct {
	Name  string
	Type  types.Type
	Value LiteralValue
}

// Tag is a resolved tag definition.
type Tag struct {
	Name        string
	ElementType types.Type
	SortBy      string
	Services    []*Service // Services with this tag (sorted by priority)
}

// ServiceTag is a tag attached to a service.
type ServiceTag struct {
	Tag        *Tag
	Attributes map[string]interface{}
}

// GetterMethod returns the Provider method name for this parameter's type.
func (p *Parameter) GetterMethod() string {
	switch {
	case isString(p.Type):
		return "GetString"
	case isInt(p.Type):
		return "GetInt"
	case isBool(p.Type):
		return "GetBool"
	case isFloat64(p.Type):
		return "GetFloat"
	case isDuration(p.Type):
		return "GetDuration"
	default:
		return ""
	}
}

func isString(t types.Type) bool {
	return types.Identical(t, types.Typ[types.String])
}

func isInt(t types.Type) bool {
	return types.Identical(t, types.Typ[types.Int])
}

func isBool(t types.Type) bool {
	return types.Identical(t, types.Typ[types.Bool])
}

func isFloat64(t types.Type) bool {
	return types.Identical(t, types.Typ[types.Float64])
}

func isDuration(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "Duration"
}

// DurationValue represents a parsed duration.
type DurationValue time.Duration
