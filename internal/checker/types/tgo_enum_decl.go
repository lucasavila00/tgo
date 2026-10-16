package types

import (
	"go/constant"
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/ast"
	. "github.com/lucasavila00/tgo/internal/checker/upstream/errors"
)

// collectEnumObjects declares the generated Go API before package expressions are checked.
func (check *Checker) collectEnumObjects(spec *ast.TypeSpec, object *TypeName, scope *Scope) {
	declaration, ok := spec.Type.(*ast.EnumType)
	if !ok {
		return
	}
	if check.tgo.enums == nil {
		check.tgo.enums = make(map[*ast.EnumType]*Enum)
	}
	enum := &Enum{object: object, declaration: declaration}
	check.tgo.enums[declaration] = enum
	if len(declaration.Variants) == 0 {
		check.error(declaration, InvalidSyntaxTree, "enum requires at least one variant")
		return
	}
	check.declareEnumTag(enum)
	seen := make(map[string]bool)
	for index, declaration := range declaration.Variants {
		if seen[declaration.Name.Name] || declaration.Name.Name == "_" {
			check.errorf(declaration.Name, DuplicateDecl, "invalid or duplicate enum variant %s", declaration.Name.Name)
			continue
		}
		seen[declaration.Name.Name] = true
		variant := &EnumVariant{enum: enum, name: declaration.Name.Name, index: index, declaration: declaration}
		enum.variants = append(enum.variants, variant)
		check.declareEnumPayload(variant, scope)
		check.declareEnumConstructor(variant, scope)
		tagName := enum.object.Name() + "Tag" + variant.name
		tag := NewConst(declaration.Pos(), check.pkg, tagName, enum.tag, constant.MakeInt64(int64(index)))
		check.declare(check.pkg.scope, nil, tag, nopos)
	}
}

// declareEnumTag creates the integer type shared by the tag method and constants.
func (check *Checker) declareEnumTag(enum *Enum) {
	name := enum.object.Name() + "Tag"
	object := NewTypeName(enum.object.Pos(), check.pkg, name, nil)
	enum.tag = NewNamed(object, Typ[enumTagKind(len(enum.declaration.Variants))], nil)
	check.declare(check.pkg.scope, nil, object, nopos)
}

// declareEnumPayload registers a real Go payload type using the variant's field syntax.
func (check *Checker) declareEnumPayload(variant *EnumVariant, scope *Scope) {
	name := variant.enum.object.Name() + variant.name
	identifier := &ast.Ident{Name: name, NamePos: variant.declaration.Pos()}
	structure := variant.declaration.Type
	if structure == nil {
		structure = &ast.StructType{Struct: variant.declaration.Pos(),
			Fields: &ast.FieldList{Opening: variant.declaration.End(), Closing: variant.declaration.End()}}
	}
	variant.object = NewTypeName(identifier.Pos(), check.pkg, name, nil)
	spec := &ast.TypeSpec{Name: identifier, Type: structure}
	check.declarePkgObj(identifier, variant.object, &declInfo{file: scope, version: check.version, tdecl: spec})
	check.recordDef(variant.declaration.Name, variant.object)
}

// declareEnumConstructor gives ordinary Go callers a checked function with one argument per field.
func (check *Checker) declareEnumConstructor(variant *EnumVariant, scope *Scope) {
	position := variant.declaration.Pos()
	identifier := &ast.Ident{Name: "New" + variant.enum.object.Name() + variant.name, NamePos: position}
	parameters := &ast.FieldList{Opening: position, Closing: position}
	if structure := variant.declaration.Type; structure != nil {
		for _, field := range structure.Fields.List {
			for range max(1, len(field.Names)) {
				name := "tgoField" + strconv.Itoa(len(parameters.List))
				parameters.List = append(parameters.List, &ast.Field{
					Names: []*ast.Ident{{Name: name, NamePos: field.Pos()}}, Type: field.Type})
			}
		}
	}
	result := &ast.Ident{Name: variant.enum.object.Name(), NamePos: position}
	signature := &ast.FuncType{Func: position, Params: parameters,
		Results: &ast.FieldList{Opening: position, List: []*ast.Field{{Type: result}}, Closing: position}}
	declaration := &ast.FuncDecl{Name: identifier, Type: signature}
	object := NewFunc(position, check.pkg, identifier.Name, nil)
	check.declarePkgObj(identifier, object, &declInfo{file: scope, version: check.version, fdecl: declaration})
}

// enumType checks payload types and connects their identities to the parent enum.
func (check *Checker) enumType(declaration *ast.EnumType, definition *TypeName) Type {
	enum := check.tgo.enums[declaration]
	if enum == nil || definition == nil {
		check.error(declaration, InvalidSyntaxTree, "enum requires a named package declaration")
		return Typ[Invalid]
	}
	named, ok := definition.typ.(*Named)
	if !ok || len(enum.variants) == 0 {
		check.error(declaration, InvalidSyntaxTree, "enum requires a non-empty type definition")
		return Typ[Invalid]
	}
	enum.named, named.enum = named, enum
	for _, variant := range enum.variants {
		check.objDecl(variant.object)
		variant.typ, ok = variant.object.Type().(*Named)
		if !ok || !isValid(variant.typ.Underlying()) {
			return Typ[Invalid]
		}
	}
	check.declareEnumMethods(enum)
	return enumStorage(enum)
}

// declareEnumMethods exposes the tag and copied payloads through ordinary Go methods.
func (check *Checker) declareEnumMethods(enum *Enum) {
	check.addEnumMethod(enum, "Tag", enum.tag)
	for _, variant := range enum.variants {
		check.addEnumMethod(enum, variant.name+"Payload", variant.typ)
	}
}

// addEnumMethod attaches one generated value-receiver method to its declared enum.
func (check *Checker) addEnumMethod(enum *Enum, name string, result Type) {
	position := enum.object.Pos()
	receiver := NewVar(position, check.pkg, "", enum.named)
	results := NewTuple(NewVar(position, check.pkg, "", result))
	signature := NewSignatureType(receiver, nil, nil, nil, results, false)
	method := NewFunc(position, check.pkg, name, signature)
	for _, existing := range enum.named.methods {
		if existing.Name() == name {
			check.errorf(atPos(existing.Pos()), DuplicateDecl, "method %s conflicts with generated enum API", name)
			return
		}
	}
	enum.named.AddMethod(method)
}

// checkEnums resolves source declarations before ordinary initializers use the generated API.
func (check *Checker) checkEnums() {
	for _, file := range check.files {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, specification := range group.Specs {
				spec := specification.(*ast.TypeSpec)
				if source, ok := spec.Type.(*ast.EnumType); ok {
					check.objDecl(check.tgo.enums[source].object)
				}
			}
		}
	}
}
