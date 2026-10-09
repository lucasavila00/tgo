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
