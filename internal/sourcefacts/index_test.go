package sourcefacts

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"tgo/pkg/syntax"
)

type objectFact struct {
	position token.Pos
	name     string
	object   types.Object
}

type projectionStub struct {
	definitions []objectFact
	uses        []objectFact
}

func (*projectionStub) RangeTypes(
	func(token.Pos, token.Pos, types.TypeAndValue, bool),
) {
}

func (p *projectionStub) RangeDefinitions(
	yield func(token.Pos, string, types.Object, bool),
) {
	for _, fact := range p.definitions {
		yield(fact.position, fact.name, fact.object, false)
	}
}

func (p *projectionStub) RangeUses(
	yield func(token.Pos, types.Object, bool),
) {
	for _, fact := range p.uses {
		yield(fact.position, fact.object, false)
	}
}

func (*projectionStub) RangeSelections(
	func(token.Pos, token.Pos, *types.Selection, bool),
) {
}

func (*projectionStub) RangeInstances(
	func(token.Pos, types.Instance, bool),
) {
}

func (*projectionStub) RangeImplicits(
	func(token.Pos, token.Pos, types.Object, bool),
) {
}

func TestProjectionFindsShiftedDefinition(t *testing.T) {
	t.Parallel()
	projected := []byte("package sample\ntype value struct {\n      field int\n}\n")
	source := []byte("package sample\ntype value struct {\n\tfield int\n}\n")
	files := token.NewFileSet()
	projectedFile := files.AddFile("sample.tgo", -1, len(projected))
	projectedFile.SetLinesForContent(projected)
	position := projectedFile.Pos(strings.Index(string(projected), "field"))
	object := types.NewVar(position, nil, "field", types.Typ[types.Int])
	parsed := parseSource(t, files, source)
	index := NewProjection(parsed, &projectionStub{definitions: []objectFact{{
		position: position, name: "field", object: object,
	}}}, files)
	declaration := syntax.GeneralDeclarationOf(parsed.Declarations[0])
	specification := syntax.TypeSpecificationOf(declaration.Specs[0])
	structure := syntax.StructTypeExpressionOf(specification.Type)
	name := structure.Fields.List[0].Names[0]
	if got := index.DefinitionName(name); got != object {
		t.Fatalf("shifted field definition = %v, want %v", got, object)
	}
}

func TestProjectionDoesNotClassifySameLineUseAsDefinition(t *testing.T) {
	t.Parallel()
	projected := []byte("package sample\nvar      value = 1; var copy = value\n")
	source := []byte("package sample\nvar value = 1; var copy = value\n")
	files := token.NewFileSet()
	projectedFile := files.AddFile("sample.tgo", -1, len(projected))
	projectedFile.SetLinesForContent(projected)
	definitionPosition := projectedFile.Pos(strings.Index(string(projected), "value"))
	usePosition := projectedFile.Pos(strings.LastIndex(string(projected), "value"))
	object := types.NewVar(definitionPosition, nil, "value", types.Typ[types.Int])
	parsed := parseSource(t, files, source)
	index := NewProjection(parsed, &projectionStub{
		definitions: []objectFact{{
			position: definitionPosition, name: "value", object: object,
		}},
		uses: []objectFact{{position: usePosition, object: object}},
	}, files)
	definition, use := sourceIdentifiers(t, parsed)
	definitionObject, definitionFact := index.IdentifierFact(parsed, definition)
	useObject, useFact := index.IdentifierFact(parsed, use)
	if definitionObject != object || useObject != object {
		t.Fatalf("objects = %v, %v, want %v", definitionObject, useObject, object)
	}
	if !definitionFact {
		t.Fatal("declaration is not classified as a definition")
	}
	if useFact {
		t.Fatal("same-line use is classified as a definition")
	}
}

func parseSource(t *testing.T, files *token.FileSet, source []byte) *syntax.File {
	t.Helper()
	parsed, err := syntax.ParseGoFile(
		files, "sample.tgo", source, syntax.AllErrors,
	)
	if err != nil {
		t.Fatalf("parse source syntax: %v", err)
	}
	if parsed == nil {
		t.Fatal("parse source syntax returned nil")
	}
	return parsed
}

func sourceIdentifiers(
	t *testing.T,
	file *syntax.File,
) (*syntax.Node, *syntax.Node) {
	t.Helper()
	first := syntax.GeneralDeclarationOf(file.Declarations[0])
	firstValue := syntax.ValueSpecificationOf(first.Specs[0])
	definition := identifierNode(file, firstValue.Names[0])
	second := syntax.GeneralDeclarationOf(file.Declarations[1])
	secondValue := syntax.ValueSpecificationOf(second.Specs[0])
	useName := syntax.IdentifierExpressionOf(secondValue.Values[0])
	use := identifierNode(file, useName)
	if definition == nil || use == nil {
		t.Fatal("identifier node is absent")
	}
	return definition, use
}

func identifierNode(file *syntax.File, identifier *syntax.Identifier) *syntax.Node {
	var result *syntax.Node
	syntax.Inspect(file, func(node *syntax.Node) bool {
		value, ok := syntax.IdentifierOf(node)
		if ok && value.Start == identifier.Start && value.Stop == identifier.Stop {
			result = node
			return false
		}
		return result == nil
	})
	return result
}
