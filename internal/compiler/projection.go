package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// ProjectionFacts provides typed Go facts without exposing Go syntax nodes.
// The final callback value reports whether a fact is in a generated function.
type ProjectionFacts struct {
	info      *types.Info
	synthetic map[ast.Node]bool
}

func newProjectionFacts(unit *packageUnit) *ProjectionFacts {
	synthetic := make(map[ast.Node]bool)
	for declaration := range unit.generated {
		if _, ok := declaration.(*ast.FuncDecl); !ok {
			continue
		}
		ast.Inspect(declaration, func(node ast.Node) bool {
			if node != nil {
				synthetic[node] = true
			}
			return true
		})
	}
	return &ProjectionFacts{info: unit.info, synthetic: synthetic}
}

// RangeTypes visits expression type facts in the Go projection.
func (p *ProjectionFacts) RangeTypes(
	yield func(token.Pos, token.Pos, types.TypeAndValue, bool),
) {
	for expression, value := range p.info.Types {
		yield(expression.Pos(), expression.End(), value, p.synthetic[expression])
	}
}

// RangeDefinitions visits definition facts in the Go projection.
func (p *ProjectionFacts) RangeDefinitions(
	yield func(token.Pos, string, types.Object, bool),
) {
	for identifier, object := range p.info.Defs {
		yield(identifier.Pos(), identifier.Name, object, p.synthetic[identifier])
	}
}

// RangeUses visits use facts in the Go projection.
func (p *ProjectionFacts) RangeUses(
	yield func(token.Pos, string, types.Object, bool),
) {
	for identifier, object := range p.info.Uses {
		yield(identifier.Pos(), identifier.Name, object, p.synthetic[identifier])
	}
}

// RangeSelections visits selector facts in the Go projection.
func (p *ProjectionFacts) RangeSelections(
	yield func(token.Pos, token.Pos, *types.Selection, bool),
) {
	for expression, selection := range p.info.Selections {
		yield(expression.Pos(), expression.End(), selection, p.synthetic[expression])
	}
}

// RangeInstances visits generic instance facts in the Go projection.
func (p *ProjectionFacts) RangeInstances(
	yield func(token.Pos, types.Instance, bool),
) {
	for identifier, instance := range p.info.Instances {
		yield(identifier.Pos(), instance, p.synthetic[identifier])
	}
}

// RangeImplicits visits implicit object facts in the Go projection.
func (p *ProjectionFacts) RangeImplicits(
	yield func(token.Pos, token.Pos, types.Object, bool),
) {
	for node, object := range p.info.Implicits {
		yield(node.Pos(), node.End(), object, p.synthetic[node])
	}
}

func projectionReferences(unit *packageUnit) map[token.Pos]types.Object {
	references := make(map[token.Pos]types.Object, len(unit.sourceReferences))
	for position, object := range unit.sourceReferences {
		references[position] = object
	}
	for _, reference := range unit.references {
		if object := unit.info.Uses[reference.Name]; object != nil {
			references[reference.At] = object
		}
	}
	return references
}
