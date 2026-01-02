package ir

import (
	"go/types"
	"slices"
	"time"

	"github.com/asp24/gendi/internal/typeutil"
)

// Container is the fully resolved intermediate representation of a DI container.
type Container struct {
	Services   map[string]*Service
	Parameters map[string]*Parameter
	Tags       map[string]*Tag

	// Computed fields
	ServiceOrder []string // Topologically sorted service IDs
}

// ParamGetters returns parameter getter methods needed by the container.
func (c *Container) ParamGetters() map[string]string {
	getters := make(map[string]string)
	for name, param := range c.Parameters {
		method := param.GetterMethod()
		if method != "" {
			getters[name] = method
		}
	}
	for _, svc := range c.Services {
		if svc.Constructor == nil {
			continue
		}
		for _, arg := range svc.Constructor.Args {
			if arg.Kind != ParamRefArg || arg.Parameter == nil {
				continue
			}
			method := arg.Parameter.GetterMethod()
			if method != "" {
				getters[arg.Parameter.Name] = method
			}
		}
	}
	return getters
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

func (s *Service) Clone() *Service {
	result := *s
	if s.Constructor != nil {
		result.Constructor = s.Constructor.Clone()
	}
	result.Tags = slices.Clone(s.Tags)
	result.Dependencies = slices.Clone(s.Dependencies)

	return &result
}

// Constructor defines how a service is constructed.
type Constructor struct {
	Kind ConstructorKind
	Func *types.Func // For FuncConstructor
	Args []*Argument

	// For method constructors
	Receiver *Service // The service whose method is called

	// For generic constructors
	TypeArgs []types.Type // Resolved type arguments for generic functions

	// Signature info
	Params       []types.Type
	ResultType   types.Type
	ReturnsError bool
	Variadic     bool // True if function/method has variadic parameters
}

func (c *Constructor) Clone() *Constructor {
	result := *c

	if len(c.Args) > 0 {
		result.Args = make([]*Argument, len(c.Args))
		for i, arg := range c.Args {
			if arg == nil {
				continue
			}

			argClone := *arg
			result.Args[i] = &argClone
		}
	}

	result.Params = slices.Clone(c.Params)
	result.TypeArgs = slices.Clone(c.TypeArgs)

	return &result
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
	Service   *Service     // For ServiceRef
	Parameter *Parameter   // For ParamRef
	Tag       *Tag         // For Tagged
	Literal   LiteralValue // For Literal
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
	Public      bool
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
	case typeutil.IsString(p.Type):
		return "GetString"
	case typeutil.IsInt(p.Type):
		return "GetInt"
	case typeutil.IsBool(p.Type):
		return "GetBool"
	case typeutil.IsFloat64(p.Type):
		return "GetFloat"
	case typeutil.IsDuration(p.Type):
		return "GetDuration"
	default:
		return ""
	}
}

// DurationValue represents a parsed duration.
type DurationValue time.Duration
