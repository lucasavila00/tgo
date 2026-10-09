package comments

const (
	header = "a" +
		"bb" + // first part
		"c" // last part
	palette = "dddd" // palette
)

func assignments() {
	var (
		short any
		typed interface {
			String() string
		} = value
		longer = value
	)
}

const (
	flags = first | /* first flag */
		second | /* second flag */
		last /* last flag */
)

var firstLine = lineNumber() // 0
var (                        // 1
	lineVar = lineNumber() // 2
)                        // 3
var composite = []int{ // 4
	lineNumber(), // 5
}                     // 6
var sum = lineNumber() + // 7
	lineNumber() // 8
func next() { // 9
}
