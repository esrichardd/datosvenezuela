package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDirOutputCreate(t *testing.T) {
	dir := t.TempDir()
	o := &dirOutput{dir: dir, baseURL: "https://example.com/data-1"}

	got, err := o.create("hello.txt", "text/plain", func(w io.Writer) error {
		_, err := io.WriteString(w, "hello\n")
		return err
	})
	if err != nil {
		t.Fatalf("create() error = %v", err)
	}

	sum := sha256.Sum256([]byte("hello\n"))
	want := File{
		Name:      "hello.txt",
		Mediatype: "text/plain",
		Bytes:     6,
		SHA256:    hex.EncodeToString(sum[:]),
		URL:       "https://example.com/data-1/hello.txt",
	}
	if got != want {
		t.Errorf("create() = %+v, want %+v", got, want)
	}
	content, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil || string(content) != "hello\n" {
		t.Errorf("hello.txt = %q, %v", content, err)
	}
}

func TestDirOutputCreateErrors(t *testing.T) {
	o := &dirOutput{dir: t.TempDir()}

	_, err := o.create("a.txt", "text/plain", func(io.Writer) error { return errWrite })
	if !errors.Is(err, errWrite) {
		t.Errorf("create() with a failing write: error = %v, want %v", err, errWrite)
	}

	_, err = o.create(filepath.Join("missing", "a.txt"), "text/plain", func(io.Writer) error { return nil })
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("create() in a missing directory: error = %v, want %v", err, os.ErrNotExist)
	}
}
