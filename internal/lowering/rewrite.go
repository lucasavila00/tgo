package lowering

import (
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker/types"
	"github.com/lucasavila00/tgo/internal/frontend"
	"github.com/lucasavila00/tgo/internal/lowering/names"
)

// After receives references to the evaluated expression results.
type After func(results []ast.Expr) []ast.Stmt

type rewrite struct {
	input         *loweringInput
	request       *rewriteRequest
	file          *fileState
	functionState *functionState
}

// loweringInput holds source facts and the selected operations. Build it before emission.
type loweringInput struct {
	source         *Source
	target         ast.Expr
	after          After
	propagations   map[ast.Expr]Propagation
	comprehensions map[*ast.CallExpr]*frontend.Comprehension
	conditionals   map[*ast.CallExpr]ast.Stmt
	shortReturns   map[*ast.ReturnStmt]frontend.ShortReturn
}

// rewriteRequest owns progress and fresh names for the complete output request.
type rewriteRequest struct {
	names          names.Allocator
	inserted       bool
	lowered        map[ast.Expr]bool
	loweredReturns map[*ast.ReturnStmt]bool
	err            error
}

// functionState owns evaluation and control flow within one function.
type functionState struct {
	hasGoto             bool
	labels              map[string]string
	breakExits          map[*ast.BranchStmt]*ast.Ident
	controlTargets      map[ast.Stmt]ast.Stmt
	suppressedRoot      ast.Expr
	comprehensionRanges map[*ast.RangeStmt]bool
	collectionOutputs   map[string]*collectionDestination
	frame               *returnFrame
	suppressed          bool
	origins             map[ast.Expr]ast.Expr
	deferredBases       map[ast.Expr]bool
	temporaryTypes      map[ast.Expr]types.Type
}

func (r *rewrite) name() *ast.Ident {
	return r.nameFor("")
}

// nameFor adds the source name to a fresh generated identifier.
func (r *rewrite) nameFor(sourceName string) *ast.Ident {
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

func newRewrite(source *Source) *rewrite {
	return &rewrite{
		input:         &loweringInput{source: source},
		request:       &rewriteRequest{},
		file:          &fileState{},
		functionState: &functionState{},
	}
}
