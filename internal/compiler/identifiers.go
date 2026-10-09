package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	pathpkg "path"
	"strconv"
	"strings"

	"tgo/pkg/syntax"
)

type generatedImport struct {
	name        string
	importPath  string
	defaultName string
	add         bool
	dot         bool
	rename      *syntax.Identifier
}

type enumImports struct {
	json     generatedImport
	jsonV2   generatedImport
	jsonText generatedImport
	strings  generatedImport
	fmt      generatedImport
}

// freshIdentifier returns the readable base or its first free suffix.
func freshIdentifier(base string, used map[string]bool) string {
	name := base
	for suffix := 1; used[name]; suffix++ {
		name = base + "_" + strconv.Itoa(suffix)
	}
	used[name] = true
	return name
}

func freshASTIdentifier(root ast.Node, base string) string {
	used := make(map[string]bool)
	ast.Inspect(root, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			used[identifier.Name] = true
		}
		return true
	})
	return freshIdentifier(base, used)
}

func reserveImportNames(tree *syntax.File, used map[string]bool) {
	for _, specification := range tree.Imports {
		if specification.Name != nil {
			continue
		}
		importPath, err := strconv.Unquote(specification.Path.Value)
		if err == nil {
			used[defaultImportName(importPath)] = true
		}
	}
}

func planEnumImports(tree *syntax.File, used map[string]bool) enumImports {
	return enumImports{
		json: planGeneratedImport(
			tree, "encoding/json", "json", "json", used,
		),
		jsonV2: planGeneratedImport(
			tree, "encoding/json/v2", "jsonv2", "json", used,
		),
		jsonText: planGeneratedImport(
			tree, "encoding/json/jsontext", "jsontext", "jsontext", used,
		),
		strings: planGeneratedImport(tree, "strings", "strings", "strings", used),
		fmt:     planGeneratedImport(tree, "fmt", "fmt", "fmt", used),
	}
}

func (i enumImports) addEdits(
	edits []edit,
	file *token.File,
	packageEnd token.Pos,
	use enumJSONUse,
) []edit {
	if !use.enum {
		return edits
	}
	var declarations strings.Builder
	for _, item := range []generatedImport{i.json, i.jsonV2, i.jsonText, i.fmt} {
		if item.add {
			declarations.WriteString(item.line())
		}
		if change, ok := item.renameEdit(file); ok {
			edits = append(edits, change)
		}
	}
	if use.foldedAdjacent {
		if i.strings.add {
			declarations.WriteString(i.strings.line())
		}
		if change, ok := i.strings.renameEdit(file); ok {
			edits = append(edits, change)
		}
	}
	if declarations.Len() > 0 {
		offset := file.Offset(packageEnd)
		edits = append(edits, edit{
			start: offset, end: offset, text: "\n" + declarations.String(),
		})
	}
	replaceDotQualifiers(
		edits, i.json, i.jsonV2, i.jsonText, i.strings, i.fmt,
	)
	return edits
}

func planGeneratedImport(
	tree *syntax.File,
	importPath string,
	preferred string,
	defaultName string,
	used map[string]bool,
) generatedImport {
	result := generatedImport{importPath: importPath, defaultName: defaultName}
	for _, specification := range tree.Imports {
		path, err := strconv.Unquote(specification.Path.Value)
		if err != nil || path != importPath {
			continue
		}
		if specification.Name == nil {
			result.name = defaultName
			return result
		}
		switch specification.Name.Name {
		case "_":
			result.name = freshIdentifier(preferred, used)
			result.rename = specification.Name
		case ".":
			result.name = freshIdentifier("tgoDotImport", used)
			result.dot = true
		default:
			result.name = specification.Name.Name
		}
		return result
	}
	result.name = freshIdentifier(preferred, used)
	result.add = true
	return result
}

func defaultImportName(importPath string) string {
	if importPath == "encoding/json/v2" {
		return "json"
	}
	return pathpkg.Base(importPath)
}

func (i generatedImport) line() string {
	if i.name == i.defaultName {
		return fmt.Sprintf("import %q\n", i.importPath)
	}
	return fmt.Sprintf("import %s %q\n", i.name, i.importPath)
}

func (i generatedImport) renameEdit(file *token.File) (edit, bool) {
	if i.rename == nil {
		return edit{}, false
	}
	name := i.name
	if name == i.defaultName {
		name = ""
	}
	return edit{
		start: file.Offset(i.rename.Start), end: file.Offset(i.rename.Stop),
		text: name,
	}, true
}

func replaceDotQualifiers(edits []edit, imports ...generatedImport) {
	for _, item := range imports {
		if !item.dot {
			continue
		}
		for index := range edits {
			edits[index].text = strings.ReplaceAll(
				edits[index].text, item.name+".", "",
			)
		}
	}
}
