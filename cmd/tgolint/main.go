// Command tgolint checks TGo source rules and Go use of generated TGo types.
package main

import (
	"tgo/internal/tgolint"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(tgolint.Analyzer)
}
