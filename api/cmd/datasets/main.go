// Command datasets builds the downloadable files of each dataset in the data
// directory, for a data release:
//
//   - <dataset>.zip: the CSV files and their datapackage.json manifest;
//   - <dataset>.json: every table as an array of objects;
//   - <dataset>.sql: a PostgreSQL script that creates and fills the tables;
//   - index.json: the release manifest, with every dataset, its tables and
//     row counts, and its files with size, SHA-256 and download URL;
//   - SHA256SUMS: the checksums of every file, for sha256sum -c.
//
// Every dataset is validated against its manifest first, so invalid data is
// never published.
//
// Usage:
//
//	go run ./cmd/datasets -version 2026.11.21 [-data ../data] [-out ../dist]
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

// releaseURL is where the files of a release are downloaded from. Releases
// are tagged data-<version>.
const releaseURL = "https://github.com/esrichardd/datosvenezuela/releases/download/data-"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "datasets:", err)
		os.Exit(1)
	}
}

// run builds the release files as the command-line arguments say.
func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("datasets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "release version, as YYYY.MM.DD (required)")
	dataDir := fs.String("data", "../data", "directory with one folder per dataset")
	outDir := fs.String("out", "../dist", "directory to create for the release files; it must not exist")
	baseURL := fs.String("base-url", "", "URL the files are downloaded from (default: the GitHub release of the version)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *version == "" {
		return errors.New("missing -version")
	}
	date, err := time.Parse("2006.01.02", *version)
	if err != nil {
		return fmt.Errorf("invalid -version %q: want YYYY.MM.DD", *version)
	}
	// Noon UTC is still the same day in almost every time zone, so tools
	// that show the time of the files in local time show the release date.
	date = date.Add(12 * time.Hour)
	if *baseURL == "" {
		*baseURL = releaseURL + *version
	}

	// Creating the directory, instead of reusing one, guarantees that it only
	// holds the files of this release.
	if err := os.Mkdir(*outDir, 0o750); err != nil {
		return err
	}
	out := &dirOutput{dir: *outDir, baseURL: *baseURL}
	files, err := build(*dataDir, *version, date, out)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "wrote %d files to %s\n", len(files), *outDir)
	return err
}
