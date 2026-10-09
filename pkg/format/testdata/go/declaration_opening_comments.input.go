package openingcomments

import ( // imports
	"fmt"
)

const ( // value constants
	first = 1
)

var ( /* block variables */
	second = 2
)

type ( // named types
	third int
)

var _ = fmt.Sprint(first, second, third(3))
