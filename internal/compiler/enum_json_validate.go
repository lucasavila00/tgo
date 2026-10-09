package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// addEnumJSONNonNilChecks adds payload checks after the first type check.
func (p *packageUnit) addEnumJSONNonNilChecks() bool {
	engine := newEnumJSONContractEngine(p)
	changed := false
	for _, source := range p.Sources {
		for _, declaration := range source.Models {
			if !declaration.Enum {
				continue
			}
			contracts := make(map[string]enumJSONNilContract)
			for _, item := range declaration.Variants {
				name := declaration.Name + item.Name
				object, _ := p.typed.Scope().Lookup(name).(*types.TypeName)
				if contract := engine.finalContract(engine.objectContract(object)); len(contract) != 0 {
					contracts[name] = contract
				}
			}
			if len(contracts) == 0 {
				continue
			}
			functions := []*ast.FuncDecl(nil)
			for _, item := range source.File.Decls {
				function, ok := item.(*ast.FuncDecl)
				if !ok || function.Body == nil || function.Name == nil {
					continue
				}
				receiver, ok := receiverName(function)
				if !ok || receiver != declaration.Name ||
					(function.Name.Name != "UnmarshalJSON" &&
						function.Name.Name != "UnmarshalJSONFrom") {
					continue
				}
				functions = append(functions, function)
			}
			for _, function := range functions {
				changed = p.addEnumJSONChecksToFunction(
					function,
					source.File,
					declaration,
					contracts,
					engine,
					enumJSONErrorFunction(source.File),
				) || changed
			}
		}
	}
	return changed
}

type enumJSONInjection struct {
	unit        *packageUnit
	file        *ast.File
	declaration *model
	contracts   map[string]enumJSONNilContract
	engine      *enumJSONContractEngine
	errorFunc   string
	untagged    bool
	errorName   string
	usedError   bool
	problem     error
	changed     bool
}

func (p *packageUnit) addEnumJSONChecksToFunction(
	function *ast.FuncDecl,
	file *ast.File,
	declaration *model,
	contracts map[string]enumJSONNilContract,
	engine *enumJSONContractEngine,
	errorFunction string,
) bool {
	injection := &enumJSONInjection{
		unit: p, file: file, declaration: declaration, contracts: contracts,
		engine:    engine,
		errorFunc: errorFunction,
		untagged: declaration.JSON.Form == "untagged" &&
			function.Name.Name == "UnmarshalJSON",
	}
	if injection.untagged {
		injection.errorName = freshASTIdentifier(function, "tgoJSONNonNilError")
	}
	injection.statements(&function.Body.List)
	if injection.problem != nil {
		p.errors = append(p.errors, injection.problem)
		return injection.changed
	}
	if !injection.usedError {
		return injection.changed
	}
	declarationStatement := &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(injection.errorName)},
			Type:  ast.NewIdent("error"),
		}},
	}}
	function.Body.List = append([]ast.Stmt{declarationStatement}, function.Body.List...)
	for index := len(function.Body.List) - 1; index >= 0; index-- {
		if _, ok := function.Body.List[index].(*ast.ReturnStmt); !ok {
			continue
		}
		check := &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent(injection.errorName),
				Op: token.NEQ,
				Y:  ast.NewIdent("nil"),
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{
				Results: []ast.Expr{ast.NewIdent(injection.errorName)},
			}}},
		}
		function.Body.List = enumJSONInsertStatements(
			function.Body.List, index, []ast.Stmt{check},
		)
		break
	}
	return injection.changed
}

func (i *enumJSONInjection) statements(statements *[]ast.Stmt) {
	items := *statements
	for index := 0; index < len(items); index++ {
		payloadType, ok := i.payloadDeclaration(items[index])
		if ok && index+1 < len(items) {
			decode, decodeOK := items[index+1].(*ast.IfStmt)
			contract := i.contracts[payloadType]
			if decodeOK && len(contract) != 0 {
				checks := i.validationStatements(payloadType, contract)
				if len(checks) != 0 {
					if i.untagged && enumJSONNilComparison(decode.Cond, token.EQL) {
						i.wrapUntagged(decode, checks)
					} else if enumJSONNilComparison(decode.Cond, token.NEQ) &&
						index+2 < len(items) && enumJSONReceiverAssignment(items[index+2]) {
						position := index + 2
						items = enumJSONInsertStatements(items, position, checks)
						index += len(checks)
						i.changed = true
					}
				}
			}
		}
		i.nested(items[index])
	}
	*statements = items
}

func (i *enumJSONInjection) nested(statement ast.Stmt) {
	switch value := statement.(type) {
	case *ast.BlockStmt:
		i.statements(&value.List)
	case *ast.IfStmt:
		i.statements(&value.Body.List)
		if block, ok := value.Else.(*ast.BlockStmt); ok {
			i.statements(&block.List)
		} else if next, ok := value.Else.(*ast.IfStmt); ok {
			i.nested(next)
		}
	case *ast.SwitchStmt:
		for _, clause := range value.Body.List {
			if item, ok := clause.(*ast.CaseClause); ok {
				i.statements(&item.Body)
			}
		}
	case *ast.TypeSwitchStmt:
		for _, clause := range value.Body.List {
			if item, ok := clause.(*ast.CaseClause); ok {
				i.statements(&item.Body)
			}
		}
	case *ast.ForStmt:
		i.statements(&value.Body.List)
	case *ast.RangeStmt:
		i.statements(&value.Body.List)
	}
}

func (i *enumJSONInjection) payloadDeclaration(statement ast.Stmt) (string, bool) {
	declaration, ok := statement.(*ast.DeclStmt)
	if !ok {
		return "", false
	}
	general, ok := declaration.Decl.(*ast.GenDecl)
	if !ok || general.Tok != token.VAR || len(general.Specs) != 1 {
		return "", false
	}
	value, ok := general.Specs[0].(*ast.ValueSpec)
	if !ok || len(value.Names) != 1 || value.Names[0].Name != "payload" || value.Type == nil {
		return "", false
	}
	typ := types.Unalias(i.unit.info.TypeOf(value.Type))
	named, ok := typ.(*types.Named)
	if !ok || named.Obj() == nil {
		return "", false
	}
	return named.Obj().Name(), true
}

func (i *enumJSONInjection) validationStatements(
	payloadType string,
	contract enumJSONNilContract,
) []ast.Stmt {
	object := i.unit.typed.Scope().Lookup(payloadType)
	if object == nil {
		return nil
	}
	variant := strings.TrimPrefix(payloadType, i.declaration.Name)
	emitter := enumJSONValidationEmitter{
		unit: i.unit, errorFunc: i.errorFunc,
		prefix: "invalid " + i.declaration.Name + "." + variant + " JSON payload: ",
		state: &enumJSONValidationState{
			engine: i.engine, recursive: make(map[string]string),
		},
	}
	emitter.emit(object.Type(), contract, "payload", "")
	code := emitter.state.helperCode() + emitter.output.String()
	if emitter.state.needsReflect {
		qualifier, changed := enumJSONReflectQualifier(i.unit.fs, i.file)
		i.changed = i.changed || changed
		code = strings.ReplaceAll(code, "tgoJSONReflect.", qualifier)
	}
	statements, err := enumJSONParseStatements(i.unit.fs, code)
	if err != nil {
		i.problem = fmt.Errorf("generate enum JSON non-null checks: %w", err)
		return nil
	}
	return statements
}

func (i *enumJSONInjection) wrapUntagged(
	decode *ast.IfStmt,
	checks []ast.Stmt,
) {
	i.usedError = true
	i.changed = true
	localError := freshASTIdentifier(decode, "tgoJSONError")
	body := append([]ast.Stmt(nil), checks...)
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}})
	call := &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{List: []*ast.Field{{
				Type: ast.NewIdent("error"),
			}}},
		},
		Body: &ast.BlockStmt{List: body},
	}}
	inner := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(localError)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{call},
		},
		Cond: &ast.BinaryExpr{
			X: ast.NewIdent(localError), Op: token.EQL, Y: ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{List: decode.Body.List},
		Else: &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: ast.NewIdent(i.errorName), Op: token.EQL, Y: ast.NewIdent("nil"),
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent(i.errorName)},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{ast.NewIdent(localError)},
			}}},
		},
	}
	decode.Body.List = []ast.Stmt{inner}
}

func enumJSONNilComparison(expression ast.Expr, operator token.Token) bool {
	binary, ok := expression.(*ast.BinaryExpr)
	if !ok || binary.Op != operator {
		return false
	}
	identifier, ok := binary.Y.(*ast.Ident)
	return ok && identifier.Name == "nil"
}

func enumJSONInsertStatements(
	statements []ast.Stmt,
	index int,
	inserted []ast.Stmt,
) []ast.Stmt {
	result := make([]ast.Stmt, 0, len(statements)+len(inserted))
	result = append(result, statements[:index]...)
	result = append(result, inserted...)
	return append(result, statements[index:]...)
}

func enumJSONReceiverAssignment(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 {
		return false
	}
	star, ok := assignment.Lhs[0].(*ast.StarExpr)
	if !ok {
		return false
	}
	identifier, ok := star.X.(*ast.Ident)
	return ok && identifier.Name == "v"
}

func enumJSONErrorFunction(file *ast.File) string {
	for _, specification := range file.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != "fmt" {
			continue
		}
		if specification.Name == nil {
			return "fmt.Errorf"
		}
		if specification.Name.Name == "." {
			return "Errorf"
		}
		return specification.Name.Name + ".Errorf"
	}
	return "fmt.Errorf"
}

func enumJSONParseStatements(files *token.FileSet, code string) ([]ast.Stmt, error) {
	if code == "" {
		return nil, nil
	}
	file, err := parser.ParseFile(
		files,
		"",
		"package generated\nfunc validate() error {\n"+code+"return nil\n}",
		0,
	)
	if err != nil {
		return nil, err
	}
	if len(file.Decls) != 1 {
		return nil, fmt.Errorf("generated check has %d declarations", len(file.Decls))
	}
	function, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || function.Body == nil || len(function.Body.List) == 0 {
		return nil, fmt.Errorf("generated check has no function body")
	}
	return function.Body.List[:len(function.Body.List)-1], nil
}

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
		for index := range item.NumFields() {
			child := enumJSONNilChild(contract, "f"+strconv.Itoa(index))
			if len(child) == 0 {
				continue
			}
			field := item.Field(index)
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
