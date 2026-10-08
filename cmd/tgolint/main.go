// Command tgolint checks Go callers of generated tgo types.
package main

import (
	"tgo/internal/tgolint"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(tgolint.Analyzer)
}
