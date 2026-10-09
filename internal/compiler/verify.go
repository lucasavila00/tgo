package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
)

// VerifyGeneratedModels verifies all model declarations that the current emitter owns.
func VerifyGeneratedModels(
	sourceName string,
	sourceData, generatedBody []byte,
	pkg *types.Package,
) error {
	parsed, err := parseSource(token.NewFileSet(), sourceName, sourceData)
	if err != nil {
		return fmt.Errorf("parse tgo source: %w", err)
	}
	if !parsed.Lowered {
		return verifyOrdinaryGoOutput(sourceData, generatedBody)
	}
	for _, declaration := range parsed.Models {
		if !declaration.Enum {
			continue
		}
		for _, variant := range declaration.Variants {
			name := declaration.Name + variant.Name
			if pkg.Scope().Lookup(name) == nil {
				return fmt.Errorf(
					"generated declaration type %s does not match the current emitter",
					name,
				)
			}
		}
		enumLayout(declaration, pkg)
	}
	expectedFiles := token.NewFileSet()
	expected, err := parser.ParseFile(
		expectedFiles,
		sourceName+".expected.go",
		"package verify\n"+expectedModelText(sourceName, parsed),
		parser.SkipObjectResolution,
	)
	if err != nil {
		return fmt.Errorf("parse expected Go declarations: %w", err)
	}
	generatedFiles := token.NewFileSet()
	generated, err := parser.ParseFile(
		generatedFiles,
		sourceName+".generated.go",
		generatedBody,
		parser.SkipObjectResolution,
	)
	if err != nil {
		return fmt.Errorf("parse generated Go declarations: %w", err)
	}
	want, err := indexedDeclarations(expectedFiles, expected.Decls)
	if err != nil {
		return err
	}
	got, err := indexedDeclarations(generatedFiles, generated.Decls)
	if err != nil {
		return err
	}
	for key, declaration := range want {
		actual, ok := got[key]
		if !ok || !bytes.Equal(declaration, actual) {
			return fmt.Errorf("generated declaration %s does not match the current emitter", key)
		}
	}
	return nil
}

func verifyOrdinaryGoOutput(sourceData, generatedBody []byte) error {
	if !bytes.Equal(sourceData, generatedBody) {
		return fmt.Errorf("ordinary Go output does not match source")
	}
	return nil
}

func expectedModelText(sourceName string, parsed *source) string {
	var output bytes.Buffer
	for _, declaration := range parsed.Models {
		switch {
		case declaration.Enum:
			output.WriteString(enumGo(sourceName, declaration, parsed.FmtPackage))
			output.WriteString(enumJSONGo(
				declaration, parsed.JSONPackage, parsed.JSONV2Package,
				parsed.JSONTextPackage, parsed.StringsPackage, parsed.FmtPackage,
				parsed.ExternalJSONTo, parsed.AdjacentJSONTo,
			))
		case declaration.Predicate != "":
			output.WriteString(checkedGo(sourceName, declaration))
		default:
			fmt.Fprintf(
				&output,
				"type %s struct {\n%s}\n",
				declaration.Name,
				fieldDecls(sourceName, declaration.Fields),
			)
		}
	}
	return output.String()
}

func indexedDeclarations(files *token.FileSet, declarations []ast.Decl) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, declaration := range declarations {
		key := declarationKey(declaration)
		if key == "" {
			continue
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("generated declaration %s occurs more than once", key)
		}
		var normalized bytes.Buffer
		if err := format.Node(&normalized, files, declaration); err != nil {
			return nil, fmt.Errorf("normalize generated declaration %s: %w", key, err)
		}
		result[key] = normalized.Bytes()
	}
	return result, nil
}

func declarationKey(declaration ast.Decl) string {
	switch node := declaration.(type) {
	case *ast.GenDecl:
		if node.Tok != token.TYPE || len(node.Specs) != 1 {
			return ""
		}
		specification, ok := node.Specs[0].(*ast.TypeSpec)
		if !ok {
			return ""
		}
		return "type " + specification.Name.Name
	case *ast.FuncDecl:
		receiver := ""
		if node.Recv != nil && len(node.Recv.List) == 1 {
			receiver = receiverTypeName(node.Recv.List[0].Type) + "."
		}
		return "func " + receiver + node.Name.Name
	default:
		return ""
	}
}

func receiverTypeName(expression ast.Expr) string {
	for {
		switch node := expression.(type) {
		case *ast.Ident:
			return node.Name
		case *ast.StarExpr:
			expression = node.X
		case *ast.IndexExpr:
			expression = node.X
		case *ast.IndexListExpr:
			expression = node.X
		default:
			return ""
		}
	}
}
