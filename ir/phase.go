package ir

import (
	di "github.com/asp24/gendi"
)

type Phase interface {
	Apply(cfg *di.Config, container *Container) error
}
