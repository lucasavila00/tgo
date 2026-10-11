package driver

import "testing"

func TestGeneratedName(t *testing.T) {
	for source, want := range map[string]string{
		"main.tgo": "main_tgo.go", "main_test.tgo": "main_tgo_test.go",
		"sub/name_linux_amd64_test.tgo": "sub/name_tgo_linux_amd64_test.go",
		"name_windows.tgo":              "name_tgo_windows.go", "name_arm64.tgo": "name_tgo_arm64.go",
		"linux.tgo": "linux_tgo.go", "name_nacl.tgo": "name_tgo_nacl.go",
		"name_armbe.tgo": "name_tgo_armbe.go", "name_linux_extra.tgo": "name_linux_extra_tgo.go",
	} {
		t.Run(source, func(t *testing.T) {
			if got := generatedName(source); got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}
