package listsections

func arguments() {
	f(
		receiver.Handle(), // handle
		nil,               // address
		protection,        // protection
		0,                 // high
		0,                 // low
		nil,               // name
	)
}

func clauses(kind int) {
	switch kind {
	case
		one, two, three, four, five, six, seven, eight, // first group
		nine, ten, eleven, twelve, // second group
		thirteen, fourteen, fifteen: // third group
	}
}

func ratios() {
	f(
		"short", // keep this comment near its value
		fmt.Sprintf("a much longer argument: %d", value), // align separately
	)
}
