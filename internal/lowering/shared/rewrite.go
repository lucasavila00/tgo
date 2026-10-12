package shared

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
	"github.com/lucasavila00/tgo/internal/lowering/names"
	"github.com/lucasavila00/tgo/internal/lowering/source"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

// Context owns one function rewrite and shares file and request state.
type Context struct {
	Input         *Input
	Request       *Request
	file          *fileState
	functionState *functionState
}

// Input holds source facts and the selected operations. Build it before emission.
type Input struct {
	Inventory      *source.Inventory
	Dispatch       Dispatch
	Target         ast.Expr
	After          After
	Propagations   map[ast.Expr]Propagation
	Comprehensions map[*ast.CallExpr]*frontend.Comprehension
	Conditionals   map[*ast.CallExpr]ast.Stmt
	ShortReturns   map[*ast.ReturnStmt]frontend.ShortReturn
}

// Request owns progress and fresh names for the complete output request.
type Request struct {
	names          names.Allocator
	Inserted       bool
	Lowered        map[ast.Expr]bool
	LoweredReturns map[*ast.ReturnStmt]bool
	Err            error
}

// functionState owns evaluation and control flow within one function.
type functionState struct {
	hasGoto             bool
	labels              map[string]string
	breakExits          map[*ast.BranchStmt]*ast.Ident
	controlTargets      map[ast.Stmt]ast.Stmt
	suppressedRoot      ast.Expr
	comprehensionRanges map[*ast.RangeStmt]bool
	collectionOutputs   map[string]*CollectionDestination
	frame               *returnFrame
	suppressed          bool
	deferredBases       map[ast.Expr]bool
	values              map[ast.Expr]Value
}

// Name allocates a fresh output identifier.
func (r *Context) Name() *ast.Ident {
	return r.NameFor("")
}

// NameFor adds the source name to a fresh generated identifier.
func (r *Context) NameFor(sourceName string) *ast.Ident {
	return r.Request.names.Fresh(sourceName)
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

// New starts a rewrite with checked source facts and current-context dispatch.
func New(inventory *source.Inventory, dispatch Dispatch) *Context {
	return &Context{
		Input:         &Input{Inventory: inventory, Dispatch: dispatch},
		Request:       &Request{},
		file:          &fileState{},
		functionState: &functionState{},
	}
}
