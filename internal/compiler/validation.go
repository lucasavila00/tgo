package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// checkValidationNameCollisions rejects source declarations that use validator names.
func (p *packageUnit) checkValidationNameCollisions() {
	for _, source := range p.Sources {
		for _, declaration := range source.Models {
			names := map[string]bool{
				"Validate" + declaration.Name:                true,
				"tgo" + declaration.Name + "ValidationError": true,
			}
			for _, variant := range declaration.Variants {
				names["tgoReconstruct"+declaration.Name+variant.Name] = true
			}
			for name := range names {
				p.rejectDuplicateValidationName(name, declaration)
			}
			p.rejectDuplicateValidationMethod(declaration)
		}
	}
}

func (p *packageUnit) rejectDuplicateValidationName(name string, model *model) {
	var declarations []ast.Node
	for _, file := range p.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil && declaration.Name.Name == name {
					declarations = append(declarations, declaration.Name)
				}
			case *ast.GenDecl:
				for _, item := range declaration.Specs {
					typ, ok := item.(*ast.TypeSpec)
					if ok && typ.Name.Name == name {
						declarations = append(declarations, typ.Name)
					}
				}
			}
		}
	}
	if len(declarations) > 1 {
		p.failAt(collisionPosition(p.fs, declarations, model),
			"generated validation name %s is reserved", name)
	}
}

func (p *packageUnit) rejectDuplicateValidationMethod(model *model) {
	var declarations []ast.Node
	for _, file := range p.Files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != "TgoReconstruct" {
				continue
			}
			receiver, ok := receiverName(function)
			if ok && receiver == model.Name {
				declarations = append(declarations, function.Name)
			}
		}
	}
	if len(declarations) > 1 {
		p.failAt(collisionPosition(p.fs, declarations, model),
			"generated validation method %s.TgoReconstruct is reserved", model.Name)
	}
}

func collisionPosition(files *token.FileSet, declarations []ast.Node, model *model) token.Pos {
	for _, declaration := range declarations {
		position := files.Position(declaration.Pos())
		if position.Line != model.Line {
			return declaration.Pos()
		}
	}
	return declarations[len(declarations)-1].Pos()
}

// structValidationGo emits explicit reconstruction for one tgo struct.
func structValidationGo(declaration *model, runtimeAlias string) string {
	var output strings.Builder
	emitValidationHeader(&output, declaration.Name, runtimeAlias)
	fmt.Fprintf(
		&output,
		"func (v %s) TgoReconstruct(context *%s.Context) (any, error) {\n",
		declaration.Name,
		runtimeAlias,
	)
	emitFieldReconstruction(&output, "v", declaration.Fields, runtimeAlias)
	output.WriteString("}\n")
	return output.String()
}

// checkedValidationGo emits predicate rechecking for one checked value.
func checkedValidationGo(declaration *model, runtimeAlias string) string {
	var output strings.Builder
	emitValidationHeader(&output, declaration.Name, runtimeAlias)
	fmt.Fprintf(
		&output,
		"func (v %s) TgoReconstruct(context *%s.Context) (any, error) {\n",
		declaration.Name,
		runtimeAlias,
	)
	fmt.Fprintf(
		&output,
		"rebuilt, err := %s.RebuildAs(v.value, context)\n",
		runtimeAlias,
	)
	output.WriteString("if err != nil { return nil, err }\n")
	fmt.Fprintf(&output, "return New%s(rebuilt)\n}\n", declaration.Name)
	return output.String()
}

// enumValidationGo emits tag checks and active-payload reconstruction.
func enumValidationGo(declaration *model, runtimeAlias string) string {
	var output strings.Builder
	emitValidationHeader(&output, declaration.Name, runtimeAlias)
	for _, variant := range declaration.Variants {
		payload := declaration.Name + variant.Name
		fmt.Fprintf(
			&output,
			"func tgoReconstruct%s(value %s, context *%s.Context) (%s, error) {\n",
			payload,
			payload,
			runtimeAlias,
			payload,
		)
		emitFieldReconstruction(&output, "value", variant.Fields, runtimeAlias)
		output.WriteString("}\n")
	}
	fmt.Fprintf(
		&output,
		"func (v %s) TgoReconstruct(context *%s.Context) (any, error) {\n",
		declaration.Name,
		runtimeAlias,
	)
	output.WriteString("switch v.tgoTag {\n")
	for index, variant := range declaration.Variants {
		payload := declaration.Name + variant.Name
		fmt.Fprintf(&output, "case %d:\n", index+1)
		fmt.Fprintf(
			&output,
			"rebuilt, err := tgoReconstruct%s(v.tgo%s, context)\n",
			payload,
			variant.Name,
		)
		output.WriteString("if err != nil { return nil, err }\n")
		fmt.Fprintf(&output, "return New%s(rebuilt), nil\n", payload)
	}
	fmt.Fprintf(
		&output,
		"default: return nil, tgo%sValidationError(\"invalid %s tag\")\n",
		declaration.Name,
		declaration.Name,
	)
	output.WriteString("}\n}\n")
	return output.String()
}

func emitValidationHeader(output *strings.Builder, name, runtimeAlias string) {
	fmt.Fprintf(output, "type tgo%sValidationError string\n", name)
	fmt.Fprintf(
		output,
		"func (e tgo%sValidationError) Error() string { return string(e) }\n",
		name,
	)
	fmt.Fprintf(
		output,
		"// Validate%s checks and reconstructs one foreign %s graph.\n",
		name,
		name,
	)
	fmt.Fprintf(output, "func Validate%s(value %s) (%s, error) {\n", name, name, name)
	fmt.Fprintf(
		output,
		"return %s.RebuildAs(value, %s.NewContext())\n}\n",
		runtimeAlias,
		runtimeAlias,
	)
}

func emitFieldReconstruction(
	output *strings.Builder,
	valueName string,
	fields []field,
	runtimeAlias string,
) {
	fmt.Fprintf(output, "result := %s\n", valueName)
	for index, item := range fields {
		selector := item.Name
		if selector == "" {
			selector = embeddedFieldName(item.Type)
		}
		fmt.Fprintf(
			output,
			"field%d, err := %s.RebuildAs(%s.%s, context)\n",
			index,
			runtimeAlias,
			valueName,
			selector,
		)
		output.WriteString("if err != nil { return result, err }\n")
		fmt.Fprintf(output, "result.%s = field%d\n", selector, index)
	}
	output.WriteString("return result, nil\n")
}

func embeddedFieldName(text string) string {
	expression, err := parser.ParseExpr(text)
	if err != nil {
		return text
	}
	for {
		switch node := expression.(type) {
		case *ast.StarExpr:
			expression = node.X
		case *ast.IndexExpr:
			expression = node.X
		case *ast.IndexListExpr:
			expression = node.X
		case *ast.SelectorExpr:
			return node.Sel.Name
		case *ast.Ident:
			return node.Name
		default:
			return text
		}
	}
}
