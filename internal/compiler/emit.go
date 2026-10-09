package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

// enumGo emits the tagged Go representation for one tgo enum.
func enumGo(sourceName string, declaration *model, fmtPackage string) string {
	var output strings.Builder
	name := declaration.Name
	fmt.Fprintf(&output, "// %s requires a variant constructor. Its zero value is invalid.\n", name)
	output.WriteString("// Shared data keeps Go aliases. Callers must keep model values valid.\n")
	tagType := enumTagType(len(declaration.Variants))
	fmt.Fprintf(&output, "type %sTag %s\n", name, tagType)
	fmt.Fprintf(&output, "const (\n")
	for index, variant := range declaration.Variants {
		if index == 0 {
			fmt.Fprintf(&output, "%sTag%s %sTag = iota + 1\n", name, variant.Name, name)
			continue
		}
		fmt.Fprintf(&output, "%sTag%s\n", name, variant.Name)
	}
	output.WriteString(")\n")
	fmt.Fprintf(&output, "type %s struct {\n tgoTag %sTag\n", name, name)
	boxed := false
	for _, variant := range declaration.Variants {
		if variant.Boxed {
			boxed = true
			continue
		}
		if len(variant.Fields) == 0 {
			continue
		}
		fmt.Fprintf(&output, "tgo%s %s%s\n", variant.Name, name, variant.Name)
	}
	if boxed {
		output.WriteString("tgoPayload interface{}\n")
	}
	output.WriteString("}\n")
	output.WriteString("// Tag returns the active tag.\n")
	fmt.Fprintf(&output, "func (v %s) Tag() %sTag { return v.tgoTag }\n", name, name)
	output.WriteString("// UnknownTag describes an invalid tag.\n")
	fmt.Fprintf(&output,
		"func (v %s) UnknownTag() string { return %s.Sprintf(%q, v.tgoTag) }\n",
		name, fmtPackage,
		name+": unknown tag %d — tgolint proves every tag has a case, so this is unreachable")
	for _, variant := range declaration.Variants {
		emitVariant(&output, sourceName, name, variant)
	}
	output.WriteByte('\n')
	return output.String()
}

// emitVariant emits one payload type, constructor, and payload accessor.
func emitVariant(
	output *strings.Builder,
	sourceName string,
	enum string,
	variant variant,
) {
	payload := enum + variant.Name
	accessor := variant.Name + "Payload"
	tagName := enum + "Tag" + variant.Name
	fmt.Fprintf(output, "// %s is the %s payload.\n", payload, variant.Name)
	if len(variant.Fields) == 0 {
		fmt.Fprintf(output, "type %s struct{}\n", payload)
	} else {
		fmt.Fprintf(
			output,
			"type %s struct {\n%s}\n",
			payload,
			fieldDecls(sourceName, variant.Fields),
		)
	}
	fmt.Fprintf(output, "// %s constructs %s. Model fields must be valid.\n", enum, enum)
	output.WriteString("// Shared fields keep their aliases and caller duties.\n")
	fmt.Fprintf(output, "func (value %s) %s() %s {\n", payload, enum, enum)
	switch {
	case len(variant.Fields) == 0:
		fmt.Fprintf(output, "return %s{tgoTag: %s}\n}\n", enum, tagName)
	case variant.Boxed:
		fmt.Fprintf(output, "return %s{tgoTag: %s, tgoPayload: value}\n}\n", enum, tagName)
	default:
		fmt.Fprintf(
			output,
			"return %s{tgoTag: %s, tgo%s: value}\n}\n",
			enum,
			tagName,
			variant.Name,
		)
	}
	fmt.Fprintf(
		output,
		"// %s requires %s. No tag check.\n",
		accessor,
		variant.Name,
	)
	switch {
	case len(variant.Fields) == 0:
		fmt.Fprintf(output, "func (%s) %s() %s { return %s{} }\n", enum, accessor, payload, payload)
	case variant.Boxed:
		fmt.Fprintf(
			output,
			"func (v %s) %s() %s { return v.tgoPayload.(%s) }\n",
			enum,
			accessor,
			payload,
			payload,
		)
	default:
		fmt.Fprintf(
			output,
			"func (v %s) %s() %s { return v.tgo%s }\n",
			enum,
			accessor,
			payload,
			variant.Name,
		)
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
