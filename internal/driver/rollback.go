package driver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type savedOutput struct {
	path    string
	content []byte
	mode    os.FileMode
	exists  bool
}

// saveOutputs reads owned generated files before compilation changes them.
func saveOutputs(source *moduleFiles) []savedOutput {
	var outputs []savedOutput
	for _, file := range source.files {
		if !strings.HasSuffix(file.name, ".tgo") {
			continue
		}
		output := savedOutput{path: filepath.Join(source.root, generatedName(file.name))}
		content, err := os.ReadFile(output.path)
		if errors.Is(err, os.ErrNotExist) {
			outputs = append(outputs, output)
			continue
		}
		must(err)
		info, err := os.Stat(output.path)
		must(err)
		output.content = content
		output.mode = info.Mode().Perm()
		output.exists = true
		outputs = append(outputs, output)
	}
	return outputs
}

// restoreOutputs restores old files and removes outputs that did not exist.
func restoreOutputs(outputs []savedOutput) error {
	var failures []error
	for _, output := range outputs {
		var err error
		if output.exists {
			err = os.WriteFile(output.path, output.content, output.mode)
			if err == nil {
				err = os.Chmod(output.path, output.mode)
			}
		} else {
			err = os.Remove(output.path)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("restore %s: %w", output.path, err))
		}
	}
	return errors.Join(failures...)
}
