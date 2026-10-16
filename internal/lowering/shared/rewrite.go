package shared

import (
	"github.com/lucasavila00/tgo/internal/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/lowering/names"
	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

// Context owns one function rewrite and shares file and request state.
type Context struct {
	input         *Input
	dispatch      Dispatch
	request       *requestState
	file          *fileState
	functionState *functionState
	// addressRoot delays insertion at this address operand until its checks finish.
	addressRoot ast.Expr
}

// Input holds source facts and the selected operations. Build it before emission.
type Input struct {
	Inventory *source.Inventory
	Target    ast.Expr
	After     After
	TGo       bool
}

// requestState owns progress and fresh names for the complete output request.
type requestState struct {
	names          names.Allocator
	inserted       bool
	lowered        map[ast.Expr]bool
	loweredReturns map[*ast.ShortReturnStmt]bool
	err            error
}

// functionState owns evaluation and control flow within one function.
type functionState struct {
	hasGoto        bool
	labels         map[string]string
	breakExits     map[*ast.BranchStmt]*ast.Ident
	controlTargets map[ast.Node]ast.Stmt
	frame          *returnFrame
	deferredBases  map[ast.Expr]bool
	values         map[ast.Expr]Value
}

// Name allocates a fresh output identifier.
func (r *Context) Name() *ast.Ident {
	return r.NameFor("")
}

// NameFor adds the source name to a fresh generated identifier.
func (r *Context) NameFor(sourceName string) *ast.Ident {
	return r.request.names.Fresh(sourceName)
}

// fileState owns output imports, declarations, and binding changes.
// Binding changes and type aliases stay available to closures in the file.
type fileState struct {
	bindings          map[types.Object]string
	parameterAliases  map[*types.TypeParam]*ast.Ident
	aliasStatements   map[ast.Node][]ast.Stmt
	formatAlias       *ast.Ident
	extraImports      []*ast.ImportSpec
	extraDeclarations []ast.Decl
	zeroSequence      *ast.Ident
	iterAlias         *ast.Ident
	mapAlias          *ast.Ident
	sliceAlias        *ast.Ident
	reflectAlias      *ast.Ident
	typeImports       map[*types.Package]*ast.Ident
}

// New starts a rewrite from complete source facts and selected operations.
// The caller must keep input maps and syntax unchanged while the rewrite runs.
func New(input Input, dispatch Dispatch) *Context {
	return &Context{
		input:    &input,
		dispatch: dispatch,
		request: &requestState{
			lowered:        make(map[ast.Expr]bool),
			loweredReturns: make(map[*ast.ShortReturnStmt]bool),
		},
		file:          &fileState{},
		functionState: &functionState{},
	}
}

// Output contains the generated files and the selected insertion's completion.
type Output struct {
	Files    map[string][]byte
	Inserted bool
}
