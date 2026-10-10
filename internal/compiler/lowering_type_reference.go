package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
)

// plannedTypeReference keeps the source objects needed to spell one type.
type plannedTypeReference struct {
	typ         types.Type
	sourceScope *types.Scope
	use         token.Pos
	objects     []plannedTypeObjectReference
	anchor      plannedTypeDeclarationAnchor
	blockers    []plannedTypeBlocker
	direct      bool
}

type plannedTypeBlocker struct {
	intended    types.Object
	object      types.Object
	definition  *ast.Ident
	identifiers []*ast.Ident
	typeName    bool
	alias       *plannedTypeAliasAction
}

type plannedTypeAliasAction struct {
	object       types.Object
	before       *ast.Ident
	packageScope bool
	parameters   []plannedTypeAliasParameter
}

type plannedTypeAliasParameter struct {
	name       string
	constraint types.Type
}

type plannedTypeObjectReference struct {
	object     types.Object
	definition *ast.Ident
	direct     bool
}

type plannedTypeObjectCapture struct {
	reference   *plannedTypeReference
	current     *types.Package
	info        *types.Info
	file        *ast.File
	body        *ast.BlockStmt
	definitions map[types.Object]*ast.Ident
	seenObjects map[types.Object]int
	seenTypes   map[types.Type]bool
}

// plannedTypeDeclarationAnchor gives a scope where a type alias is legal.
type plannedTypeDeclarationAnchor struct {
	owner *types.Scope
	after *ast.Ident
	entry *ast.BlockStmt
}

func capturePlannedTypeReference(
	typ types.Type,
	sourceScope *types.Scope,
	use token.Pos,
	current *types.Package,
	info *types.Info,
	file *ast.File,
	body *ast.BlockStmt,
) plannedTypeReference {
	reference := plannedTypeReference{
		typ: typ, sourceScope: sourceScope, use: use,
	}
	definitions := typeObjectDefinitions(info)
	capturePlannedTypeObjects(&reference, current, info, file, body, definitions)
	capturePlannedTypeBlockers(&reference, current, info)
	if reference.anchor.after != nil || reference.anchor.entry != nil {
		reference.direct = true
	}
	if reference.anchor.owner == nil {
		reference.anchor.owner = sourceScope
	}
	return reference
}

func typeObjectDefinitions(info *types.Info) map[types.Object]*ast.Ident {
	definitions := make(map[types.Object]*ast.Ident)
	for identifier, object := range info.Defs {
		if object != nil {
			definitions[object] = identifier
		}
	}
	return definitions
}

func capturePlannedTypeObjects(
	reference *plannedTypeReference,
	current *types.Package,
	info *types.Info,
	file *ast.File,
	body *ast.BlockStmt,
	definitions map[types.Object]*ast.Ident,
) {
	capture := plannedTypeObjectCapture{
		reference: reference, current: current, info: info, file: file, body: body,
		definitions: definitions,
		seenObjects: make(map[types.Object]int),
		seenTypes:   make(map[types.Type]bool),
	}
	capture.visit(reference.typ)
}

func (capture *plannedTypeObjectCapture) addObject(object types.Object, parameter, direct bool) {
	if object == nil {
		return
	}
	if index, exists := capture.seenObjects[object]; exists {
		capture.reference.objects[index].direct = capture.reference.objects[index].direct || direct
		return
	}
	definition := capture.definitions[object]
	capture.seenObjects[object] = len(capture.reference.objects)
	capture.reference.objects = append(capture.reference.objects, plannedTypeObjectReference{
		object: object, definition: definition, direct: direct,
	})
	if object.Pkg() != nil && object.Pkg() != capture.current {
		packageObject := filePackageObject(capture.file, capture.info, object.Pkg())
		capture.addObject(packageObject, false, packageObject != nil &&
			packageObject.Name() != "." && packageObject.Name() != "_")
	}
	if object.Pkg() != capture.current || object.Parent() == capture.current.Scope() {
		return
	}
	candidate := plannedTypeDeclarationAnchor{owner: object.Parent()}
	if parameter {
		candidate.entry = capture.body
	} else if _, ok := object.(*types.TypeName); ok {
		candidate.after = definition
	} else {
		return
	}
	if deeperTypeScope(candidate.owner, capture.reference.anchor.owner) ||
		candidate.owner == capture.reference.anchor.owner &&
			candidate.after != nil &&
			(capture.reference.anchor.after == nil ||
				candidate.after.Pos() > capture.reference.anchor.after.Pos()) {
		capture.reference.anchor = candidate
	}
}

func (capture *plannedTypeObjectCapture) visit(item types.Type) {
	if item == nil || capture.seenTypes[item] {
		return
	}
	capture.seenTypes[item] = true
	switch item := item.(type) {
	case *types.Alias:
		capture.addObject(item.Obj(), false, capture.typeObjectIsDirect(item.Obj()))
		capture.visit(item.Rhs())
		visitTypeArguments(item.TypeArgs(), capture.visit)
	case *types.Named:
		capture.addObject(item.Obj(), false, capture.typeObjectIsDirect(item.Obj()))
		visitTypeArguments(item.TypeArgs(), capture.visit)
	case *types.TypeParam:
		capture.addObject(item.Obj(), true, true)
	case *types.Basic:
		if item.Kind() == types.UnsafePointer {
			object := types.Unsafe.Scope().Lookup("Pointer")
			capture.addObject(object, false, capture.typeObjectIsDirect(object))
		} else {
			capture.addObject(types.Universe.Lookup(item.Name()), false, true)
		}
	case *types.Pointer:
		capture.visit(item.Elem())
	case *types.Slice:
		capture.visit(item.Elem())
	case *types.Array:
		capture.visit(item.Elem())
	case *types.Map:
		capture.visit(item.Key())
		capture.visit(item.Elem())
	case *types.Chan:
		capture.visit(item.Elem())
	case *types.Struct:
		for index := range item.NumFields() {
			capture.visit(item.Field(index).Type())
		}
	case *types.Interface:
		for index := range item.NumExplicitMethods() {
			capture.visit(item.ExplicitMethod(index).Type())
		}
		for index := range item.NumEmbeddeds() {
			capture.visit(item.EmbeddedType(index))
		}
	case *types.Union:
		for index := range item.Len() {
			capture.visit(item.Term(index).Type())
		}
	case *types.Signature:
		visitTypeParameters(item.TypeParams(), capture.visit)
		visitTupleTypes(item.Params(), capture.visit)
		visitTupleTypes(item.Results(), capture.visit)
	case *types.Tuple:
		visitTupleTypes(item, capture.visit)
	}
}

func (capture *plannedTypeObjectCapture) typeObjectIsDirect(object types.Object) bool {
	return typeObjectIsDirect(object, capture.current, capture.file, capture.info)
}

func capturePlannedTypeBlockers(
	reference *plannedTypeReference,
	current *types.Package,
	info *types.Info,
) {
	seenBlockers := make(map[types.Object]bool)
	for _, required := range reference.objects {
		object := required.object
		if object == nil || !required.direct || reference.sourceScope == nil {
			continue
		}
		reference.direct = true
		_, blocker := reference.sourceScope.LookupParent(object.Name(), reference.use)
		if blocker == nil || blocker == object || seenBlockers[blocker] {
			continue
		}
		seenBlockers[blocker] = true
		planned := plannedTypeBlocker{intended: object, object: blocker}
		_, planned.typeName = blocker.(*types.TypeName)
		for identifier, defined := range info.Defs {
			if defined == blocker {
				planned.definition = identifier
				planned.identifiers = append(planned.identifiers, identifier)
			}
		}
		for identifier, used := range info.Uses {
			if used == blocker {
				planned.identifiers = append(planned.identifiers, identifier)
			}
		}
		if planned.typeName {
			planned.alias = plannedTypeAlias(object, planned.definition, current)
		}
		reference.blockers = append(reference.blockers, planned)
	}
}

func plannedTypeAlias(
	object types.Object,
	before *ast.Ident,
	current *types.Package,
) *plannedTypeAliasAction {
	action := &plannedTypeAliasAction{
		object: object, before: before,
		packageScope: object.Pkg() != current || object.Parent() == current.Scope(),
	}
	parameters := typeAliasParameters(object.Type())
	if parameters == nil {
		return action
	}
	for index := range parameters.Len() {
		parameter := parameters.At(index)
		action.parameters = append(action.parameters, plannedTypeAliasParameter{
			name: parameter.Obj().Name(), constraint: parameter.Constraint(),
		})
	}
	return action
}

func typeAliasParameters(typ types.Type) *types.TypeParamList {
	switch declared := typ.(type) {
	case *types.Named:
		return declared.TypeParams()
	case *types.Alias:
		return declared.TypeParams()
	default:
		return nil
	}
}

func filePackageObject(file *ast.File, info *types.Info, pkg *types.Package) types.Object {
	for _, specification := range file.Imports {
		object := importPackageName(specification, info)
		if object != nil && object.Imported() == pkg {
			return object
		}
	}
	return nil
}

func typeObjectIsDirect(
	object types.Object,
	current *types.Package,
	file *ast.File,
	info *types.Info,
) bool {
	packageObject := filePackageObject(file, info, object.Pkg())
	return object.Pkg() == nil || object.Pkg() == current || packageObject == nil ||
		packageObject.Name() == "."
}

func deeperTypeScope(candidate, current *types.Scope) bool {
	if candidate == nil || candidate == current {
		return false
	}
	if current == nil {
		return true
	}
	for scope := candidate.Parent(); scope != nil; scope = scope.Parent() {
		if scope == current {
			return true
		}
	}
	return false
}

func visitTypeArguments(arguments *types.TypeList, visit func(types.Type)) {
	if arguments == nil {
		return
	}
	for index := range arguments.Len() {
		visit(arguments.At(index))
	}
}

func visitTypeParameters(parameters *types.TypeParamList, visit func(types.Type)) {
	if parameters == nil {
		return
	}
	for index := range parameters.Len() {
		visit(parameters.At(index))
	}
}

func visitTupleTypes(tuple *types.Tuple, visit func(types.Type)) {
	if tuple == nil {
		return
	}
	for index := range tuple.Len() {
		visit(tuple.At(index).Type())
	}
}
