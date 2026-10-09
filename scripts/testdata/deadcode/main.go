package main

func main() {
	live()
	platformOnly()
}

func live() {}

func testOnly() {}

// PublicAPI is an approved public API exclusion.
func PublicAPI() {}

func deadGo() {}

type deadGoType struct{}

var deadGoVar int

const deadGoConst = 1

type testOnlyType struct{}

var testOnlyVar testOnlyType

const testOnlyConst = 1

// PublicType is an approved public type exclusion.
type PublicType struct{}

// PublicVar is an approved public variable exclusion.
var PublicVar int

// PublicConst is an approved public constant exclusion.
const PublicConst = 1
