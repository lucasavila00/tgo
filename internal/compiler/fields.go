package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

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

func (p *sourceParser) field(limit int) ([]field, error) {
	start := p.cursor
	assignment := -1
	for p.cursor < limit && !p.has(0, token.SEMICOLON) {
		if p.has(0, token.ASSIGN) {
			assignment = p.cursor
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
	end := p.cursor
	if assignment >= 0 {
		end = assignment
	}
	declaration := p.text(start, end)
	parsed, err := parser.ParseExpr("struct {" + declaration + "}")
	if err != nil {
		return nil, err
	}
	parsedField := parsed.(*ast.StructType).Fields.List[0]
	if len(parsedField.Names) == 0 {
		if assignment >= 0 {
			return nil, fmt.Errorf("embedded fields cannot have defaults")
		}
		return []field{{Type: declaration}}, nil
	}
	firstTypeToken := start + len(parsedField.Names)*2 - 1
	typeText := p.text(firstTypeToken, end)
	defaultText := ""
	if assignment >= 0 {
		defaultText = p.text(assignment+1, p.cursor)
		if defaultText == "" {
			return nil, fmt.Errorf("missing field default")
		}
	}
	fields := make([]field, 0, len(parsedField.Names))
	for _, name := range parsedField.Names {
		fields = append(fields, field{Name: name.Name, Type: typeText, Default: defaultText})
	}
	return fields, nil
}

func fieldDecls(fields []field) string {
	var text strings.Builder
	for _, field := range fields {
		text.WriteString(field.Name)
		text.WriteByte(' ')
		text.WriteString(field.Type)
		text.WriteByte('\n')
	}
	return text.String()
}
