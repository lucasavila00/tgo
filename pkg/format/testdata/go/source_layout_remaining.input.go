package layout

type mode int // keep the next declaration attached
const attached mode = 1

func rawLiterals() {
	const first = `
first
`
	// Keep this comment with second.
	const second = `
second
`
	_, _ = first, second
}

func result(a, b uint32) (uint32, bool) {
	return uint32(a)<<16 |
			uint32(b)<<8,
		true
}

func clauses(value int) {
	switch value {
	case
		// first group
		1,  // one
		22, // two
		333:
	case 4: // four
	case 55: // fifty-five
	case 6:
		// body comment

	// disabled case
	case 7:
	}

next:

	use(value)
}
