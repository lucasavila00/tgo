package compiler

import "go/types"

// enumLayout selects payload storage with the 64-bit Go compiler layout.
func enumLayout(declaration *model, pkg *types.Package) *types.Struct {
	sizes := types.SizesFor("gc", "amd64")
	payloads := make([]types.Type, len(declaration.Variants))
	for index, variant := range declaration.Variants {
		payloads[index] = pkg.Scope().Lookup(declaration.Name + variant.Name).Type()
	}
	for {
		layout := enumStorage(declaration, payloads, pkg)
		if sizes.Sizeof(layout) <= 80 {
			return layout
		}
		largest := largestInlinePayload(declaration, payloads, sizes)
		declaration.Variants[largest].Boxed = true
	}
}

// enumStorage builds the tag, inline fields, and shared box field.
func enumStorage(declaration *model, payloads []types.Type, pkg *types.Package) *types.Struct {
	tag := pkg.Scope().Lookup(declaration.Name + "Tag").Type()
	fields := []*types.Var{types.NewVar(0, pkg, "tgoTag", tag)}
	boxed := false
	for index, variant := range declaration.Variants {
		if len(variant.Fields) == 0 {
			continue
		}
		if variant.Boxed {
			boxed = true
			continue
		}
		fields = append(fields, types.NewVar(0, pkg, "tgo"+variant.Name, payloads[index]))
	}
	if boxed {
		box := types.NewInterfaceType(nil, nil).Complete()
		fields = append(fields, types.NewVar(0, pkg, "tgoPayload", box))
	}
	return types.NewStruct(fields, nil)
}

// largestInlinePayload keeps the first declared variant when sizes are equal.
func largestInlinePayload(declaration *model, payloads []types.Type, sizes types.Sizes) int {
	largest := -1
	for index, variant := range declaration.Variants {
		if len(variant.Fields) == 0 || variant.Boxed {
			continue
		}
		if largest < 0 || sizes.Sizeof(payloads[index]) > sizes.Sizeof(payloads[largest]) {
			largest = index
		}
	}
	return largest
}

// selectEnumLayouts resolves nested value types before containing enums.
func (p *packageUnit) selectEnumLayouts(typ types.Type, seen map[types.Type]bool) {
	typ = types.Unalias(typ)
	if seen[typ] {
		return
	}
	seen[typ] = true
	switch typ := typ.(type) {
	case *types.Named:
		p.selectEnumLayouts(typ.Underlying(), seen)
		if typ.Obj().Pkg() != p.typed {
			return
		}
		declaration := p.Models[typ.Obj().Name()]
		if declaration != nil && declaration.Enum {
			typ.SetUnderlying(enumLayout(declaration, p.typed))
		}
	case *types.Struct:
		for index := 0; index < typ.NumFields(); index++ {
			p.selectEnumLayouts(typ.Field(index).Type(), seen)
		}
	case *types.Array:
		p.selectEnumLayouts(typ.Elem(), seen)
	}
}
