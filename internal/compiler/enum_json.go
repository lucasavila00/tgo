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

// enumJSONGo emits JSON methods without a change to enum storage.
func enumJSONGo(declaration *model, jsonPackage, fmtPackage string) string {
	var out strings.Builder
	name := declaration.Name
	config := declaration.JSON
	q := strconv.Quote
	fmt.Fprintf(&out, "func (v %s) MarshalJSON() ([]byte, error) {\nswitch v.tgoTag {\n", name)
	for index, variant := range declaration.Variants {
		fmt.Fprintf(&out, "case %d:\npayload := v.Tgo%s()\n", index+1, variant.Name)
		switch config.Form {
		case "external":
			fmt.Fprintf(&out,
				"return %s.Marshal(map[string]interface{}{%s: payload})\n",
				jsonPackage,
				q(variant.JSONName))
		case "untagged":
			fmt.Fprintf(&out, "return %s.Marshal(payload)\n", jsonPackage)
		case "adjacent":
			fmt.Fprintf(&out,
				"return %s.Marshal(map[string]interface{}{%s: %s, %s: payload})\n",
				jsonPackage,
				q(config.Tag),
				q(variant.JSONName),
				q(config.Content))
		case "internal":
			tagData, _ := json.Marshal(variant.JSONName)
			fmt.Fprintf(&out,
				"data, err := %s.Marshal(payload)\n"+
					"if err != nil { return nil, err }\n"+
					"var object map[string]%s.RawMessage\n"+
					"if err := %s.Unmarshal(data, &object); err != nil { return nil, err }\n"+
					"if object == nil { return nil, %s.Errorf(%s) }\n"+
					"object[%s] = %s.RawMessage(%s)\n"+
					"return %s.Marshal(object)\n",
				jsonPackage,
				jsonPackage,
				jsonPackage,
				fmtPackage, q("expected "+name+" JSON payload object"),
				q(config.Tag),
				jsonPackage,
				q(string(tagData)),
				jsonPackage)
		}
	}
	fmt.Fprintf(&out,
		"default: return nil, %s.Errorf(%s)\n}\n}\n",
		fmtPackage,
		q("invalid "+name+" JSON tag"))
	fmt.Fprintf(&out, "func (v *%s) UnmarshalJSON(data []byte) error {\n", name)
	if config.Form == "untagged" {
		for _, variant := range declaration.Variants {
			fmt.Fprintf(&out,
				"{ var payload %s%s\n"+
					"if err := %s.Unmarshal(data, &payload); err == nil {\n"+
					"*v = New%s%s(payload); return nil } }\n",
				name,
				variant.Name,
				jsonPackage,
				name,
				variant.Name)
		}
		fmt.Fprintf(&out,
			"return %s.Errorf(%s)\n}\n",
			fmtPackage,
			q("no matching "+name+" JSON variant"))
		return out.String()
	}
	fmt.Fprintf(&out,
		"var object map[string]%s.RawMessage\n"+
			"if err := %s.Unmarshal(data, &object); err != nil { return err }\n",
		jsonPackage, jsonPackage)
	out.WriteString("var variant string\nvar payloadData []byte\n")
	if config.Form == "external" {
		fmt.Fprintf(&out,
			"if len(object) != 1 { return %s.Errorf(%s) }\n"+
				"for key, value := range object { variant = key; payloadData = value }\n",
			fmtPackage,
			q("expected one "+name+" JSON variant"))
	} else {
		fmt.Fprintf(&out,
			"tag, ok := object[%s]\n"+
				"if !ok { return %s.Errorf(%s) }\n"+
				"if err := %s.Unmarshal(tag, &variant); err != nil { return err }\n",
			q(config.Tag),
			fmtPackage,
			q("missing "+name+" JSON tag"),
			jsonPackage)
		if config.Form == "adjacent" {
			fmt.Fprintf(&out,
				"var present bool\n"+
					"payloadData, present = object[%s]\n"+
					"if !present { return %s.Errorf(%s) }\n",
				q(config.Content),
				fmtPackage,
				q("missing "+name+" JSON content"))
		} else {
			fmt.Fprintf(&out,
				"delete(object, %s)\n"+
					"var err error\n"+
					"payloadData, err = %s.Marshal(object)\n"+
					"if err != nil { return err }\n",
				q(config.Tag),
				jsonPackage)
		}
	}
	out.WriteString("switch variant {\n")
	for _, variant := range declaration.Variants {
		fmt.Fprintf(&out,
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
	fmt.Fprintf(&out,
		"default: return %s.Errorf(%s, variant)\n}\n}\n",
		fmtPackage,
		q("unknown "+name+" JSON variant %q"))
	return out.String()
}
