package lowering

import (
	"fmt"
	"github.com/lucasavila00/tgo/internal/lowering/source"
	"go/ast"

	"github.com/lucasavila00/tgo/internal/checker"
	"golang.org/x/tools/go/packages"
)

// Site identifies an expression in the original source. Offsets are bytes.
type Site source.Site

// Source contains the original syntax and its type information.
type Source struct {
	Package   *packages.Package
	Sites     []Site
	inventory source.Inventory
}

// Load reads and type-checks one Go package selected by pattern in dir.
// It records expression positions for later statement insertion.
func Load(dir, pattern string) (*Source, error) {
	pkgs, err := packages.Load(&packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedImports | packages.NeedDeps | packages.NeedExportFile | packages.NeedModule,
	}, pattern)
	if err != nil {
		return nil, err
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected one package, got %d", len(pkgs))
	}
	return SourceFromPackage(pkgs[0], dir)
}

// SourceFromPackage records expression positions and insertion limits in a checked package.
// File names are relative to dir; a package with type errors returns an error.
func SourceFromPackage(p *packages.Package, dir string) (*Source, error) {
	if len(p.Errors) != 0 {
		return nil, fmt.Errorf("load source: %s", p.Errors[0])
	}
	checked, err := checker.CheckGo(p, dir)
	if err != nil {
		return nil, err
	}
	s, err := SourceFromCheckedPackage(checked, dir)
	if err == nil {
		s.Package = p
	}
	return s, err
}

// SourceFromCheckedPackage records expressions from private source checking.
func SourceFromCheckedPackage(p *checker.Package, dir string) (*Source, error) {
	inventory := source.New(p, dir)
	var sites []Site
	for _, site := range inventory.Sites {
		sites = append(sites, Site(site))
	}
	return &Source{Sites: sites, inventory: *inventory}, nil
}

func (s *Source) rangeElided(statement *ast.RangeStmt) bool {
	return source.RangeElided(s.inventory.Checked, statement)
}

func contains(parent, child ast.Node) bool {
	return parent != nil && child != nil && parent.Pos() <= child.Pos() && child.End() <= parent.End()
}
