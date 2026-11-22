package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// dirOutput writes the release files to dir. baseURL is where they are
// downloaded from once published.
type dirOutput struct {
	dir     string
	baseURL string
}

func (o *dirOutput) create(name, mediatype string, write func(io.Writer) error) (File, error) {
	// The files are small: building them in memory makes their size and hash
	// simple to get.
	var buf bytes.Buffer
	if err := write(&buf); err != nil {
		return File{}, fmt.Errorf("%s: %w", name, err)
	}
	// The files are public, so everyone may read them.
	if err := os.WriteFile(filepath.Join(o.dir, name), buf.Bytes(), 0o644); err != nil { //nolint:gosec // public release files
		return File{}, err
	}

	sum := sha256.Sum256(buf.Bytes())
	return File{
		Name:      name,
		Mediatype: mediatype,
		Bytes:     buf.Len(),
		SHA256:    hex.EncodeToString(sum[:]),
		URL:       o.baseURL + "/" + name,
	}, nil
}
