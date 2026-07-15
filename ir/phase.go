package ir

import (
	di "github.com/gendi-org/gendi"
)

type Phase interface {
	Apply(cfg *di.Config, container *Container) error
}
