package compiler

import (
	"go/ast"
	"go/token"
)

type controlExit struct {
	token token.Token
	label string
}

type controlFlow struct {
	next  bool
	exits []controlExit
}

// loopPostReachable reports whether one body path reaches the loop post.
func loopPostReachable(body *ast.BlockStmt, label string) bool {
	flow := controlBlock(body.List)
	return flow.next || flow.hasExit(token.CONTINUE, label)
}

// controlBlock follows reachable statements through one block.
func controlBlock(statements []ast.Stmt) controlFlow {
	flow := controlFlow{next: true}
	for _, statement := range statements {
		if !flow.next {
			break
		}
		next := controlStatement(statement, "")
		flow.exits = append(flow.exits, next.exits...)
		flow.next = next.next
	}
	return flow
}

// controlStatement gets normal and branch exits from one statement.
func controlStatement(statement ast.Stmt, label string) controlFlow {
	switch statement := statement.(type) {
	case *ast.BlockStmt:
		return controlBlock(statement.List)
	case *ast.LabeledStmt:
		return controlStatement(statement.Stmt, statement.Label.Name)
	case *ast.IfStmt:
		return controlIf(statement)
	case *ast.ForStmt:
		return controlFor(statement, label)
	case *ast.RangeStmt:
		return controlRange(statement, label)
	case *ast.SwitchStmt:
		return controlSwitch(statement.Body.List, label)
	case *ast.TypeSwitchStmt:
		return controlSwitch(statement.Body.List, label)
	case *ast.SelectStmt:
		return controlSelect(statement, label)
	case *ast.ReturnStmt:
		return controlFlow{}
	case *ast.BranchStmt:
		return controlBranch(statement)
	default:
		return controlFlow{next: true}
	}
}

// controlIf merges the exits from both possible branches.
func controlIf(statement *ast.IfStmt) controlFlow {
	trueFlow := controlBlock(statement.Body.List)
	falseFlow := controlFlow{next: true}
	if statement.Else != nil {
		falseFlow = controlStatement(statement.Else, "")
	}
	return mergeControl(trueFlow, falseFlow)
}

// controlFor consumes branches that target one for statement.
func controlFor(statement *ast.ForStmt, label string) controlFlow {
	body := controlBlock(statement.Body.List)
	exits, broke := body.consume(token.BREAK, label)
	exits, _ = exits.consume(token.CONTINUE, label)
	return controlFlow{
		next:  statement.Cond != nil || broke,
		exits: exits.exits,
	}
}

// controlRange consumes branches that target one range statement.
func controlRange(statement *ast.RangeStmt, label string) controlFlow {
	body := controlBlock(statement.Body.List)
	exits, _ := body.consume(token.BREAK, label)
	exits, _ = exits.consume(token.CONTINUE, label)
	return controlFlow{next: true, exits: exits.exits}
}

// controlSwitch merges case exits and consumes breaks for this switch.
func controlSwitch(clauses []ast.Stmt, label string) controlFlow {
	branches := make([]controlFlow, len(clauses))
	for index := len(clauses) - 1; index >= 0; index-- {
		clause := clauses[index].(*ast.CaseClause)
		branch := controlBlock(clause.Body)
		branch, fallsThrough := branch.consume(token.FALLTHROUGH, "")
		if fallsThrough && index+1 < len(branches) {
			branch = mergeControl(branch, branches[index+1])
		}
		branches[index] = branch
	}

	flow := controlFlow{}
	hasDefault := false
	for index, item := range clauses {
		clause := item.(*ast.CaseClause)
		hasDefault = hasDefault || len(clause.List) == 0
		branch := branches[index]
		branch, broke := branch.consume(token.BREAK, label)
		flow.next = flow.next || branch.next || broke
		flow.exits = append(flow.exits, branch.exits...)
	}
	flow.next = flow.next || !hasDefault
	return flow
}

// controlSelect merges clause exits and consumes breaks for this select.
func controlSelect(statement *ast.SelectStmt, label string) controlFlow {
	flow := controlFlow{}
	for _, item := range statement.Body.List {
		clause := item.(*ast.CommClause)
		branch := controlBlock(clause.Body)
		branch, broke := branch.consume(token.BREAK, label)
		flow.next = flow.next || branch.next || broke
		flow.exits = append(flow.exits, branch.exits...)
	}
	return flow
}

// controlBranch turns one branch statement into a targeted exit.
func controlBranch(statement *ast.BranchStmt) controlFlow {
	if statement.Tok == token.GOTO {
		return controlFlow{}
	}
	label := ""
	if statement.Label != nil {
		label = statement.Label.Name
	}
	return controlFlow{exits: []controlExit{{token: statement.Tok, label: label}}}
}

// mergeControl joins alternative control paths.
func mergeControl(left, right controlFlow) controlFlow {
	exits := make([]controlExit, 0, len(left.exits)+len(right.exits))
	exits = append(exits, left.exits...)
	exits = append(exits, right.exits...)
	return controlFlow{next: left.next || right.next, exits: exits}
}

// consume removes exits for one statement and reports whether one existed.
func (flow controlFlow) consume(kind token.Token, label string) (controlFlow, bool) {
	found := false
	exits := flow.exits[:0]
	for _, exit := range flow.exits {
		matches := exit.token == kind && (exit.label == "" || exit.label == label)
		if matches {
			found = true
			continue
		}
		exits = append(exits, exit)
	}
	flow.exits = exits
	return flow, found
}

// hasExit reports whether a branch targets the selected statement.
func (flow controlFlow) hasExit(kind token.Token, label string) bool {
	for _, exit := range flow.exits {
		if exit.token == kind && (exit.label == "" || exit.label == label) {
			return true
		}
	}
	return false
}
