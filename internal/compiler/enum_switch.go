package compiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"tgo/internal/variantflow"
)

// checkEnumSwitches checks exhaustive enum tag switches and records permitted accessors.
func (p *packageUnit) checkEnumSwitches(
	root ast.Node,
	parents map[ast.Node]ast.Node,
	exhaustive map[token.Pos]bool,
) (map[*ast.SelectorExpr]bool, map[*ast.SelectorExpr]bool) {
	safe := make(map[*ast.SelectorExpr]bool)
	handled := make(map[*ast.SelectorExpr]bool)
	ast.Inspect(root, func(node ast.Node) bool {
		statement, ok := node.(*ast.SwitchStmt)
		if ok {
			p.checkEnumSwitch(statement, parents, exhaustive, safe, handled)
		}
		return true
	})
	return safe, handled
}

const enumDefaultComment = "// unreachable: tgolint requires a case per tag"

// checkEnumSwitch checks one switch whose tag is a generated Tag call.
func (p *packageUnit) checkEnumSwitch(
	statement *ast.SwitchStmt,
	parents map[ast.Node]ast.Node,
	exhaustive map[token.Pos]bool,
	safe map[*ast.SelectorExpr]bool,
	handled map[*ast.SelectorExpr]bool,
) {
	receiver, selector, model, tagType := p.enumTagCall(statement.Tag)
	if model == nil {
		return
	}
	safe[selector] = true
	seen := make(map[int]bool)
	clauseTypes := make(map[*ast.CaseClause]variantflow.Type)
	declaredType := variantflow.All(len(model.Variants))
	explicitType := variantflow.Never()
	hasSentinelDefault := false
	hasDefault := false
	labelsResolved := true
	for _, item := range statement.Body.List {
		clause := item.(*ast.CaseClause)
		fallthroughBranch := enumClauseFallthrough(clause)
		if fallthroughBranch != nil {
			p.fail(fallthroughBranch, "%s: fallthrough is not allowed in a tag switch", model.Name)
		}
		if len(clause.List) == 0 {
			hasDefault = true
			hasSentinelDefault = p.enumDefaultIsExhaustive(
				clause, receiver, model, exhaustive[clause.Case],
			)
			continue
		}
		tags, resolved := p.enumCaseTags(clause, model, tagType, seen)
		labelsResolved = labelsResolved && resolved
		clauseTypes[clause] = variantflow.Intersect(declaredType, enumVariantType(tags))
		explicitType = variantflow.Union(explicitType, clauseTypes[clause])
		for tag := range tags {
			seen[tag] = true
		}
	}
	defaultType := variantflow.Without(declaredType, explicitType)
	for _, item := range statement.Body.List {
		clause := item.(*ast.CaseClause)
		flowType, explicitClause := clauseTypes[clause]
		defaultClause := !explicitClause
		if defaultClause {
			flowType = defaultType
		}
		p.checkEnumCaseAccessors(
			clause, statement, receiver, model, flowType, defaultClause,
			parents, safe, handled,
		)
	}
	if labelsResolved && hasSentinelDefault {
		var missing []string
		for _, tag := range variantflow.Tags(defaultType) {
			missing = append(missing, enumTagConstant(model, tag))
		}
		if len(missing) > 0 {
			p.fail(statement, "%s: switch is missing cases: %s",
				model.Name, strings.Join(missing, ", "))
		}
	}
	if !hasDefault {
		p.fail(statement, "%s: switch must have a default clause", model.Name)
	}
}

func enumVariantType(tags map[int]bool) variantflow.Type {
	result := variantflow.Never()
	for tag := range tags {
		result = variantflow.Union(result, variantflow.Variant(tag))
	}
	return result
}

func (p *packageUnit) enumDefaultIsExhaustive(
	clause *ast.CaseClause,
	receiver ast.Expr,
	model *model,
	sourceExhaustive bool,
) bool {
	sentinel := p.enumDefaultSentinel(clause, receiver, model)
	if sourceExhaustive && !sentinel {
		p.fail(clause, "%s: exhaustive clause requires the predeclared panic", model.Name)
	}
	return sourceExhaustive || sentinel
}

// enumTagCall resolves a generated tag call and its receiver model.
//
//nolint:cyclop // Each guard rejects one noncanonical form.
func (p *packageUnit) enumTagCall(
	expression ast.Expr,
) (ast.Expr, *ast.SelectorExpr, *model, types.Type) {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return nil, nil, nil, nil
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Tag" {
		return nil, nil, nil, nil
	}
	selection := p.info.Selections[selector]
	if selection == nil || !p.enumAccessor(selection, selector.Sel.Name) {
		return nil, nil, nil, nil
	}
	function, ok := selection.Obj().(*types.Func)
	if !ok {
		return nil, nil, nil, nil
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil || signature.Results().Len() != 1 {
		return nil, nil, nil, nil
	}
	model := p.modelForType(dereference(signature.Recv().Type()))
	if model == nil {
		if parameter, ok := types.Unalias(p.info.TypeOf(selector.X)).(*types.TypeParam); ok {
			model = p.enumTypeParameterModel(parameter)
		}
	}
	if model == nil || len(model.Variants) == 0 {
		return nil, nil, nil, nil
	}
	return selector.X, selector, model, signature.Results().At(0).Type()
}

func (p *packageUnit) enumCaseTags(
	clause *ast.CaseClause,
	model *model,
	tagType types.Type,
	seen map[int]bool,
) (map[int]bool, bool) {
	resolved := true
	tags := make(map[int]bool)
	for _, expression := range clause.List {
		value := p.info.Types[expression].Value
		if value == nil || value.Kind() != constant.Int ||
			!p.enumTagExpression(expression, model, tagType) {
			p.fail(expression, "%s: case label must be a tag constant", model.Name)
			resolved = false
			continue
		}
		tag64, exact := constant.Int64Val(value)
		if !exact || tag64 < 1 || tag64 > int64(len(model.Variants)) {
			p.fail(expression, "%s: case label must be a tag constant", model.Name)
			resolved = false
			continue
		}
		tag := int(tag64)
		if tags[tag] || seen[tag] {
			p.fail(expression, "%s occurs more than once", enumTagConstant(model, tag))
			continue
		}
		tags[tag] = true
	}
	return tags, resolved
}

func (p *packageUnit) enumTagExpression(
	expression ast.Expr, model *model, tagType types.Type,
) bool {
	return p.enumTagExpressionSeen(expression, model, tagType, make(map[*types.Const]bool))
}

func (p *packageUnit) enumTagExpressionSeen(
	expression ast.Expr,
	model *model,
	tagType types.Type,
	seen map[*types.Const]bool,
) bool {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	switch expression := expression.(type) {
	case *ast.Ident:
		constantObject, ok := p.info.Uses[expression].(*types.Const)
		return ok && p.enumTagConstantExpression(constantObject, model, tagType, seen)
	case *ast.SelectorExpr:
		constantObject, ok := p.info.Uses[expression.Sel].(*types.Const)
		return ok && p.enumTagConstantExpression(constantObject, model, tagType, seen)
	case *ast.CallExpr:
		return len(expression.Args) == 1 && p.info.Types[expression.Fun].IsType() &&
			types.Identical(p.info.TypeOf(expression.Fun), tagType)
	default:
		return false
	}
}

//nolint:cyclop,gocognit // The scan follows Go constant declarations and aliases.
func (p *packageUnit) enumTagConstantExpression(
	object *types.Const,
	model *model,
	tagType types.Type,
	seen map[*types.Const]bool,
) bool {
	if !types.Identical(object.Type(), tagType) || seen[object] {
		return false
	}
	named, ok := types.Unalias(tagType).(*types.Named)
	if ok && named.Obj().Pkg() == object.Pkg() {
		for tag := 1; tag <= len(model.Variants); tag++ {
			if object.Name() == enumTagConstant(model, tag) {
				return true
			}
		}
	}
	seen[object] = true
	defer delete(seen, object)
	for _, file := range p.Files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, specification := range general.Specs {
				value := specification.(*ast.ValueSpec)
				for index, name := range value.Names {
					if p.info.Defs[name] != object || len(value.Values) == 0 {
						continue
					}
					right := value.Values[len(value.Values)-1]
					if index < len(value.Values) {
						right = value.Values[index]
					}
					return p.enumTagExpressionSeen(right, model, tagType, seen)
				}
			}
		}
	}
	return false
}

func enumTagConstant(model *model, tag int) string {
	return model.Name + "Tag" + model.Variants[tag-1].Name
}

//nolint:cyclop,gocognit // Each guard checks one required sentinel token.
func (p *packageUnit) enumDefaultSentinel(
	clause *ast.CaseClause,
	receiver ast.Expr,
	model *model,
) bool {
	if len(clause.Body) != 1 {
		return false
	}
	expression, ok := clause.Body[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	panicCall, ok := expression.X.(*ast.CallExpr)
	if !ok || len(panicCall.Args) != 1 {
		return false
	}
	panicName, ok := panicCall.Fun.(*ast.Ident)
	if !ok || panicName.Name != "panic" {
		return false
	}
	if _, ok := p.info.Uses[panicName].(*types.Builtin); !ok {
		return false
	}
	unknownCall, ok := panicCall.Args[0].(*ast.CallExpr)
	if !ok || len(unknownCall.Args) != 0 {
		return false
	}
	selector, ok := unknownCall.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "UnknownTag" ||
		!sameEnumReceiver(p.info, receiver, selector.X) {
		return false
	}
	selection := p.info.Selections[selector]
	if selection == nil {
		return false
	}
	function, ok := selection.Obj().(*types.Func)
	if !ok || function.Name() != "UnknownTag" {
		return false
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	unknownModel := p.modelForType(dereference(signature.Recv().Type()))
	if unknownModel == nil {
		if parameter, ok := types.Unalias(p.info.TypeOf(selector.X)).(*types.TypeParam); ok {
			unknownModel = p.enumTypeParameterModel(parameter)
		}
	}
	if unknownModel != model {
		return false
	}
	expressionPosition := p.fs.Position(expression.End())
	for _, file := range p.Files {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				commentPosition := p.fs.Position(comment.Slash)
				if comment.Text == enumDefaultComment &&
					commentPosition.Filename == expressionPosition.Filename &&
					commentPosition.Line == expressionPosition.Line &&
					comment.Slash > expression.End() {
					return true
				}
			}
		}
	}
	return false
}

//nolint:cyclop,gocognit // The walk enforces each canonical switch boundary.
func (p *packageUnit) checkEnumCaseAccessors(
	clause *ast.CaseClause,
	tagSwitch *ast.SwitchStmt,
	receiver ast.Expr,
	model *model,
	flowType variantflow.Type,
	defaultClause bool,
	parents map[ast.Node]ast.Node,
	safe map[*ast.SelectorExpr]bool,
	handled map[*ast.SelectorExpr]bool,
) {
	if enumClauseAssignsReceiver(p.info, clause, receiver) {
		return
	}
	for _, statement := range clause.Body {
		ast.Inspect(statement, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			if nested, ok := node.(*ast.SwitchStmt); ok && nested != tagSwitch {
				nestedReceiver, _, nestedModel, _ := p.enumTagCall(nested.Tag)
				if nestedModel == model && sameEnumReceiver(p.info, receiver, nestedReceiver) {
					return false
				}
			}
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || !sameEnumReceiver(p.info, receiver, selector.X) {
				return true
			}
			tag := enumVariantTag(model, selector.Sel.Name)
			if tag == 0 {
				return true
			}
			call, direct := parents[selector].(*ast.CallExpr)
			if !direct || call.Fun != selector {
				return true
			}
			handled[selector] = true
			active, narrowed := variantflow.Singleton(flowType)
			if narrowed && active == tag {
				safe[selector] = true
				return true
			}
			caseName := "default"
			if !defaultClause && !narrowed {
				caseName = "a multi-tag case"
			}
			if !defaultClause && narrowed {
				caseName = "case " + enumTagConstant(model, active)
			}
			p.fail(selector, "%s: %s called under %s",
				model.Name, selector.Sel.Name, caseName)
			return true
		})
	}
}

func enumVariantTag(model *model, method string) int {
	for index, variant := range model.Variants {
		if method == variant.Name+"Payload" {
			return index + 1
		}
	}
	return 0
}

func enumClauseFallthrough(clause *ast.CaseClause) *ast.BranchStmt {
	for index := len(clause.Body) - 1; index >= 0; index-- {
		statement := clause.Body[index]
		if _, empty := statement.(*ast.EmptyStmt); empty {
			continue
		}
		for {
			label, ok := statement.(*ast.LabeledStmt)
			if !ok {
				break
			}
			statement = label.Stmt
		}
		branch, ok := statement.(*ast.BranchStmt)
		if ok && branch.Tok == token.FALLTHROUGH {
			return branch
		}
		return nil
	}
	return nil
}

func enumClauseAssignsReceiver(
	info *types.Info,
	clause *ast.CaseClause,
	receiver ast.Expr,
) bool {
	assigned := false
	ast.Inspect(clause, func(node ast.Node) bool {
		if node == nil || assigned {
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, target := range node.Lhs {
				if enumReceiverWrite(info, target, receiver) {
					assigned = true
					return false
				}
			}
		case *ast.IncDecStmt:
			assigned = enumReceiverWrite(info, node.X, receiver)
		case *ast.RangeStmt:
			assigned = enumReceiverWrite(info, node.Key, receiver) ||
				enumReceiverWrite(info, node.Value, receiver)
		}
		return !assigned
	})
	return assigned
}

// enumTypeParameterModel gets one exact enum constraint.
func (p *packageUnit) enumTypeParameterModel(parameter *types.TypeParam) *model {
	return p.enumConstraintModel(parameter.Constraint(), make(map[types.Type]bool))
}

func (p *packageUnit) enumConstraintModel(
	typ types.Type,
	seen map[types.Type]bool,
) *model {
	typ = types.Unalias(typ)
	if seen[typ] {
		return nil
	}
	seen[typ] = true
	if model := p.modelForType(typ); model != nil && len(model.Variants) > 0 {
		return model
	}
	switch typ := typ.(type) {
	case *types.TypeParam:
		return p.enumConstraintModel(typ.Constraint(), seen)
	case *types.Named:
		if _, ok := typ.Underlying().(*types.Interface); ok {
			return p.enumConstraintModel(typ.Underlying(), seen)
		}
	case *types.Interface:
		for index := 0; index < typ.NumEmbeddeds(); index++ {
			if model := p.enumConstraintModel(typ.EmbeddedType(index), seen); model != nil {
				return model
			}
		}
	}
	return nil
}

func enumReceiverWrite(info *types.Info, target, receiver ast.Expr) bool {
	targetRoot, targetPath, targetOK := enumReceiverPath(info, target)
	receiverRoot, receiverFields, receiverOK := enumReceiverPath(info, receiver)
	if !targetOK || !receiverOK || targetRoot != receiverRoot ||
		len(targetPath) > len(receiverFields) {
		return false
	}
	for index := range targetPath {
		if targetPath[index] != receiverFields[index] {
			return false
		}
	}
	return true
}

func sameEnumReceiver(info *types.Info, left, right ast.Expr) bool {
	leftRoot, leftPath, leftOK := enumReceiverPath(info, left)
	rightRoot, rightPath, rightOK := enumReceiverPath(info, right)
	if !leftOK || !rightOK || leftRoot != rightRoot || len(leftPath) != len(rightPath) {
		return false
	}
	for index := range leftPath {
		if leftPath[index] != rightPath[index] {
			return false
		}
	}
	return true
}

func enumReceiverPath(
	info *types.Info,
	expression ast.Expr,
) (types.Object, []types.Object, bool) {
	switch expression := expression.(type) {
	case *ast.Ident:
		object := info.ObjectOf(expression)
		return object, nil, object != nil
	case *ast.SelectorExpr:
		root, path, ok := enumReceiverPath(info, expression.X)
		if !ok {
			return nil, nil, false
		}
		selection := info.Selections[expression]
		if selection == nil || selection.Kind() != types.FieldVal {
			return nil, nil, false
		}
		return root, append(path, selection.Obj()), true
	case *ast.ParenExpr:
		return enumReceiverPath(info, expression.X)
	case *ast.StarExpr:
		return enumReceiverPath(info, expression.X)
	case *ast.UnaryExpr:
		if expression.Op == token.AND {
			return enumReceiverPath(info, expression.X)
		}
	}
	return nil, nil, false
}
