package tgolint

import (
	"go/ast"
	"go/types"
	"strings"
)

const (
	checkedKind = "checked"
	enumKind    = "enum"
)

type modelFact struct {
	Kind     string
	Name     string
	Variants []string
	Mixed    bool
}

func (*modelFact) AFact() {}

type objectKey struct {
	pkg  *types.Package
	name string
}

func (c *checker) findModels() {
	for _, file := range c.pass.Files {
		if !c.hasGeneratedHeader(file) {
			continue
		}
		c.generated[file] = true
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, item := range general.Specs {
				specification, ok := item.(*ast.TypeSpec)
				if ok {
					c.findModel(specification)
				}
			}
		}
	}
}

func (c *checker) hasGeneratedHeader(file *ast.File) bool {
	if len(file.Comments) == 0 || len(file.Comments[0].List) == 0 {
		return false
	}
	comment := file.Comments[0].List[0]
	position := c.pass.Fset.Position(comment.Pos())
	return position.Line == 1 && position.Column == 1 && comment.Text == generatedHeader
}

func (c *checker) findModel(specification *ast.TypeSpec) {
	structure, ok := specification.Type.(*ast.StructType)
	if !ok {
		return
	}
	object, ok := c.pass.TypesInfo.Defs[specification.Name].(*types.TypeName)
	if !ok {
		return
	}
	fact := emittedModel(specification.Name.Name, structure, object.Type())
	if fact == nil {
		return
	}
	key := objectKey{pkg: object.Pkg(), name: object.Name()}
	c.models[key] = fact
	c.pass.ExportObjectFact(object, fact)
}

func emittedModel(name string, structure *ast.StructType, typ types.Type) *modelFact {
	fields := structure.Fields.List
	underlying, ok := typ.Underlying().(*types.Struct)
	if !ok || underlying.NumFields() != len(fields) {
		return nil
	}
	if len(fields) == 1 && fieldName(fields[0]) == "value" &&
		validCheckedAPI(typ, underlying.Field(0).Type(), "New"+name) {
		return &modelFact{Kind: checkedKind, Name: name}
	}
	if len(fields) < 2 || fieldName(fields[0]) != "tgoTag" ||
		!validTagMethod(typ, underlying.Field(0).Type()) {
		return nil
	}
	variants := make([]string, 0, len(fields)-1)
	for index, field := range fields[1:] {
		field := fieldName(field)
		if !strings.HasPrefix(field, "tgo") || len(field) == len("tgo") {
			return nil
		}
		variant := strings.TrimPrefix(field, "tgo")
		payload := underlying.Field(index + 1).Type()
		if !validEnumAPI(typ, payload, "Tgo"+variant, "New"+name+variant) {
			return nil
		}
		variants = append(variants, variant)
	}
	return &modelFact{Kind: enumKind, Name: name, Variants: variants}
}

func fieldName(field *ast.Field) string {
	if len(field.Names) != 1 {
		return ""
	}
	return field.Names[0].Name
}

func method(typ types.Type, name string) *types.Signature {
	object, _, _ := types.LookupFieldOrMethod(typ, true, nil, name)
	function, ok := object.(*types.Func)
	if !ok {
		return nil
	}
	signature, _ := function.Type().(*types.Signature)
	return signature
}

func constructor(typ types.Type, name string) *types.Signature {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return nil
	}
	function, ok := named.Obj().Pkg().Scope().Lookup(name).(*types.Func)
	if !ok {
		return nil
	}
	signature, _ := function.Type().(*types.Signature)
	return signature
}

func validCheckedAPI(typ, base types.Type, constructorName string) bool {
	value := method(typ, "Value")
	makeValue := constructor(typ, constructorName)
	if value == nil || makeValue == nil {
		return false
	}
	errorType := types.Universe.Lookup("error").Type()
	return value.Params().Len() == 0 && value.Results().Len() == 1 &&
		types.Identical(value.Results().At(0).Type(), base) &&
		makeValue.Params().Len() == 1 && makeValue.Results().Len() == 2 &&
		types.Identical(makeValue.Params().At(0).Type(), base) &&
		types.Identical(makeValue.Results().At(0).Type(), typ) &&
		types.Identical(makeValue.Results().At(1).Type(), errorType)
}

func validTagMethod(typ, tag types.Type) bool {
	method := method(typ, "TgoTag")
	return method != nil && method.Params().Len() == 0 && method.Results().Len() == 1 &&
		types.Identical(method.Results().At(0).Type(), tag)
}

func validEnumAPI(typ, payload types.Type, accessor, constructorName string) bool {
	read := method(typ, accessor)
	makeValue := constructor(typ, constructorName)
	return read != nil && makeValue != nil &&
		read.Params().Len() == 0 && read.Results().Len() == 1 &&
		types.Identical(read.Results().At(0).Type(), payload) &&
		makeValue.Params().Len() == 1 && makeValue.Results().Len() == 1 &&
		types.Identical(makeValue.Params().At(0).Type(), payload) &&
		types.Identical(makeValue.Results().At(0).Type(), typ)
}

func (c *checker) modelFor(typ types.Type) *modelFact {
	typ = types.Unalias(typ)
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return nil
	}
	key := objectKey{pkg: named.Obj().Pkg(), name: named.Obj().Name()}
	if fact, ok := c.models[key]; ok {
		return fact
	}
	fact := new(modelFact)
	if !c.pass.ImportObjectFact(named.Obj(), fact) {
		c.models[key] = nil
		return nil
	}
	c.models[key] = fact
	return fact
}

func (c *checker) modelForReceiver(typ types.Type) *modelFact {
	typ = types.Unalias(typ)
	for {
		if parameter, ok := typ.(*types.TypeParam); ok {
			return c.modelForTypeParameter(parameter)
		}
		pointer, ok := typ.(*types.Pointer)
		if !ok {
			return c.modelFor(typ)
		}
		typ = types.Unalias(pointer.Elem())
	}
}

func (c *checker) modelForTypeParameter(parameter *types.TypeParam) *modelFact {
	terms, supported := simpleTerms(parameter.Constraint())
	if !supported {
		return nil
	}
	var found *modelFact
	for _, term := range terms {
		model := c.modelForReceiver(term.Type())
		if model == nil {
			continue
		}
		if found != nil && (found.Kind != model.Kind || found.Name != model.Name) {
			return &modelFact{Mixed: true}
		}
		found = model
	}
	return found
}

func (c *checker) modelForSelector(selector *ast.SelectorExpr) *modelFact {
	if model := c.modelForReceiver(c.pass.TypesInfo.TypeOf(selector.X)); model != nil {
		return model
	}
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil {
		return nil
	}
	if function, ok := selection.Obj().(*types.Func); ok {
		signature, _ := function.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			return c.modelForReceiver(signature.Recv().Type())
		}
	}
	current := selection.Recv()
	for offset, index := range selection.Index() {
		current = dereference(current)
		structure, ok := current.Underlying().(*types.Struct)
		if !ok || index >= structure.NumFields() {
			return nil
		}
		if offset == len(selection.Index())-1 {
			return c.modelForReceiver(current)
		}
		current = structure.Field(index).Type()
	}
	return nil
}

func dereference(typ types.Type) types.Type {
	typ = types.Unalias(typ)
	for {
		pointer, ok := typ.(*types.Pointer)
		if !ok {
			return typ
		}
		typ = types.Unalias(pointer.Elem())
	}
}
