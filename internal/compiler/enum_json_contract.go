package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

type enumJSONNilContract map[string]bool

type enumJSONTypeDeclaration struct {
	expression ast.Expr
	nonNil     map[token.Pos]bool
}

type enumJSONContractEngine struct {
	unit          *packageUnit
	engines       map[*packageUnit]*enumJSONContractEngine
	declarations  map[string]enumJSONTypeDeclaration
	contracts     map[string]enumJSONNilContract
	resolving     map[string]bool
	typeContracts map[string]enumJSONNilContract
	typeResolving map[string]bool
}

const enumJSONRecursiveContractPrefix = "@"

func newEnumJSONContractEngine(unit *packageUnit) *enumJSONContractEngine {
	engines := make(map[*packageUnit]*enumJSONContractEngine)
	return enumJSONEngineFor(unit, engines)
}

func enumJSONEngineFor(
	unit *packageUnit,
	engines map[*packageUnit]*enumJSONContractEngine,
) *enumJSONContractEngine {
	if engine := engines[unit]; engine != nil {
		return engine
	}
	engine := &enumJSONContractEngine{
		unit: unit, engines: engines,
		declarations:  make(map[string]enumJSONTypeDeclaration),
		contracts:     make(map[string]enumJSONNilContract),
		resolving:     make(map[string]bool),
		typeContracts: make(map[string]enumJSONNilContract),
		typeResolving: make(map[string]bool),
	}
	engines[unit] = engine
	sources := make(map[*ast.File]*source)
	for _, item := range unit.Sources {
		sources[item.File] = item
	}
	for _, file := range unit.Files {
		var nonNil map[token.Pos]bool
		if item := sources[file]; item != nil {
			nonNil = item.NonNil
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				item, ok := specification.(*ast.TypeSpec)
				if !ok {
					continue
				}
				engine.declarations[item.Name.Name] = enumJSONTypeDeclaration{
					expression: item.Type,
					nonNil:     nonNil,
				}
			}
		}
	}
	return engine
}

func (e *enumJSONContractEngine) typeContract(typ types.Type) enumJSONNilContract {
	if typ == nil {
		return nil
	}
	if alias, ok := typ.(*types.Alias); ok {
		result := enumJSONCopyContract(e.objectContract(alias.Obj()))
		enumJSONMergeContract(result, e.typeContract(alias.Rhs()))
		return enumJSONContractOrNil(result)
	}
	if named, ok := typ.(*types.Named); ok {
		if declaration := e.unit.modelForType(named); declaration != nil && declaration.Enum {
			return nil
		}
		key := enumJSONContractTypeKey(types.TypeString(named, func(pkg *types.Package) string {
			return pkg.Path()
		}))
		if contract, ok := e.typeContracts[key]; ok {
			return contract
		}
		if e.typeResolving[key] {
			return enumJSONNilContract{enumJSONRecursiveContractPrefix + key: true}
		}
		e.typeResolving[key] = true
		result := enumJSONCopyContract(e.objectContract(named.Obj()))
		enumJSONMergeContract(result, e.typeContract(named.Underlying()))
		delete(e.typeResolving, key)
		result = enumJSONContractOrNil(result)
		e.typeContracts[key] = result
		return result
	}
	result := make(enumJSONNilContract)
	switch value := typ.Underlying().(type) {
	case *types.Pointer:
		enumJSONAddNilPath(result, "e", e.typeContract(value.Elem()))
	case *types.Array:
		enumJSONAddNilPath(result, "e", e.typeContract(value.Elem()))
	case *types.Slice:
		enumJSONAddNilPath(result, "e", e.typeContract(value.Elem()))
	case *types.Map:
		enumJSONAddNilPath(result, "k", e.typeContract(value.Key()))
		enumJSONAddNilPath(result, "v", e.typeContract(value.Elem()))
	case *types.Chan:
		enumJSONAddNilPath(result, "e", e.typeContract(value.Elem()))
	case *types.Struct:
		for index := range value.NumFields() {
			enumJSONAddNilPath(
				result,
				"f"+strconv.Itoa(index),
				e.typeContract(value.Field(index).Type()),
			)
		}
	}
	return enumJSONContractOrNil(result)
}

func (e *enumJSONContractEngine) objectContract(
	object *types.TypeName,
) enumJSONNilContract {
	if object == nil || object.Pkg() == nil {
		return nil
	}
	if object.Pkg().Path() != e.unit.Path {
		owner := e.unit.Imports[object.Pkg().Path()]
		if owner == nil || owner.typed == nil {
			return nil
		}
		mapped, _ := owner.typed.Scope().Lookup(object.Name()).(*types.TypeName)
		return enumJSONEngineFor(owner, e.engines).objectContract(mapped)
	}
	name := object.Name()
	if contract, ok := e.contracts[name]; ok {
		return contract
	}
	if e.resolving[name] {
		key := enumJSONContractTypeKey(object.Pkg().Path() + "." + name)
		return enumJSONNilContract{enumJSONRecursiveContractPrefix + key: true}
	}
	declaration, ok := e.declarations[name]
	if !ok {
		return nil
	}
	e.resolving[name] = true
	contract := e.expressionContract(declaration.expression, declaration.nonNil)
	delete(e.resolving, name)
	e.contracts[name] = contract
	return contract
}

func (e *enumJSONContractEngine) finalContract(
	contract enumJSONNilContract,
) enumJSONNilContract {
	result := make(enumJSONNilContract)
	for path := range contract {
		marker := enumJSONRecursiveMarker(path)
		if marker == "" || e.typeHasConcreteContract(marker, make(map[string]bool)) {
			result[path] = true
		}
	}
	return enumJSONContractOrNil(result)
}

func (e *enumJSONContractEngine) typeHasConcreteContract(
	key string,
	visiting map[string]bool,
) bool {
	if visiting[key] {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	for path := range e.typeContracts[key] {
		marker := enumJSONRecursiveMarker(path)
		if marker == "" || e.typeHasConcreteContract(marker, visiting) {
			return true
		}
	}
	return false
}

func enumJSONRecursiveMarker(path string) string {
	segment := path
	if index := strings.LastIndexByte(path, '/'); index >= 0 {
		segment = path[index+1:]
	}
	if !strings.HasPrefix(segment, enumJSONRecursiveContractPrefix) {
		return ""
	}
	return strings.TrimPrefix(segment, enumJSONRecursiveContractPrefix)
}

func enumJSONContractTypeKey(key string) string {
	return strings.ReplaceAll(key, "/", "\x00")
}

func enumJSONHasRecursiveContract(contract enumJSONNilContract) bool {
	for path := range contract {
		if strings.HasPrefix(path, enumJSONRecursiveContractPrefix) {
			return true
		}
	}
	return false
}

func enumJSONCopyContract(source enumJSONNilContract) enumJSONNilContract {
	result := make(enumJSONNilContract, len(source))
	for path := range source {
		result[path] = true
	}
	return result
}

func enumJSONMergeContract(target, source enumJSONNilContract) {
	for path := range source {
		target[path] = true
	}
}

func (e *enumJSONContractEngine) expressionContract(
	expression ast.Expr,
	nonNil map[token.Pos]bool,
) enumJSONNilContract {
	if expression == nil {
		return nil
	}
	result := make(enumJSONNilContract)
	switch value := expression.(type) {
	case *ast.ParenExpr:
		return e.expressionContract(value.X, nonNil)
	case *ast.StarExpr:
		if nonNil[value.Star] {
			result[""] = true
		}
		enumJSONAddNilPath(result, "e", e.expressionContract(value.X, nonNil))
	case *ast.ArrayType:
		enumJSONAddNilPath(result, "e", e.expressionContract(value.Elt, nonNil))
	case *ast.MapType:
		enumJSONAddNilPath(result, "k", e.expressionContract(value.Key, nonNil))
		enumJSONAddNilPath(result, "v", e.expressionContract(value.Value, nonNil))
	case *ast.ChanType:
		enumJSONAddNilPath(result, "e", e.expressionContract(value.Value, nonNil))
	case *ast.StructType:
		index := 0
		for _, field := range value.Fields.List {
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			contract := e.expressionContract(field.Type, nonNil)
			for range count {
				enumJSONAddNilPath(result, "f"+strconv.Itoa(index), contract)
				index++
			}
		}
	case *ast.Ellipsis:
		enumJSONAddNilPath(result, "e", e.expressionContract(value.Elt, nonNil))
	default:
		return e.typeContract(e.unit.info.TypeOf(expression))
	}
	return enumJSONContractOrNil(result)
}

func enumJSONAddNilPath(
	target enumJSONNilContract,
	prefix string,
	source enumJSONNilContract,
) {
	for path := range source {
		if path == "" {
			target[prefix] = true
		} else {
			target[prefix+"/"+path] = true
		}
	}
}

func enumJSONNilChild(
	source enumJSONNilContract,
	prefix string,
) enumJSONNilContract {
	result := make(enumJSONNilContract)
	for path := range source {
		if path == prefix {
			result[""] = true
		} else if rest, ok := strings.CutPrefix(path, prefix+"/"); ok {
			result[rest] = true
		}
	}
	return enumJSONContractOrNil(result)
}

func enumJSONContractOrNil(contract enumJSONNilContract) enumJSONNilContract {
	if len(contract) == 0 {
		return nil
	}
	return contract
}
