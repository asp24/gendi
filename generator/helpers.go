package generator

import (
	"go/types"
)

func isNilable(t types.Type) bool {
	switch tt := t.(type) {
	case *types.Pointer, *types.Interface, *types.Slice, *types.Map, *types.Chan, *types.Signature:
		return true
	case *types.Named:
		return isNilable(tt.Underlying())
	case *types.Alias:
		return isNilable(tt.Underlying())
	default:
		return false
	}
}
