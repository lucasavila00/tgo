package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

// addEnumJSONNonNilChecks adds payload checks after the first type check.
func (p *packageUnit) addEnumJSONNonNilChecks() bool {
	engine := newEnumJSONContractEngine(p)
	changed := false
	for _, source := range p.Sources {
		for _, declaration := range source.Models {
			changed = p.addEnumJSONNonNilChecksToModel(
				source, declaration, engine,
			) || changed
		}
	}
	return changed
}

func (p *packageUnit) addEnumJSONNonNilChecksToModel(
	source *source,
	declaration *model,
	engine *enumJSONContractEngine,
) bool {
	if !declaration.Enum {
		return false
	}
	contracts := make(map[string]enumJSONNilContract)
	for _, item := range declaration.Variants {
		name := declaration.Name + item.Name
		object, _ := p.typed.Scope().Lookup(name).(*types.TypeName)
		contract := engine.finalContract(engine.objectContract(object))
		if len(contract) != 0 {
			contracts[name] = contract
		}
	}
	if len(contracts) == 0 {
		return false
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
	changed := false
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
		items, index = i.injectValidation(items, index)
		i.nested(items[index])
	}
	*statements = items
}

func (i *enumJSONInjection) injectValidation(
	items []ast.Stmt,
	index int,
) ([]ast.Stmt, int) {
	payloadType, ok := i.payloadDeclaration(items[index])
	if !ok || index+1 >= len(items) {
		return items, index
	}
	decode, ok := items[index+1].(*ast.IfStmt)
	contract := i.contracts[payloadType]
	if !ok || len(contract) == 0 {
		return items, index
	}
	checks := i.validationStatements(payloadType, contract)
	if len(checks) == 0 {
		return items, index
	}
	if i.untagged && enumJSONNilComparison(decode.Cond, token.EQL) {
		i.wrapUntagged(decode, checks)
		return items, index
	}
	if !enumJSONNilComparison(decode.Cond, token.NEQ) ||
		index+2 >= len(items) || !enumJSONReceiverAssignment(items[index+2]) {
		return items, index
	}
	items = enumJSONInsertStatements(items, index+2, checks)
	i.changed = true
	return items, index + len(checks)
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
