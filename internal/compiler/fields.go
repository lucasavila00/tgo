package compiler

import "strings"

func fieldDecls(sourceName string, fields []field) string {
	var text strings.Builder
	for _, field := range fields {
		prefix := ""
		if field.Name != "" {
			prefix = field.Name + " "
		}
		column := field.TypeColumn - len(prefix)
		text.WriteString(inlineLineDirective(sourceName, field.TypeLine, column))
		text.WriteString(prefix)
		text.WriteString(field.Type)
		if field.Tag != "" {
			text.WriteByte(' ')
			text.WriteString(field.Tag)
		}
		text.WriteByte('\n')
	}
	return text.String()
}
