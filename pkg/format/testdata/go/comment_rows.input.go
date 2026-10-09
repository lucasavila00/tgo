package rows

func use(...any) {}

func rows(value int) {
	switch value {
	case 1,
		22,  // twenty-two
		333: // three hundred thirty-three
	}
	use(
		"short",                        // short
		"a much longer argument value") // closing
	_ = []any{
		"short",                             // short
		"a much longer composite row value", // long
	}
}

var (
	short any
	multi interface {
		String() string
	} = struct{ string }{}
	longerName any
)
