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
	definitions := make(map[types.Object]*ast.Ident)
	for identifier, object := range info.Defs {
		if object != nil {
			definitions[object] = identifier
		}
	}
	seenObjects := make(map[types.Object]int)
	seenTypes := make(map[types.Type]bool)
	var addObject func(types.Object, bool, bool)
	addObject = func(object types.Object, parameter, direct bool) {
		if object == nil {
			return
		}
		if index, exists := seenObjects[object]; exists {
			reference.objects[index].direct = reference.objects[index].direct || direct
			return
		}
		definition := definitions[object]
		seenObjects[object] = len(reference.objects)
		reference.objects = append(reference.objects, plannedTypeObjectReference{
			object: object, definition: definition, direct: direct,
		})
		if object.Pkg() != nil && object.Pkg() != current {
			packageObject := filePackageObject(file, info, object.Pkg())
			addObject(packageObject, false, packageObject != nil &&
				packageObject.Name() != "." && packageObject.Name() != "_")
		}
		if object.Pkg() != current || object.Parent() == current.Scope() {
			return
		}
		candidate := plannedTypeDeclarationAnchor{owner: object.Parent()}
		if parameter {
			candidate.entry = body
		} else if _, ok := object.(*types.TypeName); ok {
			candidate.after = definition
		} else {
			return
		}
		if deeperTypeScope(candidate.owner, reference.anchor.owner) ||
			candidate.owner == reference.anchor.owner &&
				candidate.after != nil &&
				(reference.anchor.after == nil || candidate.after.Pos() > reference.anchor.after.Pos()) {
			reference.anchor = candidate
		}
	}
	var visit func(types.Type)
	visit = func(item types.Type) {
		if item == nil || seenTypes[item] {
			return
		}
		seenTypes[item] = true
		switch item := item.(type) {
		case *types.Alias:
			addObject(item.Obj(), false, typeObjectIsDirect(item.Obj(), current, file, info))
			visit(item.Rhs())
			visitTypeArguments(item.TypeArgs(), visit)
		case *types.Named:
			addObject(item.Obj(), false, typeObjectIsDirect(item.Obj(), current, file, info))
			visitTypeArguments(item.TypeArgs(), visit)
		case *types.TypeParam:
			addObject(item.Obj(), true, true)
		case *types.Basic:
			if item.Kind() == types.UnsafePointer {
				object := types.Unsafe.Scope().Lookup("Pointer")
				addObject(object, false, typeObjectIsDirect(object, current, file, info))
			} else {
				addObject(types.Universe.Lookup(item.Name()), false, true)
			}
		case *types.Pointer:
			visit(item.Elem())
		case *types.Slice:
			visit(item.Elem())
		case *types.Array:
			visit(item.Elem())
		case *types.Map:
			visit(item.Key())
			visit(item.Elem())
		case *types.Chan:
			visit(item.Elem())
		case *types.Struct:
			for index := range item.NumFields() {
				visit(item.Field(index).Type())
			}
		case *types.Interface:
			for index := range item.NumExplicitMethods() {
				visit(item.ExplicitMethod(index).Type())
			}
			for index := range item.NumEmbeddeds() {
				visit(item.EmbeddedType(index))
			}
		case *types.Union:
			for index := range item.Len() {
				visit(item.Term(index).Type())
			}
		case *types.Signature:
			visitTypeParameters(item.TypeParams(), visit)
			visitTupleTypes(item.Params(), visit)
			visitTupleTypes(item.Results(), visit)
		case *types.Tuple:
			visitTupleTypes(item, visit)
		}
	}
	visit(typ)
	seenBlockers := make(map[types.Object]bool)
	for _, required := range reference.objects {
		object := required.object
		if object == nil || !required.direct || sourceScope == nil {
			continue
		}
		reference.direct = true
		_, blocker := sourceScope.LookupParent(object.Name(), use)
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
			action := &plannedTypeAliasAction{
				object: object, before: planned.definition,
				packageScope: object.Pkg() != current || object.Parent() == current.Scope(),
			}
			if named, ok := object.Type().(*types.Named); ok && named.TypeParams() != nil {
				for index := range named.TypeParams().Len() {
					parameter := named.TypeParams().At(index)
					action.parameters = append(action.parameters, plannedTypeAliasParameter{
						name: parameter.Obj().Name(), constraint: parameter.Constraint(),
					})
				}
			}
			planned.alias = action
		}
		reference.blockers = append(reference.blockers, planned)
	}
	if reference.anchor.after != nil || reference.anchor.entry != nil {
		reference.direct = true
	}
	if reference.anchor.owner == nil {
		reference.anchor.owner = sourceScope
	}
	return reference
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
