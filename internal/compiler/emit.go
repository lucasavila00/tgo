package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"strings"
	"unicode"
)

// checkedStructGo emits the checked value and its Go construction ABI.
func checkedStructGo(sourceName string, declaration *model) string {
	var output strings.Builder
	parameters := checkedParameterNames(declaration.Fields)
	fmt.Fprintf(
		&output,
		"type %s struct {\n%s}\n",
		declaration.Name,
		fieldDecls(sourceName, declaration.Fields),
	)
	fmt.Fprintf(&output, "type %s struct {\n", checkedCarrierName(declaration.Name))
	for index, field := range declaration.Fields {
		fmt.Fprintf(&output, "%s %s\n", checkedCarrierFieldName(field, index), field.Type)
	}
	output.WriteString("}\n")
	fmt.Fprintf(
		&output,
		"// New%s constructs and checks %s.\nfunc New%s(",
		declaration.Name,
		declaration.Name,
		declaration.Name,
	)
	for index, field := range declaration.Fields {
		if index != 0 {
			output.WriteString(", ")
		}
		fmt.Fprintf(&output, "%s %s", parameters[index], field.Type)
	}
	fmt.Fprintf(&output, ") (%s, error) {\nreturn %s{", declaration.Name, declaration.Name)
	for index := range declaration.Fields {
		if index != 0 {
			output.WriteString(", ")
		}
		fmt.Fprint(&output, parameters[index])
	}
	output.WriteString("}.check()\n}\n")
	return output.String()
}

func checkedCarrierName(name string) string { return "Tgo" + name + "Input" }

func checkedCarrierFieldName(value field, index int) string {
	if value.Name == "" || value.Name == "_" {
		return fmt.Sprintf("Field%d", index)
	}
	runes := []rune(value.Name)
	runes[0] = unicode.ToUpper(runes[0])
	return "Field" + string(runes)
}

func checkedParameterNames(fields []field) []string {
	names := make([]string, len(fields))
	used := make(map[string]bool)
	for index, field := range fields {
		if field.Name != "" && field.Name != "_" {
			names[index] = field.Name
			used[field.Name] = true
		}
	}
	for index := range fields {
		if names[index] != "" {
			continue
		}
		base := fmt.Sprintf("tgoField%d", index)
		name := base
		for suffix := 1; used[name]; suffix++ {
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		names[index] = name
		used[name] = true
	}
	return names
}

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
	if payloadFreeEnum(declaration) {
		emitEnumGob(&output, declaration, fmtPackage)
	}
	for _, variant := range declaration.Variants {
		emitVariant(&output, sourceName, name, variant)
	}
	output.WriteByte('\n')
	return output.String()
}

// emitEnumGob emits a stable gob representation for a payload-free enum.
func emitEnumGob(output *strings.Builder, declaration *model, fmtPackage string) {
	name := declaration.Name
	first := name + "Tag" + declaration.Variants[0].Name
	last := name + "Tag" + declaration.Variants[len(declaration.Variants)-1].Name
	output.WriteString("// GobEncode returns the stable four-byte enum tag.\n")
	fmt.Fprintf(output, "func (v %s) GobEncode() ([]byte, error) {\n", name)
	fmt.Fprintf(output, "if v.tgoTag < %s || v.tgoTag > %s {\n", first, last)
	fmt.Fprintf(
		output,
		"return nil, %s.Errorf(%q, v.tgoTag)\n}\n",
		fmtPackage,
		name+": cannot gob encode invalid tag %d",
	)
	output.WriteString("tag := uint32(v.tgoTag)\n")
	output.WriteString(
		"return []byte{byte(tag >> 24), byte(tag >> 16), " +
			"byte(tag >> 8), byte(tag)}, nil\n}\n",
	)
	output.WriteString("// GobDecode replaces the value with a valid four-byte enum tag.\n")
	fmt.Fprintf(output, "func (v *%s) GobDecode(data []byte) error {\n", name)
	fmt.Fprintf(
		output,
		"if len(data) != 4 { return %s.Errorf(%q, len(data)) }\n",
		fmtPackage,
		name+": invalid gob data length %d",
	)
	fmt.Fprintf(
		output,
		"number := uint32(data[0]) << 24 | uint32(data[1]) << 16 | "+
			"uint32(data[2]) << 8 | uint32(data[3])\n"+
			"tag := %sTag(number)\n",
		name,
	)
	fmt.Fprintf(
		output,
		"if uint32(tag) != number || tag < %s || tag > %s {\n",
		first,
		last,
	)
	fmt.Fprintf(
		output,
		"return %s.Errorf(%q, number)\n}\n",
		fmtPackage,
		name+": cannot gob decode unknown tag %d",
	)
	output.WriteString("switch tag {\n")
	for _, variant := range declaration.Variants {
		fmt.Fprintf(
			output,
			"case %sTag%s: *v = %s()\n",
			name,
			variant.Name,
			enumConstructorName(name, variant.Name),
		)
	}
	output.WriteString("}\nreturn nil\n}\n")
}

// payloadFreeEnum reports whether every variant has no payload field.
func payloadFreeEnum(declaration *model) bool {
	if len(declaration.Variants) == 0 {
		return false
	}
	for _, variant := range declaration.Variants {
		if len(variant.Fields) != 0 {
			return false
		}
	}
	return true
}

// emitVariant emits one payload type, Go constructor, and payload accessor.
func emitVariant(
	output *strings.Builder,
	sourceName string,
	enum string,
	variant variant,
) {
	payload := enum + variant.Name
	constructor := enumConstructorName(enum, variant.Name)
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
	if len(variant.Fields) > 0 {
		fmt.Fprintf(output, "type %s struct {\n", enumCarrierName(enum, variant.Name))
		for index, fieldName := range enumCarrierFieldNames(variant.Fields) {
			fmt.Fprintf(output, "%s %s\n", fieldName, variant.Fields[index].Type)
		}
		output.WriteString("}\n")
	}
	parameters := enumParameterNames(variant.Fields, enum, payload, tagName)
	payloadValue := enumPayloadLocalName(parameters)
	fmt.Fprintf(output, "// %s constructs %s. Model fields must be valid.\n", constructor, enum)
	output.WriteString("// Shared fields keep their aliases and caller duties.\n")
	fmt.Fprintf(output, "func %s(", constructor)
	for index, field := range variant.Fields {
		if index != 0 {
			output.WriteString(", ")
		}
		fmt.Fprintf(output, "%s %s", parameters[index], field.Type)
	}
	fmt.Fprintf(output, ") %s {\n", enum)
	if len(variant.Fields) > 0 {
		fmt.Fprintf(output, "%s := %s{", payloadValue, payload)
		for index, parameter := range parameters {
			if index != 0 {
				output.WriteString(", ")
			}
			output.WriteString(parameter)
		}
		output.WriteString("}\n")
	}
	switch {
	case len(variant.Fields) == 0:
		fmt.Fprintf(output, "return %s{tgoTag: %s}\n}\n", enum, tagName)
	case variant.Boxed:
		fmt.Fprintf(
			output,
			"return %s{tgoTag: %s, tgoPayload: %s}\n}\n",
			enum,
			tagName,
			payloadValue,
		)
	default:
		fmt.Fprintf(
			output,
			"return %s{tgoTag: %s, tgo%s: %s}\n}\n",
			enum,
			tagName,
			variant.Name,
			payloadValue,
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

func enumConstructorName(enum string, variant string) string {
	return "New" + enum + variant
}

func enumCarrierName(enum string, variant string) string {
	return "Tgo" + enum + variant + "Input"
}

func enumParameterNames(fields []field, reserved ...string) []string {
	names := make([]string, len(fields))
	used := make(map[string]bool)
	for _, name := range reserved {
		used[name] = true
	}
	for index, value := range fields {
		if value.Name != "" && value.Name != "_" && !used[value.Name] {
			names[index] = value.Name
			used[value.Name] = true
			continue
		}
		base := fmt.Sprintf("tgoField%d", index)
		name := base
		for suffix := 1; used[name]; suffix++ {
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		names[index] = name
		used[name] = true
	}
	return names
}

func enumPayloadLocalName(parameters []string) string {
	used := make(map[string]bool)
	for _, name := range parameters {
		used[name] = true
	}
	base := "tgoValue"
	name := base
	for suffix := 1; used[name]; suffix++ {
		name = fmt.Sprintf("%s_%d", base, suffix)
	}
	return name
}

func enumCarrierFieldNames(fields []field) []string {
	names := make([]string, len(fields))
	used := make(map[string]bool)
	for index, value := range fields {
		base := fmt.Sprintf("Field%d", index)
		if value.Name != "" && value.Name != "_" {
			runes := []rune(value.Name)
			runes[0] = unicode.ToUpper(runes[0])
			base = "Field" + string(runes)
		}
		name := base
		for suffix := 1; used[name]; suffix++ {
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		names[index] = name
		used[name] = true
	}
	return names
}

func enumPayloadConstructorCall(enum string, value variant) string {
	arguments := make([]string, len(value.Fields))
	for index, field := range value.Fields {
		name := field.Name
		if name == "" {
			name = embeddedGoFieldName(field.Type)
		}
		switch name {
		case "", "_":
			arguments[index] = "*new(" + field.Type + ")"
		default:
			arguments[index] = "payload." + name
		}
	}
	return enumConstructorName(enum, value.Name) + "(" + strings.Join(arguments, ", ") + ")"
}

func embeddedGoFieldName(text string) string {
	expression, err := parser.ParseExpr(text)
	if err != nil {
		return ""
	}
	for {
		switch value := expression.(type) {
		case *ast.Ident:
			return value.Name
		case *ast.SelectorExpr:
			return value.Sel.Name
		case *ast.StarExpr:
			expression = value.X
		case *ast.ParenExpr:
			expression = value.X
		case *ast.IndexExpr:
			expression = value.X
		case *ast.IndexListExpr:
			expression = value.X
		default:
			return ""
		}
	}
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
