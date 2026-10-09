package format

import (
	"bytes"
	"go/token"
	"strings"
)

func (p *printer) commentAlignment(position token.Pos) (int, bool) {
	if column, ok := p.fixedCommentColumns[position]; ok {
		return column, true
	}
	column, ok := p.commentColumns[position]
	return column, ok
}

func (p *printer) sourceWhitespaceBetween(stop token.Pos, start token.Pos) bool {
	file := p.files.File(stop)
	if file == nil || p.files.File(start) != file {
		return false
	}
	left := file.Offset(stop)
	right := file.Offset(start)
	return 0 <= left && left <= right && right <= len(p.source) &&
		strings.TrimSpace(string(p.source[left:right])) == ""
}

func (p *printer) sourceLineEndsAt(position token.Pos) bool {
	file := p.files.File(position)
	if file == nil {
		return false
	}
	offset := file.Offset(position)
	if offset < 0 || offset > len(p.source) {
		return false
	}
	end := bytes.IndexByte(p.source[offset:], '\n')
	if end < 0 {
		end = len(p.source)
	} else {
		end += offset
	}
	return strings.TrimSpace(string(p.source[offset:end])) == ""
}

func (p *printer) trailingContinuationComments(position token.Pos) {
	line := p.position(position).Line
	for p.comment < len(p.comments) {
		comment := p.comments[p.comment]
		if p.position(comment.start).Line != line+1 ||
			p.sourceIndent(comment.start) < p.indent ||
			p.sourceBlankBetween(position, comment.start) {
			return
		}
		p.before(comment.stop + 1)
		position = comment.stop
		line = p.position(position).Line
	}
}
