package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// checkRules applies tgo safety rules after Go type checking.
func (p *packageUnit) checkRules() {
	for _, source := range p.Sources {
		for _, declaration := range source.File.Decls {
			if p.generatedDecl(declaration) {
				continue
			}
			trusted := p.checkedMethodReceiver(declaration)
			parents := parentNodes(declaration)
			safe, handled := p.checkEnumSwitches(declaration, parents, source.Exhaustive)
			ast.Inspect(declaration, func(node ast.Node) bool {
				p.checkNonNilType(source, node)
				p.checkNode(node, parents, safe, handled, trusted)
				return true
			})
		}
	}
}

// checkedMethodReceiver returns the local value that check can normalize.
func (p *packageUnit) checkedMethodReceiver(declaration ast.Decl) types.Object {
	function, ok := declaration.(*ast.FuncDecl)
	if !ok || function.Name.Name != "check" {
		return nil
	}
	receiver, ok := receiverName(function)
	if !ok {
		return nil
	}
	value := p.Models[receiver]
	if value == nil || !value.CheckedStruct {
		return nil
	}
	if function.Recv == nil || len(function.Recv.List) != 1 ||
		len(function.Recv.List[0].Names) != 1 {
		return nil
	}
	return p.info.Defs[function.Recv.List[0].Names[0]]
}

// checkCheckedStructs checks the required local validation method.
func (p *packageUnit) checkCheckedStructs() {
	for _, source := range p.Sources {
		for _, model := range source.Models {
			if !model.CheckedStruct {
				continue
			}
			object, ok := p.typed.Scope().Lookup(model.Name).(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := types.Unalias(object.Type()).(*types.Named)
			if !ok {
				continue
			}
			method := directMethod(named, "check")
			if method == nil {
				p.failAt(object.Pos(),
					"checked struct %s needs check() (%s, error)",
					model.Name, model.Name)
				continue
			}
			if !validCheckedStructMethod(method, named) {
				p.failAt(method.Pos(),
					"checked struct %s check method must have signature check() (%s, error)",
					model.Name, model.Name)
			}
		}
	}
}

func directMethod(named *types.Named, name string) *types.Func {
	for index := 0; index < named.NumMethods(); index++ {
		method := named.Method(index)
		if method.Name() == name {
			return method
		}
	}
	return nil
}

func validCheckedStructMethod(method *types.Func, named *types.Named) bool {
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil || signature.Params().Len() != 0 ||
		signature.Results().Len() != 2 || signature.Variadic() {
		return false
	}
	errorObject := types.Universe.Lookup("error")
	return errorObject != nil && types.Identical(signature.Recv().Type(), named) &&
		types.Identical(signature.Results().At(0).Type(), named) &&
		types.Identical(signature.Results().At(1).Type(), errorObject.Type())
}

func (p *packageUnit) checkNonNilType(source *source, node ast.Node) {
	pointer, ok := node.(*ast.StarExpr)
	if !ok || !source.NonNil[pointer.Star] || p.info.Types[pointer].IsType() {
		return
	}
	p.failAt(pointer.Star, "%% is only valid in a pointer type")
}

// parentNodes maps each AST node to its direct parent.
func parentNodes(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

// checkNode sends one source node to its applicable tgo checks.
func (p *packageUnit) checkNode(
	node ast.Node,
	parents map[ast.Node]ast.Node,
	safe map[*ast.SelectorExpr]bool,
	handled map[*ast.SelectorExpr]bool,
	trusted types.Object,
) {
	if expression, ok := node.(ast.Expr); ok {
		p.checkRepresentationExpression(expression, parents)
	}
	switch node := node.(type) {
	case *ast.Ident:
		p.checkCheckedCarrier(node)
	case *ast.TypeSpec:
		p.checkTypeSpec(node)
	case *ast.CompositeLit:
		p.checkLiteral(node)
	case *ast.CallExpr:
		p.checkCall(node)
	case *ast.SelectorExpr:
		p.checkSelector(node, safe, handled)
		p.checkCheckedFieldChange(node, parents, trusted)
	}
}

func (p *packageUnit) checkCheckedCarrier(identifier *ast.Ident) {
	if identifier.Pos() == token.NoPos {
		return
	}
	object, ok := p.info.Uses[identifier].(*types.TypeName)
	if !ok || object.Pkg() == nil {
		return
	}
	owner := p
	if object.Pkg().Path() != p.Path {
		owner = p.Imports[object.Pkg().Path()]
	}
	if owner == nil {
		return
	}
	for _, value := range owner.Models {
		if value.CheckedStruct && object.Name() == checkedCarrierName(value.Name) {
			p.fail(identifier, "%s is generated staging ABI; use a checked literal", object.Name())
			return
		}
	}
}

// checkRepresentationExpression reserves an unnamed model layout.
func (p *packageUnit) checkRepresentationExpression(
	expression ast.Expr,
	parents map[ast.Node]ast.Node,
) {
	parent := parents[expression]
	for {
		parentheses, ok := parent.(*ast.ParenExpr)
		if !ok {
			break
		}
		parent = parents[parentheses]
	}
	if specification, ok := parent.(*ast.TypeSpec); ok &&
		!specification.Assign.IsValid() {
		return
	}
	typ := p.info.TypeOf(expression)
	if typ == nil {
		return
	}
	if _, unnamed := types.Unalias(typ).(*types.Struct); !unnamed {
		return
	}
	if model := p.representationType(typ); model != nil {
		p.fail(expression, "%s representation type is private", model.Name)
	}
}

// checkTypeSpec keeps model representation out of new defined types.
func (p *packageUnit) checkTypeSpec(specification *ast.TypeSpec) {
	if specification.Assign.IsValid() {
		return
	}
	model := p.concreteModel(p.info.TypeOf(specification.Type))
	if model != nil {
		p.fail(
			specification.Type,
			"cannot define a new type from %s; use an alias",
			model.Name,
		)
	}
}

// checkSelector blocks direct access to generated model representation.
func (p *packageUnit) checkSelector(
	selector *ast.SelectorExpr,
	safe map[*ast.SelectorExpr]bool,
	handled map[*ast.SelectorExpr]bool,
) {
	if safe[selector] || handled[selector] {
		return
	}
	if selector.Sel.Pos() == token.NoPos {
		return
	}
	selection := p.info.Selections[selector]
	if selection == nil {
		return
	}
	if model := p.representationModel(selection); model != nil {
		p.fail(selector,
			"%s representation is private; use payload constructors and a checked Tag switch",
			model.Name,
		)
		return
	}
}

// representationModel gets the model that declares a selected private field.
func (p *packageUnit) representationModel(selection *types.Selection) *model {
	if selection.Kind() != types.FieldVal {
		return nil
	}
	current := selection.Recv()
	for offset, index := range selection.Index() {
		current = dereference(current)
		structure, ok := current.Underlying().(*types.Struct)
		if !ok || index >= structure.NumFields() {
			return nil
		}
		selected := structure.Field(index)
		if offset == len(selection.Index())-1 {
			model := p.representationType(current)
			if model != nil && modelField(model, selected.Name()) {
				return model
			}
			return nil
		}
		current = selected.Type()
	}
	return nil
}

// representationType matches a model or an identical unnamed struct.
func (p *packageUnit) representationType(typ types.Type) *model {
	if typ == nil {
		return nil
	}
	if model := p.modelForType(typ); model != nil && model.requiresConstructor() {
		return model
	}
	typ = types.Unalias(typ)
	if _, named := typ.(*types.Named); named {
		return nil
	}
	structure, ok := typ.(*types.Struct)
	if !ok {
		return nil
	}
	for _, source := range p.Sources {
		for _, model := range source.Models {
			if !model.requiresConstructor() {
				continue
			}
			object := p.typed.Scope().Lookup(model.Name)
			if object != nil && types.Identical(structure, object.Type().Underlying()) {
				return model
			}
		}
	}
	return nil
}

// modelField reports whether a name is part of a generated private value.
func modelField(model *model, name string) bool {
	if name == "tgoTag" {
		return true
	}
	for _, variant := range model.Variants {
		if name == "tgo"+variant.Name {
			return true
		}
	}
	return false
}

// enumAccessor identifies generated accessors, including promoted methods.
func (p *packageUnit) enumAccessor(selection *types.Selection, name string) bool {
	function, ok := selection.Obj().(*types.Func)
	if !ok {
		return false
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiver := dereference(signature.Recv().Type())
	if model := p.modelForType(receiver); model != nil {
		if name == "Tag" {
			return true
		}
		return enumVariantTag(model, name) != 0
	}
	_, dynamic := receiver.Underlying().(*types.Interface)
	return dynamic && (name == "Tag" || strings.HasSuffix(name, "Payload"))
}

// dereference removes aliases and one or more pointer layers.
func dereference(typ types.Type) types.Type {
	typ = types.Unalias(typ)
	for {
		pointer, ok := typ.(*types.Pointer)
		if !ok {
			return typ
		}
		typ = types.Unalias(pointer.Elem())
	}
}

// checkLiteral requires model construction through its generated API.
func (p *packageUnit) checkLiteral(lit *ast.CompositeLit) {
	if p.checkedLiterals[lit] {
		return
	}
	t := p.info.TypeOf(lit)
	if t == nil {
		return
	}
	if model := p.modelForType(t); model != nil && model.requiresConstructor() {
		p.fail(lit, "use a constructor for %s", model.Name)
	}
}

// checkCall applies tgo rules to conversions.
func (p *packageUnit) checkCall(c *ast.CallExpr) {
	if !p.checkedCalls[c] {
		if function, ok := p.calledFunction(c.Fun); ok &&
			p.generatedCheckedConstructor(function) {
			p.fail(c, "%s is generated Go ABI; use a checked literal", function.Name())
			return
		}
	}
	p.checkConversion(c)
}

func (p *packageUnit) calledFunction(expression ast.Expr) (*types.Func, bool) {
	var object types.Object
	switch expression := expression.(type) {
	case *ast.Ident:
		object = p.info.Uses[expression]
	case *ast.SelectorExpr:
		object = p.info.Uses[expression.Sel]
	}
	function, ok := object.(*types.Func)
	return function, ok
}

func (p *packageUnit) generatedCheckedConstructor(function *types.Func) bool {
	if function == nil || function.Pkg() == nil ||
		!strings.HasPrefix(function.Name(), "New") {
		return false
	}
	owner := p
	if function.Pkg().Path() != p.Path {
		owner = p.Imports[function.Pkg().Path()]
	}
	if owner == nil {
		return false
	}
	value := owner.Models[strings.TrimPrefix(function.Name(), "New")]
	return value != nil && value.CheckedStruct
}

// checkCheckedFieldChange keeps a checked value valid after construction.
func (p *packageUnit) checkCheckedFieldChange(
	selector *ast.SelectorExpr,
	parents map[ast.Node]ast.Node,
	trusted types.Object,
) {
	selection := p.info.Selections[selector]
	if selection == nil || selection.Kind() != types.FieldVal ||
		len(selection.Index()) != 1 {
		return
	}
	value := p.modelForType(dereference(selection.Recv()))
	if value == nil || !value.CheckedStruct {
		return
	}
	parent := parents[selector]
	for {
		wrapped, ok := parent.(*ast.ParenExpr)
		if !ok {
			break
		}
		parent = parents[wrapped]
	}
	changed := false
	switch node := parent.(type) {
	case *ast.AssignStmt:
		for _, left := range node.Lhs {
			if left == selector {
				changed = true
			}
		}
	case *ast.IncDecStmt:
		changed = node.X == selector
	case *ast.UnaryExpr:
		changed = node.Op == token.AND
	}
	if changed {
		if checkedSelectorObject(p.info, selector) == trusted {
			return
		}
		p.fail(selector, "checked field %s cannot be changed after construction", selector.Sel.Name)
	}
}

func checkedSelectorObject(info *types.Info, selector *ast.SelectorExpr) types.Object {
	expression := selector.X
	for {
		parentheses, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parentheses.X
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return nil
	}
	return info.Uses[identifier]
}

// checkConversion rejects conversions that bypass a model constructor.
func (p *packageUnit) checkConversion(call *ast.CallExpr) bool {
	if !p.info.Types[call.Fun].IsType() {
		return false
	}
	if model := p.directModel(p.info.TypeOf(call)); model != nil {
		p.fail(call, "use a constructor for %s", model.Name)
		return true
	}
	if len(call.Args) == 1 && !interfaceType(p.info.TypeOf(call)) {
		if model := p.directModel(p.info.TypeOf(call.Args[0])); model != nil {
			p.fail(call, "cannot convert %s to expose its representation", model.Name)
		}
	}
	return true
}

// directModel gets a concrete model or one admitted by a type parameter.
func (p *packageUnit) directModel(typ types.Type) *model {
	if typ == nil {
		return nil
	}
	typ = types.Unalias(typ)
	if parameter, ok := typ.(*types.TypeParam); ok {
		return p.typeParameterModel(parameter)
	}
	return p.concreteModel(typ)
}

// concreteModel gets a model through pointer layers without reading type sets.
func (p *packageUnit) concreteModel(typ types.Type) *model {
	if typ == nil {
		return nil
	}
	model := p.modelForType(dereference(typ))
	if model != nil && model.requiresConstructor() {
		return model
	}
	return nil
}

type modelMatch struct {
	model *model
	typ   types.Type
}

// typeParameterModel confirms a model against the complete type set.
func (p *packageUnit) typeParameterModel(parameter *types.TypeParam) *model {
	constraint := parameter.Constraint().Underlying().(*types.Interface).Complete()
	matches := p.constraintMatches(
		parameter.Constraint(),
		make(map[types.Type]bool),
	)
	for _, match := range matches {
		if types.Satisfies(match.typ, constraint) {
			return match.model
		}
	}
	return nil
}

// constraintMatches collects model types named by constraint terms.
func (p *packageUnit) constraintMatches(
	typ types.Type,
	seen map[types.Type]bool,
) []modelMatch {
	typ = types.Unalias(typ)
	if seen[typ] {
		return nil
	}
	seen[typ] = true
	if model := p.concreteModel(typ); model != nil {
		return []modelMatch{{model: model, typ: typ}}
	}
	switch typ := typ.(type) {
	case *types.TypeParam:
		return p.constraintMatches(typ.Constraint(), seen)
	case *types.Named:
		if _, ok := typ.Underlying().(*types.Interface); ok {
			return p.constraintMatches(typ.Underlying(), seen)
		}
	case *types.Interface:
		var matches []modelMatch
		for index := 0; index < typ.NumEmbeddeds(); index++ {
			matches = append(
				matches,
				p.constraintMatches(typ.EmbeddedType(index), seen)...,
			)
		}
		return matches
	case *types.Union:
		return p.unionMatches(typ, seen)
	}
	return nil
}

// unionMatches collects protected models named by union terms.
func (p *packageUnit) unionMatches(
	union *types.Union,
	seen map[types.Type]bool,
) []modelMatch {
	var matches []modelMatch
	for index := 0; index < union.Len(); index++ {
		term := union.Term(index)
		matches = append(matches, p.constraintMatches(term.Type(), seen)...)
		if term.Tilde() {
			if model := p.layoutModel(term.Type()); model != nil {
				if modelType := p.localModelType(model); modelType != nil {
					matches = append(matches, modelMatch{model: model, typ: modelType})
				}
			}
		}
	}
	return matches
}

// localModelType gets the Go type emitted for one local model.
func (p *packageUnit) localModelType(model *model) types.Type {
	if p.Models[model.Name] != model {
		return nil
	}
	object := p.typed.Scope().Lookup(model.Name)
	if object == nil {
		return nil
	}
	return object.Type()
}

// layoutModel matches a type whose underlying layout is a protected model.
func (p *packageUnit) layoutModel(typ types.Type) *model {
	for _, source := range p.Sources {
		for _, model := range source.Models {
			if !model.requiresConstructor() {
				continue
			}
			object := p.typed.Scope().Lookup(model.Name)
			if object != nil && types.Identical(typ.Underlying(), object.Type().Underlying()) {
				return model
			}
		}
	}
	return nil
}

// interfaceType reports a concrete interface type, but not a type parameter.
func interfaceType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	typ = types.Unalias(typ)
	if _, parameter := typ.(*types.TypeParam); parameter {
		return false
	}
	_, ok := typ.Underlying().(*types.Interface)
	return ok
}
