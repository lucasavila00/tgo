package frontend

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
)

// Parsed contains Go syntax and records for added TGo forms.
type Parsed struct {
	File           *ast.File
	Comprehensions map[string]*Comprehension
	Conditionals   map[string]token.Position
	Propagations   map[string]Marker
	ShortReturns   map[string]ShortReturn
}

// ParseFile parses added expressions and maps syntax positions to the input.
// Conditional bodies remain statements in the original enclosing function.
func ParseFile(fset *token.FileSet, filename, sourceName string, input []byte, used map[string]bool) (*Parsed,
	error) {
	original := fset.AddFile(filename, -1, len(input))
	if sourceName != filename {
		original.AddLineColumnInfo(0, sourceName, 1, 1)
	}
	reserveIdentifiers(original, input, used)
	preparation := sourcePreparation{
		source: originalSource(input), filename: filename, original: original, used: used,
		records: &Parsed{Comprehensions: make(map[string]*Comprehension),
			Conditionals: make(map[string]token.Position), Propagations: make(map[string]Marker),
			ShortReturns: make(map[string]ShortReturn)},
	}
	preparation.preparePostfix(sourceName, input)
	syntax, err := preparation.parse()
	if err != nil {
		return nil, err
	}
	return preparation.records.finish(fset, syntax.file, syntax.positions, original, preparation.source.offsets), nil
}

func (preparation *sourcePreparation) parse() (preparedSyntax, error) {
	conditionals := conditionalSyntax{
		filename: preparation.filename, original: preparation.original, used: preparation.used,
	}
	for {
		if err := preparation.prepareExpressions(conditionals); err != nil {
			return preparedSyntax{}, err
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, preparation.filename, preparation.source.text,
			parser.ParseComments|parser.AllErrors|parser.SkipObjectResolution)
		if err == nil {
			return preparedSyntax{file, positions.File(file.Pos())}, nil
		}
		diagnostics, isSyntaxError := err.(scanner.ErrorList)
		if !isSyntaxError {
			return preparedSyntax{}, err
		}
		replacement, recoveryError := conditionals.recover(preparation.source, file, positions)
		if recoveryError != nil {
			return preparedSyntax{}, recoveryError
		}
		if replacement == nil {
			return preparedSyntax{}, preparation.inputDiagnostics(diagnostics)
		}
		preparation.applyConditionals([]conditionalReplacement{*replacement})
	}
}

func (preparation *sourcePreparation) inputDiagnostics(diagnostics scanner.ErrorList) scanner.ErrorList {
	original := preparation.original
	for _, diagnostic := range diagnostics {
		diagnostic.Pos = original.Position(original.Pos(preparation.source.offsets[diagnostic.Pos.Offset]))
	}
	return diagnostics
}

func reserveIdentifiers(original *token.File, input []byte, used map[string]bool) {
	var scan scanner.Scanner
	scan.Init(original, input, nil, scanner.ScanComments)
	for {
		_, tok, literal := scan.Scan()
		if tok == token.IDENT {
			used[literal] = true
		}
		if tok == token.EOF {
			break
		}
	}
}

func (result *Parsed) recordComprehensions(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				if record := result.Comprehensions[id.Name]; record != nil {
					record.Type = call.Args[0]
					record.Body = call.Args[1].(*ast.FuncLit).Body
					record.Ranges = comprehensionRanges(record.Body)
				}
			}
		}
		return true
	})
}
