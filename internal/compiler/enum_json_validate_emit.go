package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type enumJSONValidationEmitter struct {
	unit      *packageUnit
	errorFunc string
	prefix    string
	next      int
	state     *enumJSONValidationState
	output    strings.Builder
}

type enumJSONValidationState struct {
	engine       *enumJSONContractEngine
	recursive    map[string]string
	next         int
	needsReflect bool
	helperOrder  []string
	helperBodies map[string]string
}

func (e *enumJSONValidationEmitter) emit(
	typ types.Type,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	if typ == nil || len(contract) == 0 {
		return
	}
	if enumJSONHasRecursiveContract(contract) {
		e.state.needsReflect = true
		helper := e.recursiveHelper(typ)
		if path == "" {
			path = "value"
		}
		fmt.Fprintf(
			&e.output,
			"if err := %s(tgoJSONReflect.ValueOf(&(%s)).Elem(), %s); err != nil { return err }\n",
			helper, value, strconv.Quote(path),
		)
		if len(contract) == 1 {
			return
		}
		contract = enumJSONWithoutRecursiveContract(contract)
	}
	if contract[""] {
		e.failure(value+" == nil", path)
	}
	underlying := enumJSONValidationType(typ)
	switch item := underlying.(type) {
	case *types.Pointer:
		child := enumJSONNilChild(contract, "e")
		if len(child) != 0 {
			fmt.Fprintf(&e.output, "if %s != nil {\n", value)
			e.emit(item.Elem(), child, "(*"+value+")", path)
			e.output.WriteString("}\n")
		}
	case *types.Array:
		e.emitElements(item.Elem(), enumJSONNilChild(contract, "e"), value, path+"[]")
	case *types.Slice:
		e.emitElements(item.Elem(), enumJSONNilChild(contract, "e"), value, path+"[]")
	case *types.Map:
		e.emitMap(item, contract, value, path)
	case *types.Struct:
		e.emitStruct(item, contract, value, path)
	}
}

func (e *enumJSONValidationEmitter) emitElements(
	typ types.Type,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	if len(contract) == 0 {
		return
	}
	item := e.freshValue()
	fmt.Fprintf(&e.output, "for _, %s := range %s {\n", item, value)
	e.emit(typ, contract, item, path)
	e.output.WriteString("}\n")
}

func (e *enumJSONValidationEmitter) emitMap(
	mapping *types.Map,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	keyContract := enumJSONNilChild(contract, "k")
	valueContract := enumJSONNilChild(contract, "v")
	if len(keyContract) == 0 && len(valueContract) == 0 {
		return
	}
	key := "_"
	item := "_"
	if len(keyContract) != 0 {
		key = e.freshValue()
	}
	if len(valueContract) != 0 {
		item = e.freshValue()
	}
	fmt.Fprintf(&e.output, "for %s, %s := range %s {\n", key, item, value)
	if len(keyContract) != 0 {
		e.emit(mapping.Key(), keyContract, key, path+"<key>")
	}
	if len(valueContract) != 0 {
		e.emit(mapping.Elem(), valueContract, item, path+"[]")
	}
	e.output.WriteString("}\n")
}

func (e *enumJSONValidationEmitter) emitStruct(
	structure *types.Struct,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	for index := range structure.NumFields() {
		child := enumJSONNilChild(contract, "f"+strconv.Itoa(index))
		if len(child) == 0 {
			continue
		}
		field := structure.Field(index)
		fieldPath := field.Name()
		if path != "" {
			fieldPath = path + "." + fieldPath
		}
		if field.Name() == "_" {
			if enumJSONZeroViolates(field.Type(), child) {
				e.failure("true", fieldPath)
			}
			continue
		}
		if !field.Exported() && field.Pkg() != nil && field.Pkg().Path() != e.unit.Path {
			e.state.needsReflect = true
			e.emitReflect(
				field.Type(), child,
				fmt.Sprintf("tgoJSONReflect.ValueOf(%s).Field(%d)", value, index),
				fieldPath,
			)
			continue
		}
		e.emit(field.Type(), child, value+"."+field.Name(), fieldPath)
	}
}

func (e *enumJSONValidationEmitter) emitReflect(
	typ types.Type,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	if typ == nil || len(contract) == 0 {
		return
	}
	if enumJSONHasRecursiveContract(contract) {
		helper := e.recursiveHelper(typ)
		if path == "" {
			path = "value"
		}
		fmt.Fprintf(
			&e.output,
			"if err := %s(%s, %s); err != nil { return err }\n",
			helper, value, strconv.Quote(path),
		)
		if len(contract) == 1 {
			return
		}
		contract = enumJSONWithoutRecursiveContract(contract)
	}
	if contract[""] {
		e.failure(value+".IsNil()", path)
	}
	switch item := enumJSONValidationType(typ).(type) {
	case *types.Pointer:
		child := enumJSONNilChild(contract, "e")
		if len(child) != 0 {
			fmt.Fprintf(&e.output, "if !%s.IsNil() {\n", value)
			e.emitReflect(item.Elem(), child, value+".Elem()", path)
			e.output.WriteString("}\n")
		}
	case *types.Array:
		e.emitReflectElements(item.Elem(), enumJSONNilChild(contract, "e"), value, path+"[]")
	case *types.Slice:
		e.emitReflectElements(item.Elem(), enumJSONNilChild(contract, "e"), value, path+"[]")
	case *types.Map:
		e.emitReflectMap(item, contract, value, path)
	case *types.Struct:
		for index := range item.NumFields() {
			child := enumJSONNilChild(contract, "f"+strconv.Itoa(index))
			if len(child) == 0 {
				continue
			}
			field := item.Field(index)
			e.emitReflect(
				field.Type(), child,
				fmt.Sprintf("%s.Field(%d)", value, index),
				path+"."+field.Name(),
			)
		}
	}
}

func enumJSONWithoutRecursiveContract(
	contract enumJSONNilContract,
) enumJSONNilContract {
	result := enumJSONCopyContract(contract)
	for path := range result {
		if strings.HasPrefix(path, enumJSONRecursiveContractPrefix) {
			delete(result, path)
		}
	}
	return result
}

func (e *enumJSONValidationEmitter) recursiveHelper(typ types.Type) string {
	key := types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() })
	if name := e.state.recursive[key]; name != "" {
		return name
	}
	name := fmt.Sprintf("tgoJSONValidate%d", e.state.next)
	e.state.next++
	e.state.recursive[key] = name
	e.state.helperOrder = append(e.state.helperOrder, name)
	if e.state.helperBodies == nil {
		e.state.helperBodies = make(map[string]string)
	}
	e.state.needsReflect = true

	contract := e.state.engine.finalContract(e.state.engine.typeContract(typ))
	body := enumJSONValidationEmitter{
		unit: e.unit, errorFunc: e.errorFunc, prefix: e.prefix, state: e.state,
	}
	body.emitReflectDynamic(typ, contract, "tgoJSONValue", "tgoJSONPath")
	e.state.helperBodies[name] = fmt.Sprintf(
		"%s = func(tgoJSONValue tgoJSONReflect.Value, tgoJSONPath string) error {\n"+
			"if tgoJSONValue.CanAddr() {\n"+
			"tgoJSONPointer := tgoJSONValue.Addr().Pointer()\n"+
			"if tgoJSONVisited[tgoJSONPointer] { return nil }\n"+
			"tgoJSONVisited[tgoJSONPointer] = true\n"+
			"}\n"+
			"%sreturn nil\n}\n",
		name, body.output.String(),
	)
	return name
}

func (s *enumJSONValidationState) helperCode() string {
	if len(s.helperOrder) == 0 {
		return ""
	}
	var output strings.Builder
	output.WriteString("tgoJSONVisited := make(map[uintptr]bool)\n")
	output.WriteString("var (\n")
	for _, name := range s.helperOrder {
		fmt.Fprintf(&output, "%s func(tgoJSONReflect.Value, string) error\n", name)
	}
	output.WriteString(")\n")
	for _, name := range s.helperOrder {
		output.WriteString(s.helperBodies[name])
	}
	return output.String()
}

func (e *enumJSONValidationEmitter) emitReflectDynamic(
	typ types.Type,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	if typ == nil || len(contract) == 0 {
		return
	}
	if enumJSONHasRecursiveContract(contract) {
		helper := e.recursiveHelper(typ)
		fmt.Fprintf(
			&e.output,
			"if err := %s(%s, %s); err != nil { return err }\n",
			helper, value, path,
		)
		if len(contract) == 1 {
			return
		}
		contract = enumJSONWithoutRecursiveContract(contract)
	}
	if contract[""] {
		e.failureDynamic(value+".IsNil()", path)
	}
	switch item := enumJSONValidationType(typ).(type) {
	case *types.Pointer:
		child := enumJSONNilChild(contract, "e")
		if len(child) != 0 {
			fmt.Fprintf(&e.output, "if !%s.IsNil() {\n", value)
			e.emitReflectDynamic(item.Elem(), child, value+".Elem()", path)
			e.output.WriteString("}\n")
		}
	case *types.Array:
		e.emitReflectDynamicElements(
			item.Elem(), enumJSONNilChild(contract, "e"), value,
			path+" + "+strconv.Quote("[]"),
		)
	case *types.Slice:
		e.emitReflectDynamicElements(
			item.Elem(), enumJSONNilChild(contract, "e"), value,
			path+" + "+strconv.Quote("[]"),
		)
	case *types.Map:
		e.emitReflectDynamicMap(item, contract, value, path)
	case *types.Struct:
		e.emitReflectDynamicStruct(item, contract, value, path)
	}
}

func (e *enumJSONValidationEmitter) emitReflectDynamicStruct(
	structure *types.Struct,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	for index := range structure.NumFields() {
		child := enumJSONNilChild(contract, "f"+strconv.Itoa(index))
		if len(child) == 0 {
			continue
		}
		field := structure.Field(index)
		fieldPath := path + " + " + strconv.Quote("."+field.Name())
		if field.Name() == "_" {
			if enumJSONZeroViolates(field.Type(), child) {
				e.failureDynamic("true", fieldPath)
			}
			continue
		}
		e.emitReflectDynamic(
			field.Type(), child,
			fmt.Sprintf("%s.Field(%d)", value, index), fieldPath,
		)
	}
}

func (e *enumJSONValidationEmitter) emitReflectDynamicElements(
	typ types.Type,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	if len(contract) == 0 {
		return
	}
	index := e.freshValue()
	fmt.Fprintf(
		&e.output,
		"for %s := 0; %s < %s.Len(); %s++ {\n",
		index, index, value, index,
	)
	e.emitReflectDynamic(typ, contract, value+".Index("+index+")", path)
	e.output.WriteString("}\n")
}

func (e *enumJSONValidationEmitter) emitReflectDynamicMap(
	mapping *types.Map,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	keyContract := enumJSONNilChild(contract, "k")
	valueContract := enumJSONNilChild(contract, "v")
	if len(keyContract) == 0 && len(valueContract) == 0 {
		return
	}
	iterator := e.freshValue()
	fmt.Fprintf(&e.output, "%s := %s.MapRange()\n", iterator, value)
	fmt.Fprintf(&e.output, "for %s.Next() {\n", iterator)
	if len(keyContract) != 0 {
		e.emitReflectDynamic(
			mapping.Key(), keyContract, iterator+".Key()",
			path+" + "+strconv.Quote("<key>"),
		)
	}
	if len(valueContract) != 0 {
		e.emitReflectDynamic(
			mapping.Elem(), valueContract, iterator+".Value()",
			path+" + "+strconv.Quote("[]"),
		)
	}
	e.output.WriteString("}\n")
}

func (e *enumJSONValidationEmitter) failureDynamic(condition string, path string) {
	fmt.Fprintf(
		&e.output,
		"if %s { return %s(%s, %s) }\n",
		condition, e.errorFunc,
		strconv.Quote(e.prefix+"%s must not be nil"), path,
	)
}

func (e *enumJSONValidationEmitter) emitReflectElements(
	typ types.Type,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	if len(contract) == 0 {
		return
	}
	index := e.freshValue()
	fmt.Fprintf(
		&e.output,
		"for %s := 0; %s < %s.Len(); %s++ {\n",
		index, index, value, index,
	)
	e.emitReflect(typ, contract, value+".Index("+index+")", path)
	e.output.WriteString("}\n")
}

func (e *enumJSONValidationEmitter) emitReflectMap(
	mapping *types.Map,
	contract enumJSONNilContract,
	value string,
	path string,
) {
	keyContract := enumJSONNilChild(contract, "k")
	valueContract := enumJSONNilChild(contract, "v")
	if len(keyContract) == 0 && len(valueContract) == 0 {
		return
	}
	iterator := e.freshValue()
	fmt.Fprintf(&e.output, "%s := %s.MapRange()\n", iterator, value)
	fmt.Fprintf(&e.output, "for %s.Next() {\n", iterator)
	if len(keyContract) != 0 {
		e.emitReflect(mapping.Key(), keyContract, iterator+".Key()", path+"<key>")
	}
	if len(valueContract) != 0 {
		e.emitReflect(mapping.Elem(), valueContract, iterator+".Value()", path+"[]")
	}
	e.output.WriteString("}\n")
}

func (e *enumJSONValidationEmitter) freshValue() string {
	name := fmt.Sprintf("tgoJSONValue%d", e.next)
	e.next++
	return name
}

func (e *enumJSONValidationEmitter) failure(condition string, path string) {
	if path == "" {
		path = "value"
	}
	message := e.prefix + path + " must not be nil"
	fmt.Fprintf(
		&e.output,
		"if %s { return %s(%s) }\n",
		condition,
		e.errorFunc,
		strconv.Quote(message),
	)
}

func enumJSONValidationType(typ types.Type) types.Type {
	for {
		if alias, ok := typ.(*types.Alias); ok {
			typ = alias.Rhs()
			continue
		}
		if named, ok := typ.(*types.Named); ok {
			typ = named.Underlying()
			continue
		}
		return typ.Underlying()
	}
}

func enumJSONZeroViolates(typ types.Type, contract enumJSONNilContract) bool {
	if typ == nil || len(contract) == 0 {
		return false
	}
	if contract[""] {
		return true
	}
	switch item := enumJSONValidationType(typ).(type) {
	case *types.Array:
		return item.Len() > 0 && enumJSONZeroViolates(
			item.Elem(), enumJSONNilChild(contract, "e"),
		)
	case *types.Struct:
		for index := range item.NumFields() {
			if enumJSONZeroViolates(
				item.Field(index).Type(),
				enumJSONNilChild(contract, "f"+strconv.Itoa(index)),
			) {
				return true
			}
		}
	}
	return false
}

func enumJSONReflectQualifier(files *token.FileSet, file *ast.File) (string, bool) {
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != "reflect" {
			continue
		}
		if specification.Name == nil {
			return "reflect.", false
		}
		if specification.Name.Name == "." {
			return "", false
		}
		if specification.Name.Name == "_" {
			name := freshASTIdentifier(file, "reflect")
			specification.Name.Name = name
			return name + ".", true
		}
		return specification.Name.Name + ".", false
	}
	name := freshASTIdentifier(file, "reflect")
	if name == "reflect" {
		astutil.AddImport(files, file, "reflect")
	} else {
		astutil.AddNamedImport(files, file, name, "reflect")
	}
	return name + ".", true
}
