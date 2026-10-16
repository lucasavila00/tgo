package sourcefile

import "testing"

// TestInputName checks platform, test, and repeated marker names in both directions.
func TestInputName(t *testing.T) {
	for _, input := range []string{
		"main.tgo", "main_test.tgo", "sub/name_linux_amd64_test.tgo",
		"name_windows.tgo", "name_arm64.tgo", "linux.tgo", "name_nacl.tgo",
		"name_armbe.tgo", "name_linux_extra.tgo", "name_tgo_linux_test.tgo",
		"name_tgo_tgo.tgo", "name_tgo_extra.tgo",
	} {
		if output := GeneratedName(input); InputName(output) != input {
			t.Errorf("InputName(%q) = %q; want %q", output, InputName(output), input)
		}
	}
	for _, output := range []string{"main.go", "main_tgo_extra.go", "main_tgo.go.bak", "main_tgo_test_linux.go"} {
		if input := InputName(output); input != "" {
			t.Errorf("InputName(%q) = %q; want empty", output, input)
		}
	}
}
