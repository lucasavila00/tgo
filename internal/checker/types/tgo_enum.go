package types

import "github.com/lucasavila00/tgo/internal/ast"

// Enum records a declared enum and the variants that belong to that declaration.
type Enum struct {
	layoutReady bool
	named       *Named
	tag         *Named
	variants    []*EnumVariant
	declaration *ast.EnumType
	object      *TypeName
}

// EnumVariant identifies one variant and its generated Go payload type.
type EnumVariant struct {
	constructor *Func
	enum        *Enum
	name        string
	typ         *Named
	index       int
	boxed       bool
	declaration *ast.EnumVariant
	object      *TypeName
}

// EnumOf returns the enum for a declared type, alias, or exact type parameter.
func EnumOf(typ Type) *Enum {
	if typ == nil {
		return nil
	}
	if parameter, ok := Unalias(typ).(*TypeParam); ok {
		terms := parameter.iface().typeSet().terms
		if len(terms) != 1 || terms[0] == nil || terms[0].tilde || terms[0].typ == nil {
			return nil
		}
		return EnumOf(terms[0].typ)
	}
	named, ok := Unalias(typ).(*Named)
	if !ok {
		return nil
	}
	return named.enum
}

// Named returns the enum's declared type.
func (enum *Enum) Named() *Named { return enum.named }

// TagType returns the generated named integer type used by Tag.
func (enum *Enum) TagType() *Named { return enum.tag }

// Variants returns the variants in declaration order.
func (enum *Enum) Variants() []*EnumVariant { return enum.variants }

// Declaration returns the original enum syntax.
func (enum *Enum) Declaration() *ast.EnumType { return enum.declaration }

// Enum returns the variant's owning declaration.
func (variant *EnumVariant) Enum() *Enum { return variant.enum }

// Name returns the source variant name.
func (variant *EnumVariant) Name() string { return variant.name }

// Type returns the ordinary Go type that holds this variant's fields.
func (variant *EnumVariant) Type() *Named { return variant.typ }

// Index returns the variant's tag number.
func (variant *EnumVariant) Index() int { return variant.index }

// Boxed reports whether this variant uses the shared interface field.
func (variant *EnumVariant) Boxed() bool { return variant.boxed }

// Declaration returns the original variant syntax.
func (variant *EnumVariant) Declaration() *ast.EnumVariant { return variant.declaration }
