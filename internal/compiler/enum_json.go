package compiler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"tgo/pkg/syntax"
)

type enumJSON struct {
	Form    string
	Tag     string
	Content string
}

func jsonControl(tag *syntax.BasicLiteral) (string, bool, error) {
	if tag == nil {
		return "", false, nil
	}
	value, err := strconv.Unquote(tag.Value)
	if err != nil {
		return "", false, err
	}
	value, ok := reflect.StructTag(value).Lookup("json")
	return value, ok, nil
}

func configureEnumJSON(declaration *model, node *syntax.EnumDeclaration) error {
	declaration.JSON.Form = "external"
	control, present, err := jsonControl(node.Tag)
	if err != nil {
		return err
	}
	if present {
		if err := parseEnumJSONOptions(&declaration.JSON, control); err != nil {
			return err
		}
	}
	if err := validateEnumJSON(declaration.JSON); err != nil {
		return err
	}

	names := make(map[string]bool)
	for index, nodeVariant := range node.Variants {
		item := &declaration.Variants[index]
		item.JSONName = item.Name
		name, present, err := jsonControl(nodeVariant.Tag)
		if err != nil {
			return err
		}
		if present {
			item.JSONName = name
		}
		if item.JSONName == "" || names[item.JSONName] {
			return fmt.Errorf("enum JSON variant name %q is empty or duplicate", item.JSONName)
		}
		names[item.JSONName] = true
	}
	return nil
}

func validateEnumJSON(config enumJSON) error {
	switch config.Form {
	case "external", "untagged":
		if config.Tag != "" || config.Content != "" {
			return fmt.Errorf("%s JSON does not use tag or content", config.Form)
		}
	case "internal":
		if config.Tag == "" || config.Content != "" {
			return fmt.Errorf("internal JSON requires tag and does not use content")
		}
	case "adjacent":
		if config.Tag == "" || config.Content == "" || config.Tag == config.Content {
			return fmt.Errorf("adjacent JSON requires different tag and content names")
		}
	default:
		return fmt.Errorf("unknown enum JSON form %q", config.Form)
	}
	return nil
}

func parseEnumJSONOptions(config *enumJSON, control string) error {
	parts := strings.Split(control, ",")
	config.Form = parts[0]
	seen := make(map[string]bool)
	for _, option := range parts[1:] {
		key, value, ok := strings.Cut(option, "=")
		if !ok || value == "" || seen[key] {
			return fmt.Errorf("invalid enum JSON option %q", option)
		}
		seen[key] = true
		switch key {
		case "tag":
			config.Tag = value
		case "content":
			config.Content = value
		default:
			return fmt.Errorf("unknown enum JSON option %q", key)
		}
	}
	return nil
}

func jsonStructFieldName(name string) bool {
	return name != "-" && validJSONFieldName(name)
}

func jsonFieldTag(name string) string {
	return "`json:" + strconv.Quote(name) + "`"
}

func jsonString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

// enumJSONGo emits JSON methods without a change to enum storage.
func enumJSONGo(declaration *model, jsonPackage, fmtPackage string) string {
	var out strings.Builder
	emitEnumJSONMarshal(&out, declaration, jsonPackage, fmtPackage)
	emitEnumJSONUnmarshal(&out, declaration, jsonPackage, fmtPackage)
	return out.String()
}

func emitEnumJSONMarshal(
	out *strings.Builder, declaration *model, jsonPackage string, fmtPackage string,
) {
	name := declaration.Name
	config := declaration.JSON
	fmt.Fprintf(out, "func (v %s) MarshalJSON() ([]byte, error) {\nswitch v.tgoTag {\n", name)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(out, "case %d:\npayload := v.Tgo%s()\n", index+1, variant.Name)
		switch config.Form {
		case "external":
			emitExternalJSONMarshal(out, name, variant, jsonPackage)
		case "untagged":
			fmt.Fprintf(out, "return %s.Marshal(payload)\n", jsonPackage)
		case "adjacent":
			emitAdjacentJSONMarshal(out, name, variant, config, jsonPackage)
		case "internal":
			emitInternalJSONMarshal(out, name, variant, config, jsonPackage, fmtPackage)
		}
	}
	fmt.Fprintf(out,
		"default: return nil, %s.Errorf(%s)\n}\n}\n",
		fmtPackage,
		strconv.Quote("invalid "+name+" JSON tag"))
}

func emitEnumJSONUnmarshal(
	out *strings.Builder, declaration *model, jsonPackage string, fmtPackage string,
) {
	name := declaration.Name
	config := declaration.JSON
	q := strconv.Quote
	fmt.Fprintf(out, "func (v *%s) UnmarshalJSON(data []byte) error {\n", name)
	if config.Form == "untagged" {
		for _, variant := range declaration.Variants {
			fmt.Fprintf(out,
				"{ var payload %s%s\n"+
					"if err := %s.Unmarshal(data, &payload); err == nil {\n"+
					"*v = New%s%s(payload); return nil } }\n",
				name,
				variant.Name,
				jsonPackage,
				name,
				variant.Name)
		}
		fmt.Fprintf(out,
			"return %s.Errorf(%s)\n}\n",
			fmtPackage,
			q("no matching "+name+" JSON variant"))
		return
	}
	out.WriteString("var variant string\n")
	if config.Form != "internal" {
		out.WriteString("var payloadData []byte\n")
	}
	switch {
	case config.Form == "external":
		fmt.Fprintf(out,
			"var object map[string]%s.RawMessage\n"+
				"if err := %s.Unmarshal(data, &object); err != nil { return err }\n",
			jsonPackage, jsonPackage)
		fmt.Fprintf(out,
			"if len(object) != 1 { return %s.Errorf(%s) }\n"+
				"for key, value := range object { variant = key; payloadData = value }\n",
			fmtPackage,
			q("expected one "+name+" JSON variant"))
	case jsonStructFieldName(config.Tag) &&
		(config.Form != "adjacent" || jsonStructFieldName(config.Content)):
		emitTaggedJSONHeader(out, name, config, jsonPackage, fmtPackage)
	default:
		fmt.Fprintf(out,
			"var object map[string]%s.RawMessage\n"+
				"if err := %s.Unmarshal(data, &object); err != nil { return err }\n",
			jsonPackage, jsonPackage)
		fmt.Fprintf(out,
			"tag, ok := object[%s]\n"+
				"if !ok { return %s.Errorf(%s) }\n"+
				"if err := %s.Unmarshal(tag, &variant); err != nil { return err }\n",
			q(config.Tag),
			fmtPackage,
			q("missing "+name+" JSON tag"),
			jsonPackage)
		if config.Form == "adjacent" {
			fmt.Fprintf(out,
				"var present bool\n"+
					"payloadData, present = object[%s]\n"+
					"if !present { return %s.Errorf(%s) }\n",
				q(config.Content),
				fmtPackage,
				q("missing "+name+" JSON content"))
		}
	}
	out.WriteString("switch variant {\n")
	for _, variant := range declaration.Variants {
		if config.Form == "internal" {
			fmt.Fprintf(out,
				"case %s:\n"+
					"var payload %s%s\n"+
					"if err := %s.Unmarshal(data, &payload); err != nil { return err }\n"+
					"*v = New%s%s(payload)\n"+
					"return nil\n",
				q(variant.JSONName),
				name,
				variant.Name,
				jsonPackage,
				name,
				variant.Name)
			continue
		}
		fmt.Fprintf(out,
			"case %s:\n"+
				"var payload %s%s\n"+
				"if err := %s.Unmarshal(payloadData, &payload); err != nil { return err }\n"+
				"*v = New%s%s(payload)\n"+
				"return nil\n",
			q(variant.JSONName),
			name,
			variant.Name,
			jsonPackage,
			name,
			variant.Name)
	}
	fmt.Fprintf(out,
		"default: return %s.Errorf(%s, variant)\n}\n}\n",
		fmtPackage,
		q("unknown "+name+" JSON variant %q"))
}

func emitExternalJSONMarshal(
	out *strings.Builder, enum string, variant variant, jsonPackage string,
) {
	if jsonStructFieldName(variant.JSONName) {
		fmt.Fprintf(out,
			"return %s.Marshal(struct { Payload %s%s %s }{Payload: payload})\n",
			jsonPackage,
			enum,
			variant.Name,
			jsonFieldTag(variant.JSONName))
		return
	}
	prefix := "{" + jsonString(variant.JSONName) + ":"
	emitJSONEnvelope(out, prefix, jsonPackage)
}

func emitAdjacentJSONMarshal(
	out *strings.Builder, enum string, variant variant, config enumJSON, jsonPackage string,
) {
	if jsonStructFieldName(config.Tag) && jsonStructFieldName(config.Content) {
		fmt.Fprintf(out,
			"return %s.Marshal(struct {\n"+
				"Variant string %s\n"+
				"Payload %s%s %s\n"+
				"}{Variant: %s, Payload: payload})\n",
			jsonPackage,
			jsonFieldTag(config.Tag),
			enum,
			variant.Name,
			jsonFieldTag(config.Content),
			strconv.Quote(variant.JSONName))
		return
	}
	prefix := "{" + jsonString(config.Tag) + ":" + jsonString(variant.JSONName) +
		"," + jsonString(config.Content) + ":"
	emitJSONEnvelope(out, prefix, jsonPackage)
}

func emitInternalJSONMarshal(
	out *strings.Builder,
	enum string,
	variant variant,
	config enumJSON,
	jsonPackage string,
	fmtPackage string,
) {
	prefix := "{" + jsonString(config.Tag) + ":" + jsonString(variant.JSONName)
	fmt.Fprintf(out,
		"payloadData, err := %s.Marshal(payload)\n"+
			"if err != nil { return nil, err }\n"+
			"if len(payloadData) < 2 || payloadData[0] != '{' || "+
			"payloadData[len(payloadData)-1] != '}' {\n"+
			"return nil, %s.Errorf(%s) }\n"+
			"if len(payloadData) == 2 { return []byte(%s), nil }\n"+
			"result := make([]byte, 0, len(payloadData)+%d)\n"+
			"result = append(result, %s...)\n"+
			"result = append(result, ',')\n"+
			"result = append(result, payloadData[1:]...)\n"+
			"return result, nil\n",
		jsonPackage,
		fmtPackage,
		strconv.Quote("expected "+enum+" JSON payload object"),
		strconv.Quote(prefix+"}"),
		len(prefix),
		strconv.Quote(prefix))
}

func emitJSONEnvelope(out *strings.Builder, prefix string, jsonPackage string) {
	fmt.Fprintf(out,
		"payloadData, err := %s.Marshal(payload)\n"+
			"if err != nil { return nil, err }\n"+
			"result := make([]byte, 0, len(payloadData)+%d)\n"+
			"result = append(result, %s...)\n"+
			"result = append(result, payloadData...)\n"+
			"result = append(result, '}')\n"+
			"return result, nil\n",
		jsonPackage,
		len(prefix)+1,
		strconv.Quote(prefix))
}

func emitTaggedJSONHeader(
	out *strings.Builder,
	enum string,
	config enumJSON,
	jsonPackage string,
	fmtPackage string,
) {
	contentField := ""
	contentCheck := ""
	if config.Form == "adjacent" {
		contentField = "Content " + jsonPackage + ".RawMessage " +
			jsonFieldTag(config.Content) + "\n"
		contentCheck = "payloadData = object.Content\n" +
			"if payloadData == nil { return " + fmtPackage + ".Errorf(" +
			strconv.Quote("missing "+enum+" JSON content") + ") }\n"
	}
	fmt.Fprintf(out,
		"var object struct {\n"+
			"Tag string %s\n%s"+
			"}\n"+
			"if err := %s.Unmarshal(data, &object); err != nil { return err }\n"+
			"if object.Tag == \"\" { return %s.Errorf(%s) }\n"+
			"variant = object.Tag\n%s",
		jsonFieldTag(config.Tag),
		contentField,
		jsonPackage,
		fmtPackage,
		strconv.Quote("missing "+enum+" JSON tag"),
		contentCheck)
}
