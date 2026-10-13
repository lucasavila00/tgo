package scanner

import "go/token"

// ShortReturn identifies .. without changing Go's ellipsis token.
const ShortReturn token.Token = -1
