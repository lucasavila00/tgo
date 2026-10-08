package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

// enumGo emits the tagged Go representation for one tgo enum.
func enumGo(sourceName string, declaration *model) string {
	var output strings.Builder
	name := declaration.Name
	fmt.Fprintf(&output, "// %s requires a variant constructor. Its zero value is invalid.\n", name)
	output.WriteString("// Shared data keeps Go aliases. Callers must keep model values valid.\n")
	tagType := enumTagType(len(declaration.Variants))
	fmt.Fprintf(&output, "type %s struct {\n tgoTag %s\n", name, tagType)
	for _, variant := range declaration.Variants {
		if len(variant.Fields) == 0 {
			continue
		}
		fmt.Fprintf(&output, "tgo%s %s%s\n", variant.Name, name, variant.Name)
	}
	output.WriteString("}\n")
	output.WriteString("// TgoTag returns the tag. Use only on a constructed value.\n")
	fmt.Fprintf(&output, "func (v %s) TgoTag() %s { return v.tgoTag }\n", name, tagType)
	for index, variant := range declaration.Variants {
		emitVariant(&output, sourceName, name, variant, index+1)
	}
	return output.String()
}

// emitVariant emits one payload type, constructor, and payload accessor.
func emitVariant(
	output *strings.Builder,
	sourceName string,
	enum string,
	variant variant,
	tag int,
) {
	payload := enum + variant.Name
	constructor := "New" + payload
	accessor := "Tgo" + variant.Name
	fmt.Fprintf(output, "// %s holds the variant fields. Supply every field.\n", payload)
	fmt.Fprintf(
		output,
		"type %s struct {\n%s}\n",
		payload,
		fieldDecls(sourceName, variant.Fields),
	)
	fmt.Fprintf(output, "// %s constructs %s. Model fields must be valid.\n", constructor, enum)
	output.WriteString("// Shared fields keep their aliases and caller duties.\n")
	parameter := "value"
	if len(variant.Fields) == 0 {
		parameter = "_"
	}
	fmt.Fprintf(output, "func %s(%s %s) %s {\n", constructor, parameter, payload, enum)
	if len(variant.Fields) == 0 {
		fmt.Fprintf(output, "return %s{tgoTag: %d}\n}\n", enum, tag)
	} else {
		fmt.Fprintf(
			output,
			"return %s{tgoTag: %d, tgo%s: value}\n}\n",
			enum,
			tag,
			variant.Name,
		)
	}
	fmt.Fprintf(output, "// %s requires %s. No tag check.\n", accessor, variant.Name)
	fmt.Fprintf(output, "func (v %s) %s() %s {\n", enum, accessor, payload)
	if len(variant.Fields) == 0 {
		fmt.Fprintf(output, "return %s{}\n}\n", payload)
	} else {
		fmt.Fprintf(output, "return v.tgo%s\n}\n", variant.Name)
	}
}

// checkedGo emits a checked wrapper, constructor, error, and value accessor.
func checkedGo(sourceName string, declaration *model) string {
	var output strings.Builder
	name := declaration.Name
	fmt.Fprintf(&output, "// %s requires New%s success. Zero is invalid.\n", name, name)
	output.WriteString("// Shared data keeps Go aliases. Callers must keep the rule.\n")
	baseColumn := declaration.BaseColumn - len("value ")
	directive := inlineLineDirective(sourceName, declaration.BaseLine, baseColumn)
	fmt.Fprintf(&output, "type %s struct { %svalue %s }\n", name, directive, declaration.Base)
	fmt.Fprintf(&output, "type tgo%sError struct {}\n", name)
	message := strconv.Quote("invalid " + name)
	fmt.Fprintf(&output, "func (tgo%sError) Error() string { return %s }\n", name, message)
	fmt.Fprintf(&output, "// New%s checks the rule. Check the error before use.\n", name)
	fmt.Fprintf(&output, "func New%s(value %s) (%s, error) {\n", name, declaration.Base, name)
	output.WriteString("if !(\n")
	output.WriteString(lineDirective(
		sourceName,
		declaration.PredicateLine,
		declaration.PredicateColumn,
	))
	fmt.Fprintf(&output, "%s) {\n", declaration.Predicate)
	fmt.Fprintf(&output, "return %s{}, tgo%sError{}\n}\n", name, name)
	fmt.Fprintf(&output, "return %s{value: value}, nil\n}\n", name)
	output.WriteString("// Value requires construction success. Shared data keeps its aliases.\n")
	fmt.Fprintf(&output, "func (v %s) Value() %s { return v.value }\n", name, declaration.Base)
	return output.String()
}

// enumTagType selects the smallest tag type that can name every variant.
func enumTagType(variants int) string {
	switch {
	case variants < 1<<8:
		return "uint8"
	case variants < 1<<16:
		return "uint16"
	default:
		return "uint32"
	}
}
