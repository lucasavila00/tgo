package unicodecomments

var values = []struct {
	text string
	ok   bool
}{
	{"ascii", true},
	{"имя", false}, // Cyrillic text has one display cell per rune.
	{"валю", true}, // Keep the comment in the same tabwriter column.
}
