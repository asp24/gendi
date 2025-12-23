package generator

import (
	"fmt"
	"go/types"
	"strings"

	di "github.com/asp24/gendi"
)

func resolveConstructor(id string, svc *di.Service, loader *typeLoader, services map[string]*serviceDef, resolve func(string) error) (constructorDef, error) {
	cons := svc.Constructor
	if cons.Func == "" && cons.Method == "" {
		return constructorDef{}, fmt.Errorf("service %q missing constructor", id)
	}
	if cons.Func != "" && cons.Method != "" {
		return constructorDef{}, fmt.Errorf("service %q has both func and method constructors", id)
	}

	if cons.Func != "" {
		pkgPath, name, err := splitPkgSymbol(cons.Func)
		if err != nil {
			return constructorDef{}, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
		obj, err := loader.lookupFunc(pkgPath, name)
		if err != nil {
			return constructorDef{}, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
		sig := obj.Type().(*types.Signature)
		resType, returnsErr, err := validateConstructorSignature(sig)
		if err != nil {
			return constructorDef{}, fmt.Errorf("service %q constructor.func: %w", id, err)
		}
		params := signatureParams(sig)
		return constructorDef{
			kind:         "func",
			funcObj:      obj,
			params:       params,
			result:       resType,
			returnsError: returnsErr,
			argDefs:      cons.Args,
		}, nil
	}

	// method
	methodRef := cons.Method
	if !strings.HasPrefix(methodRef, "@") {
		return constructorDef{}, fmt.Errorf("service %q constructor.method must start with @", id)
	}
	methodRef = methodRef[1:]
	parts := strings.Split(methodRef, ".")
	if len(parts) < 2 {
		return constructorDef{}, fmt.Errorf("service %q constructor.method invalid format", id)
	}
	methodName := parts[len(parts)-1]
	recvID := strings.Join(parts[:len(parts)-1], ".")
	if recvID == "" || methodName == "" {
		return constructorDef{}, fmt.Errorf("service %q constructor.method invalid format", id)
	}
	if err := resolve(recvID); err != nil {
		return constructorDef{}, err
	}
	recvSvc := services[recvID]
	if recvSvc == nil {
		return constructorDef{}, fmt.Errorf("service %q constructor.method unknown receiver service %q", id, recvID)
	}
	meth, err := loader.lookupMethod(recvSvc.typeName, methodName)
	if err != nil {
		return constructorDef{}, fmt.Errorf("service %q constructor.method: %w", id, err)
	}
	msig := meth.Type().(*types.Signature)
	resType, returnsErr, err := validateConstructorSignature(msig)
	if err != nil {
		return constructorDef{}, fmt.Errorf("service %q constructor.method: %w", id, err)
	}
	params := signatureParams(msig)
	return constructorDef{
		kind:         "method",
		methodObj:    meth,
		methodRecvID: recvID,
		params:       params,
		result:       resType,
		returnsError: returnsErr,
		argDefs:      cons.Args,
	}, nil
}

func signatureParams(sig *types.Signature) []types.Type {
	params := []types.Type{}
	for i := 0; i < sig.Params().Len(); i++ {
		params = append(params, sig.Params().At(i).Type())
	}
	return params
}
