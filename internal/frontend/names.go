package frontend

import "fmt"

// freshMarkerName reserves a name across all files in the compilation.
func freshMarkerName(used map[string]bool, prefix string) string {
	for number := 1; ; number++ {
		name := fmt.Sprintf("%s%d", prefix, number)
		if !used[name] {
			used[name] = true
			return name
		}
	}
}
