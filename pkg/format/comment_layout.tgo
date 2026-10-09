package format

import (
	"go/token"
	"strings"
)

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
