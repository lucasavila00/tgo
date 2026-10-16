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

// declareEnumConstructor reserves the public function before its payload is checked.
func (check *Checker) declareEnumConstructor(variant *EnumVariant, _ *Scope) {
	name := "New" + variant.enum.object.Name() + variant.name
	object := NewFunc(variant.declaration.Pos(), check.pkg, name, nil)
	variant.constructor = object
	if check.tgo.enumConstructors == nil {
		check.tgo.enumConstructors = make(map[Object]*EnumVariant)
	}
	check.tgo.enumConstructors[object] = variant
	check.declare(check.pkg.scope, nil, object, nopos)
}

// enumConstructorType builds its signature from checked fields without checking source syntax twice.
func (check *Checker) enumConstructorType(object Object) bool {
	variant := check.tgo.enumConstructors[object]
	if variant == nil {
		return false
	}
	if object.Type() != nil {
		return true
	}
	check.objDecl(variant.enum.object)
	check.objDecl(variant.object)
	payload, ok := variant.object.Type().Underlying().(*Struct)
	if !ok {
		object.setType(NewSignatureType(nil, nil, nil, nil, nil, false))
		return true
	}
	parameters := make([]*Var, payload.NumFields())
	for index, field := range payload.fields {
		parameters[index] = NewVar(field.Pos(), check.pkg, "tgoField"+strconv.Itoa(index), field.Type())
	}
	result := NewTuple(NewVar(object.Pos(), check.pkg, "", variant.enum.object.Type()))
	object.setType(NewSignatureType(nil, nil, nil, NewTuple(parameters...), result, false))
	return true
}

// enumType checks payload types and connects their identities to the parent enum.
func (check *Checker) enumType(declaration *ast.EnumType, definition *TypeName) Type {
	enum := check.tgo.enums[declaration]
	if enum == nil || definition == nil {
		check.error(declaration, InvalidSyntaxTree, "enum declarations must be at package level")
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

// checkEnums resolves source declarations and public functions before ordinary initializers.
func (check *Checker) checkEnums() {
	enums := check.sourceEnums()
	for _, enum := range enums {
		check.objDecl(enum.object)
	}
	for _, enum := range enums {
		if enum.named == nil || !isValid(enum.named.Underlying()) {
			continue
		}
		check.enumSizes(enum.named, enum.declaration)
		for _, variant := range enum.variants {
			check.enumConstructorType(variant.constructor)
		}
	}
}

// sourceEnums returns package enum declarations in source order.
func (check *Checker) sourceEnums() []*Enum {
	var enums []*Enum
	for _, file := range check.files {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, specification := range group.Specs {
				spec := specification.(*ast.TypeSpec)
				if source, ok := spec.Type.(*ast.EnumType); ok {
					enums = append(enums, check.tgo.enums[source])
				}
			}
		}
	}
	return enums
}
