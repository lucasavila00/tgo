package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// fields parses all fields inside one struct body.
func (p *sourceParser) fields() ([]field, error) {
	end, err := closeToken(p.tokens, p.cursor)
	if err != nil {
		return nil, err
	}
	p.cursor++
	var fields []field
	for p.cursor < end {
		if p.has(0, token.SEMICOLON) {
			p.cursor++
			continue
		}
		items, err := p.field(end)
		if err != nil {
			return nil, err
		}
		fields = append(fields, items...)
	}
	p.cursor = end + 1
	return fields, nil
}

// field parses one field group and its optional default.
func (p *sourceParser) field(limit int) ([]field, error) {
	span, err := p.scanField(limit)
	if err != nil {
		return nil, err
	}

	declaration := p.text(span.start, span.declarationEnd())
	parsed, err := parseField(declaration)
	if err != nil {
		return nil, p.errorAt(span.start, "%s", err)
	}

	defaultValue, err := p.fieldDefault(span)
	if err != nil {
		return nil, err
	}
	if len(parsed.Names) == 0 {
		if defaultValue != "" {
			return nil, p.errorAt(span.start, "embedded fields cannot have defaults")
		}
		return []field{{Type: fieldNodeText(declaration, parsed.Type)}}, nil
	}

	typeText := fieldNodeText(declaration, parsed.Type)
	tag := ""
	if parsed.Tag != nil {
		tag = parsed.Tag.Value
	}
	fields := make([]field, 0, len(parsed.Names))
	for _, name := range parsed.Names {
		fields = append(fields, field{
			Name:    name.Name,
			Type:    typeText,
			Tag:     tag,
			Default: defaultValue,
		})
	}

	return fields, nil
}

type fieldSpan struct {
	start      int
	end        int
	assignment int
}

// declarationEnd excludes a default expression from the Go field syntax.
func (s fieldSpan) declarationEnd() int {
	if s.assignment >= 0 {
		return s.assignment
	}
	return s.end
}

// scanField finds one top-level field and its default assignment.
func (p *sourceParser) scanField(limit int) (fieldSpan, error) {
	span := fieldSpan{start: p.cursor, assignment: -1}
	for p.cursor < limit && !p.has(0, token.SEMICOLON) {
		if p.has(0, token.ASSIGN) {
			span.assignment = p.cursor
		}
		if err := p.advance(); err != nil {
			return fieldSpan{}, err
		}
	}
	span.end = p.cursor
	return span, nil
}

// parseField uses the Go parser to split field names, type, and tag.
func parseField(declaration string) (*ast.Field, error) {
	const prefix = "struct {"
	expression, err := parser.ParseExpr(prefix + declaration + "}")
	if err != nil {
		return nil, err
	}
	structure := expression.(*ast.StructType)
	if len(structure.Fields.List) != 1 {
		return nil, fmt.Errorf("field declaration must define one field group")
	}
	return structure.Fields.List[0], nil
}

// fieldDefault returns the source text for a declared default.
func (p *sourceParser) fieldDefault(span fieldSpan) (string, error) {
	if span.assignment < 0 {
		return "", nil
	}
	value := p.text(span.assignment+1, span.end)
	if value == "" {
		return "", p.errorAt(span.assignment, "missing field default")
	}
	return value, nil
}

// fieldNodeText extracts an AST node from the original field text.
func fieldNodeText(declaration string, node ast.Node) string {
	const prefixLength = len("struct {")
	start := int(node.Pos()) - 1 - prefixLength
	end := int(node.End()) - 1 - prefixLength
	return declaration[start:end]
}

// fieldDecls emits Go field declarations without tgo defaults.
func fieldDecls(fields []field) string {
	var text strings.Builder
	for _, field := range fields {
		if field.Name != "" {
			text.WriteString(field.Name)
			text.WriteByte(' ')
		}
		text.WriteString(field.Type)
		if field.Tag != "" {
			text.WriteByte(' ')
			text.WriteString(field.Tag)
		}
		text.WriteByte('\n')
	}
	return text.String()
}
