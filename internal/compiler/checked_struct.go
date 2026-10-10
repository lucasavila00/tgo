package compiler

import "go/types"

// checkCheckedStructs checks fields and the required local validation method.
func (p *packageUnit) checkCheckedStructs() {
	for _, source := range p.Sources {
		for _, model := range source.Models {
			if !model.CheckedStruct {
				continue
			}
			object, ok := p.typed.Scope().Lookup(model.Name).(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := types.Unalias(object.Type()).(*types.Named)
			if !ok {
				continue
			}
			p.checkCheckedStructFields(named)
			method := directMethod(named, "check")
			if method == nil {
				p.failAt(object.Pos(),
					"checked struct %s needs check() (%s, error)",
					model.Name, model.Name)
				continue
			}
			if !validCheckedStructMethod(method, named) {
				p.failAt(method.Pos(),
					"checked struct %s check method must have signature check() (%s, error)",
					model.Name, model.Name)
			}
		}
	}
}

func (p *packageUnit) checkCheckedStructFields(named *types.Named) {
	structure, ok := named.Underlying().(*types.Struct)
	if !ok {
		return
	}
	for index := 0; index < structure.NumFields(); index++ {
		field := structure.Field(index)
		if p.validCheckedStructField(field.Type()) {
			continue
		}
		p.failAt(
			field.Pos(),
			"checked struct field %s must be boolean, numeric, string, or a checked struct",
			field.Name(),
		)
	}
}

func (p *packageUnit) validCheckedStructField(typ types.Type) bool {
	unaliased := types.Unalias(typ)
	if basic, ok := unaliased.Underlying().(*types.Basic); ok {
		const allowed = types.IsBoolean | types.IsInteger | types.IsFloat |
			types.IsComplex | types.IsString
		return basic.Info()&allowed != 0
	}
	_, model := p.modelOwner(typ)
	return model != nil && model.CheckedStruct
}

func directMethod(named *types.Named, name string) *types.Func {
	for index := 0; index < named.NumMethods(); index++ {
		method := named.Method(index)
		if method.Name() == name {
			return method
		}
	}
	return nil
}

func validCheckedStructMethod(method *types.Func, named *types.Named) bool {
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil || signature.Params().Len() != 0 ||
		signature.Results().Len() != 2 || signature.Variadic() {
		return false
	}
	errorObject := types.Universe.Lookup("error")
	return errorObject != nil && types.Identical(signature.Recv().Type(), named) &&
		types.Identical(signature.Results().At(0).Type(), named) &&
		types.Identical(signature.Results().At(1).Type(), errorObject.Type())
}
