package compiler

import "go/types"

// directModel gets a concrete model or one admitted by a type parameter.
func (p *packageUnit) directModel(typ types.Type) *model {
	if typ == nil {
		return nil
	}
	typ = types.Unalias(typ)
	if parameter, ok := typ.(*types.TypeParam); ok {
		return p.typeParameterModel(parameter)
	}
	return p.concreteModel(typ)
}

// concreteModel gets a model through pointer layers without reading type sets.
func (p *packageUnit) concreteModel(typ types.Type) *model {
	if typ == nil {
		return nil
	}
	model := p.modelForType(dereference(typ))
	if model != nil && model.requiresConstructor() {
		return model
	}
	return nil
}

type modelMatch struct {
	model *model
	typ   types.Type
}

// typeParameterModel confirms a model against the complete type set.
func (p *packageUnit) typeParameterModel(parameter *types.TypeParam) *model {
	constraint := parameter.Constraint().Underlying().(*types.Interface).Complete()
	matches := p.constraintMatches(
		parameter.Constraint(),
		make(map[types.Type]bool),
	)
	for _, match := range matches {
		if types.Satisfies(match.typ, constraint) {
			return match.model
		}
	}
	return nil
}

// constraintMatches collects model types named by constraint terms.
func (p *packageUnit) constraintMatches(
	typ types.Type,
	seen map[types.Type]bool,
) []modelMatch {
	typ = types.Unalias(typ)
	if seen[typ] {
		return nil
	}
	seen[typ] = true
	if model := p.concreteModel(typ); model != nil {
		return []modelMatch{{model: model, typ: typ}}
	}
	switch typ := typ.(type) {
	case *types.TypeParam:
		return p.constraintMatches(typ.Constraint(), seen)
	case *types.Named:
		if _, ok := typ.Underlying().(*types.Interface); ok {
			return p.constraintMatches(typ.Underlying(), seen)
		}
	case *types.Interface:
		var matches []modelMatch
		for index := 0; index < typ.NumEmbeddeds(); index++ {
			matches = append(
				matches,
				p.constraintMatches(typ.EmbeddedType(index), seen)...,
			)
		}
		return matches
	case *types.Union:
		return p.unionMatches(typ, seen)
	}
	return nil
}

// unionMatches collects protected models named by union terms.
func (p *packageUnit) unionMatches(
	union *types.Union,
	seen map[types.Type]bool,
) []modelMatch {
	var matches []modelMatch
	for index := 0; index < union.Len(); index++ {
		term := union.Term(index)
		matches = append(matches, p.constraintMatches(term.Type(), seen)...)
		if term.Tilde() {
			if model := p.layoutModel(term.Type()); model != nil {
				if modelType := p.localModelType(model); modelType != nil {
					matches = append(matches, modelMatch{model: model, typ: modelType})
				}
			}
		}
	}
	return matches
}

// localModelType gets the Go type emitted for one local model.
func (p *packageUnit) localModelType(model *model) types.Type {
	if p.Models[model.Name] != model {
		return nil
	}
	object := p.typed.Scope().Lookup(model.Name)
	if object == nil {
		return nil
	}
	return object.Type()
}

// layoutModel matches a type whose underlying layout is a protected model.
func (p *packageUnit) layoutModel(typ types.Type) *model {
	for _, source := range p.Sources {
		for _, model := range source.Models {
			if !model.requiresConstructor() {
				continue
			}
			object := p.typed.Scope().Lookup(model.Name)
			if object != nil && types.Identical(typ.Underlying(), object.Type().Underlying()) {
				return model
			}
		}
	}
	return nil
}

// interfaceType reports a concrete interface type, but not a type parameter.
func interfaceType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	typ = types.Unalias(typ)
	if _, parameter := typ.(*types.TypeParam); parameter {
		return false
	}
	_, ok := typ.Underlying().(*types.Interface)
	return ok
}
