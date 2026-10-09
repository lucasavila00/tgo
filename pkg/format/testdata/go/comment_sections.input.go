package sections

const (
	shortName      int = 1 /* first */
	muchLongerName     = 2 /* second */
	finalName          = 3 /* a multiline comment
	continues here */
	reset int = 4 /* reset */
)

func first() {
	println("short")
} // aligned end
func second() {
	println("a much longer body")
} // aligned end

func binary(first, second, third int) int {
	return first |
		second |
		// Keep the operator before this comment.
		third
}

func cases(value int) {
	switch value {
	case 1, // one
		22: // twenty-two
	}
}
