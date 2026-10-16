package shared

import (
	"go/token"
	"strconv"

	"github.com/lucasavila00/tgo/internal/ast"
	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// EnumUnevaluated keeps source operations that Go does not evaluate out of generated execution.
func (r *Context) EnumUnevaluated(expression ast.Expr) (Result, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || !r.hasEnumConstruction(call) {
		return Result{}, false
	}
	if r.unsafeOperand(call) {
		if r.input.Inventory.Checked.TypesInfo.Types[call].Value != nil {
			return r.enumUnsafeConstant(call), true
		}
		return r.enumUnsafeOperand(call), true
	}
	builtin := r.checkedBuiltin(call.Fun)
	if builtin == nil || (builtin.Name() != "len" && builtin.Name() != "cap") {
		return Result{}, false
	}
	value := r.input.Inventory.Checked.TypesInfo.Types[call].Value
	if value == nil {
		return Result{}, false
	}
	before := r.elidedBranches(call.Args[0])
	number := ast.Expr(&ast.BasicLit{Kind: token.INT, Value: value.ExactString()})
	if references := r.elidedReferences(call); len(references) != 0 {
		marker := &ast.CallExpr{Fun: call.Fun, Args: []ast.Expr{useArray(references)}}
		zero := &ast.BinaryExpr{X: marker, Op: token.SUB, Y: Integer(1)}
		number = &ast.BinaryExpr{X: number, Op: token.ADD, Y: zero}
	}
	r.discardEnumOperand(call)
	array := &ast.CompositeLit{Type: &ast.ArrayType{Len: number, Elt: &ast.StructType{Fields: &ast.FieldList{}}}}
	result := &ast.CallExpr{Fun: call.Fun, Args: []ast.Expr{array}}
	return r.FinishExpression(call, before, result), true
}

// hasEnumConstruction finds source enum literals, including literals in uncalled functions.
func (r *Context) hasEnumConstruction(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if literal, ok := node.(*ast.CompositeLit); ok {
			found = found || r.input.Inventory.Checked.TypesInfo.EnumConstructors[literal] != nil
		}
		return !found
	})
	return found
}

// checkedBuiltin resolves a builtin by the checked source binding.
func (r *Context) checkedBuiltin(expression ast.Expr) *types.Builtin {
	var name *ast.Ident
	switch function := ast.Unparen(expression).(type) {
	case *ast.Ident:
		name = function
	case *ast.SelectorExpr:
		name = function.Sel
	}
	builtin, _ := r.input.Inventory.Checked.TypesInfo.Uses[name].(*types.Builtin)
	return builtin
}

// enumUnsafeOperand keeps the checked operand type and field path without evaluating source code.
func (r *Context) enumUnsafeOperand(call *ast.CallExpr) Result {
	operand := call.Args[0]
	base := operand
	selector, selected := ast.Unparen(operand).(*ast.SelectorExpr)
	offset := r.checkedBuiltin(call.Fun).Name() == "Offsetof" && selected
	if offset {
		base = selector.X
	}
	pointer := &ast.StarExpr{X: r.TemporaryType(r.TypeOf(base), base)}
	body := referenceStatements(r.elidedReferences(operand))
	body = append(body, &ast.ReturnStmt{})
	zero := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{},
		Results: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{r.Name()}, Type: pointer}}}},
		Body: &ast.BlockStmt{List: body}}
	var replacement ast.Expr = &ast.StarExpr{X: &ast.CallExpr{Fun: zero}}
	if offset {
		field := *selector
		field.X = replacement
		replacement = &field
	}
	before := r.elidedBranches(operand)
	output := *call
	output.Args = []ast.Expr{replacement}
	r.discardEnumOperand(operand)
	return r.FinishExpression(call, before, &output)
}

// enumUnsafeConstant keeps uintptr constants without naming the discarded operand's type.
func (r *Context) enumUnsafeConstant(call *ast.CallExpr) Result {
	value := r.input.Inventory.Checked.TypesInfo.Types[call].Value
	number := &ast.BasicLit{Kind: token.INT, Value: value.ExactString()}
	references := r.elidedReferences(call.Args[0])
	var zero ast.Expr
	if r.checkedBuiltin(call.Fun).Name() == "Offsetof" {
		field := ast.NewIdent("value")
		operand := &ast.CompositeLit{Type: &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{
			{Names: []*ast.Ident{field}, Type: useArray(nil).Type},
		}}}, Elts: []ast.Expr{useArray(references)}}
		zero = &ast.CallExpr{Fun: call.Fun, Args: []ast.Expr{&ast.SelectorExpr{X: operand, Sel: field}}}
	} else {
		zero = &ast.BinaryExpr{X: &ast.CallExpr{Fun: call.Fun, Args: []ast.Expr{useArray(references)}}, Op: token.SUB,
			Y: &ast.CallExpr{Fun: call.Fun, Args: []ast.Expr{useArray(nil)}}}
	}
	before := r.elidedBranches(call.Args[0])
	r.discardEnumOperand(call.Args[0])
	return r.FinishExpression(call, before, &ast.BinaryExpr{X: number, Op: token.ADD, Y: zero})
}

// elidedReferences keeps uses of variables declared outside discarded source syntax.
func (r *Context) elidedReferences(expression ast.Expr) []*ast.Ident {
	var references []*ast.Ident
	seen := make(map[*types.Var]bool)
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := r.input.Inventory.Checked.TypesInfo.Uses[identifier].(*types.Var)
		if !ok || variable.IsField() || seen[variable] {
			return true
		}
		if variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
			return true
		}
		if variable.Pos() < expression.Pos() || variable.Pos() >= expression.End() {
			seen[variable] = true
			references = append(references, identifier)
		}
		return true
	})
	return references
}

// referenceStatements marks variable uses without copying source operations or jumps.
func referenceStatements(references []*ast.Ident) []ast.Stmt {
	var statements []ast.Stmt
	for _, reference := range references {
		statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")},
			Tok: token.ASSIGN, Rhs: []ast.Expr{reference}})
	}
	return statements
}

// useArray holds variable references in an uncalled function for a constant length operation.
func useArray(references []*ast.Ident) *ast.CompositeLit {
	function := &ast.FuncType{Params: &ast.FieldList{}}
	return &ast.CompositeLit{Type: &ast.ArrayType{Len: Integer(1), Elt: function},
		Elts: []ast.Expr{&ast.FuncLit{Type: function, Body: &ast.BlockStmt{List: referenceStatements(references)}}}}
}

// elidedBranches retains only jumps to labels outside the discarded operand.
func (r *Context) elidedBranches(expression ast.Expr) []ast.Stmt {
	labels := make(map[string]bool)
	var branches []*ast.BranchStmt
	ast.Inspect(expression, func(node ast.Node) bool {
		if _, function := node.(*ast.FuncLit); function {
			return false
		}
		switch node := node.(type) {
		case *ast.LabeledStmt:
			labels[node.Label.Name] = true
		case *ast.BranchStmt:
			if node.Label != nil {
				branches = append(branches, node)
			}
		}
		return true
	})
	var body []ast.Stmt
	for _, branch := range branches {
		if !labels[branch.Label.Name] {
			body = append(body, r.Statement(branch)...)
		}
	}
	if len(body) == 0 {
		return nil
	}
	return []ast.Stmt{&ast.IfStmt{Cond: Boolean(false), Body: &ast.BlockStmt{List: body}}}
}

// discardEnumOperand records checked syntax removed because its source value is never evaluated.
func (r *Context) discardEnumOperand(expression ast.Expr) {
	r.discardEnumTree(expression, make(map[ast.Node]bool))
}

// discardEnumTree also visits inherited constants inside discarded function bodies.
func (r *Context) discardEnumTree(expression ast.Expr, seen map[ast.Node]bool) {
	if expression == nil || seen[expression] {
		return
	}
	r.file.discardedEnumOperands = append(r.file.discardedEnumOperands, expression)
	ast.Inspect(expression, func(node ast.Node) bool {
		if node == nil || seen[node] {
			return false
		}
		seen[node] = true
		if expression, ok := node.(ast.Expr); ok && r.isTarget(expression) {
			r.CompleteExpression(expression)
		}
		if statement, ok := node.(*ast.ShortReturnStmt); ok {
			r.request.loweredReturns[statement] = true
		}
		if specification, ok := node.(*ast.ValueSpec); ok {
			r.discardConstantCopies(specification, seen)
		}
		return true
	})
}

// discardConstantCopies completes each checked initializer without repeating shared source nodes.
func (r *Context) discardConstantCopies(specification *ast.ValueSpec, seen map[ast.Node]bool) {
	for _, name := range specification.Names {
		object, ok := r.input.Inventory.Checked.TypesInfo.Defs[name].(*types.Const)
		if !ok {
			continue
		}
		initializer := r.input.Inventory.Checked.TypesInfo.ConstInitializers[object]
		r.discardEnumTree(initializer.Type, seen)
		r.discardEnumTree(initializer.Value, seen)
	}
}

// RangeOperand preserves skipped array evaluation before the generated constructor calls exist.
func (r *Context) RangeOperand(expression, value ast.Expr) Result {
	checked := r.input.Inventory.Checked
	if !r.hasEnumConstruction(expression) || !source.RangeOperandElided(checked, expression, value) {
		return r.Expression(expression)
	}
	typ := r.TypeOf(expression).Underlying()
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem().Underlying()
	}
	array, ok := typ.(*types.Array)
	if !ok {
		return r.Expression(expression)
	}
	length := ast.Expr(&ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(array.Len(), 10)})
	if references := r.elidedReferences(expression); len(references) != 0 {
		length = &ast.BinaryExpr{X: length, Op: token.ADD, Y: r.rangeUseMarker(references)}
	}
	before := r.elidedBranches(expression)
	r.discardEnumOperand(expression)
	operand := &ast.CompositeLit{Type: &ast.ArrayType{Len: length, Elt: &ast.StructType{Fields: &ast.FieldList{}}}}
	return r.FinishExpression(expression, before, &ast.ParenExpr{X: operand})
}

// rangeUseMarker adds a constant zero that retains variable uses even when len and cap are hidden.
func (r *Context) rangeUseMarker(references []*ast.Ident) ast.Expr {
	if r.file.enumUnsafeAlias == nil {
		r.file.enumUnsafeAlias = r.NameFor("unsafe")
		r.file.extraImports = append(r.file.extraImports, &ast.ImportSpec{Name: r.file.enumUnsafeAlias,
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"unsafe"`}})
	}
	size := &ast.SelectorExpr{X: r.file.enumUnsafeAlias, Sel: ast.NewIdent("Sizeof")}
	zero := &ast.BinaryExpr{X: &ast.CallExpr{Fun: size, Args: []ast.Expr{useArray(references)}}, Op: token.SUB,
		Y: &ast.CallExpr{Fun: size, Args: []ast.Expr{useArray(nil)}}}
	return zero
}
