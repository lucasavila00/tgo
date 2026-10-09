package placement

func use(...int) {}

func placement(value, other int) int {
target: // keep this label comment
	use(*(func(value int) *int { return &value })(value+1) + other)
	if value > 0 {
		return value /* first */ /* second */
	}
	return 0 /* closing comment */
}

var table = "first" +
	"second"
	// table size
