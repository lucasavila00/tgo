package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesAndListsFiles(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "sample.tgo")
	input := []byte("package sample\nfunc value()int{return 1}\n")
	if err := os.WriteFile(path, input, 0o640); err != nil {
		t.Fatal(err)
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	standardOutput := os.Stdout
	os.Stdout = write
	err = run([]string{path}, false, true)
	closeErr := write.Close()
	os.Stdout = standardOutput
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	listed, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if string(listed) != path+"\n" {
		t.Fatalf("listed files = %q, want %q", listed, path+"\n")
	}
	if err := run([]string{path}, true, false); err != nil {
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
