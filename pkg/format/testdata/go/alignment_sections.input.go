package alignment

type fields struct {
	first, // first
	second, // second
	last int // last
}

var (
	first,
	second,
	last int
)

var keyed = map[string]bool{
	"x": true,
	"a-key-that-starts-another-alignment-section-because-it-is-long": false,
	"another-long-key-in-that-section": true,
}

var rows = []row{
	{name: "a-row-with-a-name-that-is-long-enough-to-start-a-section", enabled: true}, // long
	{name: "another-row-with-a-name-that-stays-in-the-long-section", enabled: true}, // long
	{name: "x"}, // separate short row
	{name: "y"}, // short row
}

func call() {
	use(
		"short", // short
		"a longer value", // long
	)

	builder().
		With(
			item("first"),
			item("second"))
}

func signature(first, second, third,
	fourth, fifth int) {
	use(first)
}
