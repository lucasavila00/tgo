package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesAndListsFiles(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "sample.tgo")
	input := []byte("package sample\nfunc value()int{return 1}\n")
	if err := os.WriteFile(path, input, 0o640); err != nil {
		t.Fatal(err)
	}
	output := new(bytes.Buffer)
	err := run([]string{path}, false, true, strings.NewReader(""), output)
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != path+"\n" {
		t.Fatalf("listed files = %q, want %q", output.String(), path+"\n")
	}
	if err := run([]string{path}, true, false, strings.NewReader(""), output); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc value() int {\n\treturn 1\n}\n"
	if string(got) != want {
		t.Fatalf("written source:\n%s\nwant:\n%s", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("written mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestRunFormatsStandardInput(t *testing.T) {
	t.Parallel()
	input := "package sample\nfunc value()int{return 1}\n"
	output := new(bytes.Buffer)
	if err := run(nil, false, false, strings.NewReader(input), output); err != nil {
		t.Fatal(err)
	}
	want := "package sample\n\nfunc value() int {\n\treturn 1\n}\n"
	if output.String() != want {
		t.Fatalf("standard output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestRunListsChangedStandardInput(t *testing.T) {
	t.Parallel()
	output := new(bytes.Buffer)
	input := strings.NewReader("package   sample\n")
	if err := run(nil, false, true, input, output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "<standard input>\n" {
		t.Fatalf("listed input = %q", output.String())
	}
}

func TestRunDoesNotWriteMalformedFile(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "bad.tgo")
	input := []byte("package sample\nfunc {")
	if err := os.WriteFile(path, input, 0o600); err != nil {
		t.Fatal(err)
	}
	err := run([]string{path}, true, false, strings.NewReader(""), new(bytes.Buffer))
	if err == nil {
		t.Fatal("write accepted malformed input")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, input) {
		t.Fatalf("malformed file changed to %q", got)
	}
}

func TestRunRejectsWriteForStandardInput(t *testing.T) {
	t.Parallel()
	err := run(nil, true, false, strings.NewReader("package sample\n"), new(bytes.Buffer))
	if err == nil {
		t.Fatal("write accepted standard input")
	}
}
