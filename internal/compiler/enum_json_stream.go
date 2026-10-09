package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

// enumJSONHelpers emits one streaming envelope helper for each used form.
func enumJSONHelpers(
	jsonV2Package string,
	jsonTextPackage string,
	externalJSONTo string,
	adjacentJSONTo string,
	hasExternal bool,
	hasAdjacent bool,
) string {
	var out strings.Builder
	if hasExternal {
		fmt.Fprintf(
			&out,
			"func %s[T interface{}](out *%s.Encoder, name string, payload T) error {\n"+
				"if err := out.WriteToken(%s.BeginObject); err != nil { return err }\n"+
				"if err := out.WriteToken(%s.String(name)); err != nil { return err }\n"+
				"if err := %s.MarshalEncode(out, payload); err != nil { return err }\n"+
				"return out.WriteToken(%s.EndObject)\n"+
				"}\n",
			externalJSONTo,
			jsonTextPackage,
			jsonTextPackage,
			jsonTextPackage,
			jsonV2Package,
			jsonTextPackage,
		)
	}
	if hasAdjacent {
		fmt.Fprintf(
			&out,
			"func %s[T interface{}](\n"+
				"out *%s.Encoder, tag string, name string, content string, payload T,\n"+
				") error {\n"+
				"if err := out.WriteToken(%s.BeginObject); err != nil { return err }\n"+
				"if err := out.WriteToken(%s.String(tag)); err != nil { return err }\n"+
				"if err := out.WriteToken(%s.String(name)); err != nil { return err }\n"+
				"if err := out.WriteToken(%s.String(content)); err != nil { return err }\n"+
				"if err := %s.MarshalEncode(out, payload); err != nil { return err }\n"+
				"return out.WriteToken(%s.EndObject)\n"+
				"}\n",
			adjacentJSONTo,
			jsonTextPackage,
			jsonTextPackage,
			jsonTextPackage,
			jsonTextPackage,
			jsonTextPackage,
			jsonV2Package,
			jsonTextPackage,
		)
	}
	return out.String()
}

// emitJSONUnmarshalFrom passes one complete value to the byte-slice method.
func emitJSONUnmarshalFrom(
	out *strings.Builder,
	name string,
	jsonTextPackage string,
) {
	fmt.Fprintf(
		out,
		"func (v *%s) UnmarshalJSONFrom(in *%s.Decoder) error {\n"+
			"data, err := in.ReadValue()\n"+
			"if err != nil { return err }\n"+
			"return v.UnmarshalJSON(data)\n"+
			"}\n",
		name,
		jsonTextPackage,
	)
}

// emitEnumJSONMarshalTo writes directly into Go 1.27's JSON encoder.
func emitEnumJSONMarshalTo(
	out *strings.Builder,
	declaration *model,
	jsonV2Package string,
	jsonTextPackage string,
	fmtPackage string,
	externalJSONTo string,
	adjacentJSONTo string,
) {
	name := declaration.Name
	config := declaration.JSON
	fmt.Fprintf(
		out,
		"func (v %s) MarshalJSONTo(out *%s.Encoder) error {\n"+
			"switch v.tgoTag {\n",
		name,
		jsonTextPackage,
	)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(out, "case %d:\npayload := v.Tgo%s()\n", index+1, variant.Name)
		switch config.Form {
		case "external":
			fmt.Fprintf(
				out,
				"return %s(out, %s, payload)\n",
				externalJSONTo,
				strconv.Quote(variant.JSONName),
			)
		case "internal":
			emitInternalJSONMarshalTo(
				out, name, variant, config, jsonV2Package, jsonTextPackage,
			)
		case "adjacent":
			fmt.Fprintf(
				out,
				"return %s(out, %s, %s, %s, payload)\n",
				adjacentJSONTo,
				strconv.Quote(config.Tag),
				strconv.Quote(variant.JSONName),
				strconv.Quote(config.Content),
			)
		case "untagged":
			fmt.Fprintf(out, "return %s.MarshalEncode(out, payload)\n", jsonV2Package)
		}
	}
	fmt.Fprintf(
		out,
		"default: return %s.Errorf(%s)\n}\n}\n",
		fmtPackage,
		strconv.Quote("invalid "+name+" JSON tag"),
	)
}

func emitInternalJSONMarshalTo(
	out *strings.Builder,
	enum string,
	variant variant,
	config enumJSON,
	jsonV2Package string,
	jsonTextPackage string,
) {
	payloadType := enum + variant.Name
	fmt.Fprintf(
		out,
		"_, marshalsJSON := interface{}(payload).(interface { MarshalJSON() ([]byte, error) })\n"+
			"_, marshalsText := interface{}(payload)."+
			"(interface { MarshalText() ([]byte, error) })\n"+
			"_, marshalsJSONTo := interface{}(payload)."+
			"(interface { MarshalJSONTo(*%s.Encoder) error })\n"+
			"if marshalsJSON || marshalsText || marshalsJSONTo {\n"+
			"data, err := v.MarshalJSON()\n"+
			"if err != nil { return err }\n"+
			"return out.WriteValue(data)\n"+
			"}\n"+
			"return %s.MarshalEncode(out, struct {\n"+
			"Variant string %s\n"+
			"%s\n"+
			"}{Variant: %s, %s: payload})\n",
		jsonTextPackage,
		jsonV2Package,
		jsonFieldTag(config.Tag),
		payloadType,
		strconv.Quote(variant.JSONName),
		payloadType,
	)
}

// emitExternalJSONUnmarshalFrom reads the object without a map.
func emitExternalJSONUnmarshalFrom(
	out *strings.Builder,
	declaration *model,
	jsonV2Package string,
	jsonTextPackage string,
	fmtPackage string,
) {
	name := declaration.Name
	q := strconv.Quote
	fmt.Fprintf(
		out,
		"func (v *%s) UnmarshalJSONFrom(in *%s.Decoder) error {\n"+
			"token, err := in.ReadToken()\n"+
			"if err != nil { return err }\n"+
			"if token.Kind() != '{' { return %s.Errorf(%s) }\n"+
			"var payloadData %s.Value\n"+
			"var unknown string\n"+
			"selected := 0\n"+
			"haveName := false\n"+
			"multiple := false\n"+
			"for in.PeekKind() != '}' {\n"+
			"nameToken, err := in.ReadToken()\n"+
			"if err != nil { return err }\n"+
			"wireName := nameToken.String()\n"+
			"current := 0\n"+
			"switch wireName {\n",
		name,
		jsonTextPackage,
		fmtPackage,
		q("expected one "+name+" JSON variant"),
		jsonTextPackage,
	)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(out, "case %s: current = %d\n", q(variant.JSONName), index+1)
	}
	out.WriteString("}\n" +
		"same := haveName && current == selected\n" +
		"if same && current == 0 { same = wireName == unknown }\n" +
		"if !haveName {\n" +
		"haveName = true\n" +
		"selected = current\n" +
		"if current == 0 { unknown = string(append([]byte(nil), wireName...)) }\n" +
		"} else if !same { multiple = true }\n" +
		"if !multiple && current > 0 && current == selected {\n" +
		"raw, err := in.ReadValue()\n" +
		"if err != nil { return err }\n" +
		"payloadData = append(payloadData[:0], raw...)\n" +
		"} else if err := in.SkipValue(); err != nil { return err }\n" +
		"}\n" +
		"if _, err := in.ReadToken(); err != nil { return err }\n")
	fmt.Fprintf(
		out,
		"if !haveName || multiple { return %s.Errorf(%s) }\n"+
			"if selected == 0 { return %s.Errorf(%s, unknown) }\n"+
			"switch selected {\n",
		fmtPackage,
		q("expected one "+name+" JSON variant"),
		fmtPackage,
		q("unknown "+name+" JSON variant %q"),
	)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(
			out,
			"case %d:\n"+
				"var payload %s%s\n"+
				"if err := %s.Unmarshal(payloadData, &payload, in.Options()); "+
				"err != nil { return err }\n"+
				"*v = New%s%s(payload)\n"+
				"return nil\n",
			index+1,
			name,
			variant.Name,
			jsonV2Package,
			name,
			variant.Name,
		)
	}
	fmt.Fprintf(
		out,
		"default: return %s.Errorf(%s)\n}\n}\n",
		fmtPackage,
		q("invalid "+name+" JSON tag"),
	)
}

// emitAdjacentJSONUnmarshalFrom avoids the fallback map and its copied values.
func emitAdjacentJSONUnmarshalFrom(
	out *strings.Builder,
	declaration *model,
	jsonV2Package string,
	jsonTextPackage string,
	stringsPackage string,
	fmtPackage string,
) {
	name := declaration.Name
	config := declaration.JSON
	q := strconv.Quote
	fmt.Fprintf(
		out,
		"func (v *%s) UnmarshalJSONFrom(in *%s.Decoder) error {\n"+
			"token, err := in.ReadToken()\n"+
			"if err != nil { return err }\n"+
			"if token.Kind() != '{' { return %s.Errorf(%s) }\n"+
			"var contentData %s.Value\n"+
			"selected := 0\n"+
			"var unknown string\n"+
			"contentPresent := false\n"+
			"for in.PeekKind() != '}' {\n"+
			"nameToken, err := in.ReadToken()\n"+
			"if err != nil { return err }\n"+
			"wireName := nameToken.String()\n"+
			"field := 0\n",
		name,
		jsonTextPackage,
		fmtPackage,
		q("missing "+name+" JSON tag"),
		jsonTextPackage,
	)
	if jsonStructFieldName(config.Tag) && jsonStructFieldName(config.Content) {
		fmt.Fprintf(
			out,
			"switch {\n"+
				"case wireName == %s: field = 1\n"+
				"case wireName == %s: field = 2\n"+
				"case %s.EqualFold(wireName, %s): field = 1\n"+
				"case %s.EqualFold(wireName, %s): field = 2\n"+
				"}\n",
			q(config.Tag),
			q(config.Content),
			stringsPackage,
			q(config.Tag),
			stringsPackage,
			q(config.Content),
		)
	} else {
		fmt.Fprintf(
			out,
			"switch wireName {\n"+
				"case %s: field = 1\n"+
				"case %s: field = 2\n"+
				"}\n",
			q(config.Tag),
			q(config.Content),
		)
	}
	out.WriteString(
		"switch field {\n" +
			"case 1:\n" +
			"if in.PeekKind() == 'n' {\n" +
			"if _, err := in.ReadToken(); err != nil { return err }\n" +
			"break\n" +
			"}\n" +
			"if in.PeekKind() != '\"' {\n" +
			"var invalid string\n" +
			"return " + jsonV2Package + ".UnmarshalDecode(in, &invalid)\n" +
			"}\n" +
			"tagToken, err := in.ReadToken()\n" +
			"if err != nil { return err }\n" +
			"variantName := tagToken.String()\n" +
			"selected = 0\n" +
			"unknown = \"\"\n" +
			"switch variantName {\n",
	)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(out, "case %s: selected = %d\n", q(variant.JSONName), index+1)
	}
	out.WriteString(
		"default: unknown = string(append([]byte(nil), variantName...))\n" +
			"}\n" +
			"case 2:\n" +
			"raw, err := in.ReadValue()\n" +
			"if err != nil { return err }\n" +
			"contentData = append(contentData[:0], raw...)\n" +
			"contentPresent = true\n" +
			"default:\n" +
			"if err := in.SkipValue(); err != nil { return err }\n" +
			"}\n" +
			"}\n" +
			"if _, err := in.ReadToken(); err != nil { return err }\n",
	)
	fmt.Fprintf(
		out,
		"if selected == 0 && unknown == \"\" { return %s.Errorf(%s) }\n"+
			"if !contentPresent { return %s.Errorf(%s) }\n"+
			"if selected == 0 { return %s.Errorf(%s, unknown) }\n"+
			"switch selected {\n",
		fmtPackage,
		q("missing "+name+" JSON tag"),
		fmtPackage,
		q("missing "+name+" JSON content"),
		fmtPackage,
		q("unknown "+name+" JSON variant %q"),
	)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(
			out,
			"case %d:\n"+
				"var payload %s%s\n"+
				"if err := %s.Unmarshal(contentData, &payload, in.Options()); "+
				"err != nil { return err }\n"+
				"*v = New%s%s(payload)\n"+
				"return nil\n",
			index+1,
			name,
			variant.Name,
			jsonV2Package,
			name,
			variant.Name,
		)
	}
	fmt.Fprintf(
		out,
		"default: return %s.Errorf(%s)\n}\n}\n",
		fmtPackage,
		q("invalid "+name+" JSON tag"),
	)
}
