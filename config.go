package di

// Pass is a compiler pass that can mutate config before validation and generation.
type Pass interface {
	Name() string
	Process(cfg *Config) error
}

// Config is the root configuration for the DI container.
// This is a resolved configuration with no import directives.
type Config struct {
	Parameters map[string]Parameter
	Tags       map[string]Tag
	Services   map[string]*Service
}

// Parameter defines a typed parameter literal.
type Parameter struct {
	Type  string
	Value Literal
}

// Tag defines a tag declaration.
type Tag struct {
	ElementType string
	SortBy      string
}

// ServiceTag defines a tag assigned to a service.
type ServiceTag struct {
	Name       string
	Attributes map[string]interface{}
}

// Service defines a service entry.
type Service struct {
	Type               string
	Constructor        Constructor
	Shared             *bool
	Public             bool
	Decorates          string
	DecorationPriority int
	Tags               []ServiceTag
	Alias              string
}

// Constructor defines service constructor configuration.
type Constructor struct {
	Func   string
	Method string
	Args   []Argument
}

// ArgumentKind is the parsed kind of a constructor argument.
type ArgumentKind int

const (
	ArgLiteral ArgumentKind = iota
	ArgServiceRef
	ArgInner
	ArgParam
	ArgTagged
)

// Argument represents a constructor argument.
type Argument struct {
	Kind    ArgumentKind
	Value   string
	Literal Literal
}
