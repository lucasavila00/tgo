package compiler

import (
	"go/types"
	"reflect"
	"strings"
	"unicode"
)

// checkEnumJSONFields checks the JSON names after Go resolves embedded types.
func (p *packageUnit) checkEnumJSONFields() {
	for _, source := range p.Sources {
		for _, model := range source.Models {
			if !model.Enum || model.JSON.Form != "internal" {
				continue
			}
			for _, variant := range model.Variants {
				payload := p.typed.Scope().Lookup(model.Name + variant.Name).Type()
				if jsonFieldConflict(payload, model.JSON.Tag) {
					p.failAt(source.File.Package,
						"internal JSON tag %q conflicts with payload field in %s.%s",
						model.JSON.Tag,
						model.Name,
						variant.Name)
				}
			}
		}
	}
}

type jsonFieldMatch struct {
	depth  int
	tagged bool
}

func jsonFieldConflict(payload types.Type, name string) bool {
	matches := []jsonFieldMatch(nil)
	collectJSONFields(payload, name, 0, make(map[types.Type]bool), &matches)
	depth := int(^uint(0) >> 1)
	for _, match := range matches {
		if match.depth < depth {
			depth = match.depth
		}
	}
	tagged, plain := 0, 0
	for _, match := range matches {
		if match.depth != depth {
			continue
		}
		if match.tagged {
			tagged++
		} else {
			plain++
		}
	}
	return tagged == 1 || tagged == 0 && plain == 1
}

func collectJSONFields(
	typ types.Type, name string, depth int, path map[types.Type]bool, matches *[]jsonFieldMatch,
) {
	typ = dereference(typ)
	structure, ok := typ.Underlying().(*types.Struct)
	if !ok || path[typ] {
		return
	}
	path[typ] = true
	defer delete(path, typ)
	for index := range structure.NumFields() {
		field := structure.Field(index)
		embeddedType := dereference(field.Type())
		_, embeddedStruct := embeddedType.Underlying().(*types.Struct)
		if !field.Exported() && (!field.Embedded() || !embeddedStruct) {
			continue
		}
		tag := reflect.StructTag(structure.Tag(index)).Get("json")
		if tag == "-" {
			continue
		}
		fieldName := strings.Split(tag, ",")[0]
		if !validJSONFieldName(fieldName) {
			fieldName = ""
		}
		tagged := fieldName != ""
		if !tagged && field.Embedded() && embeddedStruct {
			collectJSONFields(field.Type(), name, depth+1, path, matches)
			continue
		}
		if fieldName == "" {
			fieldName = field.Name()
		}
		if strings.EqualFold(fieldName, name) {
			*matches = append(*matches, jsonFieldMatch{depth: depth, tagged: tagged})
		}
	}
}

// validJSONFieldName uses the field name rules from encoding/json.
func validJSONFieldName(name string) bool {
	if name == "" {
		return false
	}
	for _, char := range name {
		if strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", char) {
			continue
		}
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}
