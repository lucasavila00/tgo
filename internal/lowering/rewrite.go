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
	file                *fileState
	source              *Source
	target              ast.Expr
	after               After
	names               names.Allocator
	inserted            bool
	labels              map[string]string
	breakExits          map[*ast.BranchStmt]*ast.Ident
	controlTargets      map[ast.Stmt]ast.Stmt
	bindings            map[types.Object]string
	suppressedRoot      ast.Expr
	propagations        map[ast.Expr]Propagation
	comprehensions      map[*ast.CallExpr]*frontend.Comprehension
	comprehensionRanges map[*ast.RangeStmt]bool
	collectionOutputs   map[string]*collectionDestination
	conditionals        map[*ast.CallExpr]*ast.IfStmt
	lowered             map[ast.Expr]bool
	shortReturns        map[*ast.ReturnStmt]frontend.ShortReturn
	loweredReturns      map[*ast.ReturnStmt]bool
	frame               *returnFrame
	suppressed          bool
	origins             map[ast.Expr]ast.Expr
	parameterAliases    map[*types.TypeParam]*ast.Ident
	aliasStatements     map[ast.Node][]ast.Stmt
	deferredBases       map[ast.Expr]bool
	temporaryTypes      map[ast.Expr]types.Type
	err                 error
}

func (r *rewrite) name() *ast.Ident {
	return r.nameFor("")
}

// nameFor adds the source name to a fresh generated identifier.
func (r *rewrite) nameFor(sourceName string) *ast.Ident {
	return r.names.Fresh(sourceName)
}

// fileState owns imports and declarations added to one output file.
type fileState struct {
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
