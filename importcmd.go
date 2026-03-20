package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanza-go/framework/pkg/cmd"
)

// runImport extracts a zip archive into the data directory.
func runImport(ctx *cmd.Context) error {
	if ctx.Arg(0) == "" {
		return fmt.Errorf("usage: stanza import <archive.zip>")
	}

	archivePath := ctx.Arg(0)
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("archive not found: %s", archivePath)
	}

	dataDir, err := resolveDataDir(ctx.String("data-dir"))
	if err != nil {
		return err
	}

	// Validate archive looks like a stanza export.
	if err := validateArchive(archivePath); err != nil {
		return err
	}

	// Confirm before overwriting unless --force.
	if !ctx.Bool("force") {
		if _, err := os.Stat(dataDir); err == nil {
			fmt.Printf("WARNING: This will overwrite the data directory at:\n  %s\n\n", dataDir)
			fmt.Print("Continue? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Ensure data dir exists.
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	fmt.Printf("Importing %s → %s\n", archivePath, dataDir)

	n, totalBytes, err := extractZip(archivePath, dataDir)
	if err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	fmt.Printf("Done. %d files extracted (%s)\n", n, formatBytes(totalBytes))

	return nil
}

// validateArchive checks that the zip contains database.sqlite, which
// is the marker for a valid stanza data directory export.
func validateArchive(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "database.sqlite" {
			return nil
		}
	}

	return fmt.Errorf("archive does not contain database.sqlite — not a valid stanza export")
}

// extractZip extracts all files from a zip archive into the destination
// directory. Returns the number of files extracted and total bytes written.
func extractZip(archivePath, destDir string) (int, int64, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return 0, 0, fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()

	var count int
	var totalBytes int64

	for _, f := range r.File {
		destPath := filepath.Join(destDir, f.Name)

		// Guard against zip slip: ensure the destination is within destDir.
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return count, totalBytes, fmt.Errorf("illegal path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return count, totalBytes, fmt.Errorf("create directory: %w", err)
			}
			continue
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return count, totalBytes, fmt.Errorf("create parent directory: %w", err)
		}

		n, err := extractFile(f, destPath)
		if err != nil {
			return count, totalBytes, err
		}

		totalBytes += n
		count++
		fmt.Printf("  + %s (%s)\n", f.Name, formatBytes(n))
	}

	return count, totalBytes, nil
}

// extractFile writes a single zip entry to the given destination path.
func extractFile(f *zip.File, destPath string) (int64, error) {
	src, err := f.Open()
	if err != nil {
		return 0, fmt.Errorf("open entry %s: %w", f.Name, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return 0, fmt.Errorf("create file %s: %w", destPath, err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("write file %s: %w", destPath, err)
	}

	return n, nil
}
