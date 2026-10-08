package compiler

import (
	"go/ast"
	"slices"
	"strings"
)

func applyEdits(input string, edits []edit) string {
	slices.SortStableFunc(edits, compareEdits)
	var output strings.Builder
	previous := 0
	for _, change := range edits {
		output.WriteString(input[previous:change.start])
		output.WriteString(change.text)
		previous = change.end
	}
	output.WriteString(input[previous:])
	return output.String()
}

func compareEdits(left edit, right edit) int {
	return left.start - right.start
}

func removeLineDirectives(file *ast.File) {
	groups := file.Comments[:0]
	for _, group := range file.Comments {
		comments := group.List[:0]
		for _, comment := range group.List {
			lineComment := strings.HasPrefix(comment.Text, "//line ")
			blockComment := strings.HasPrefix(comment.Text, "/*line ")
			if !lineComment && !blockComment {
				comments = append(comments, comment)
			}
		}
		group.List = comments
		if len(group.List) > 0 {
			groups = append(groups, group)
		}
	}
	file.Comments = groups
}
