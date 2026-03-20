package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/stanza-go/framework/pkg/cmd"
)

// runExport creates a zip archive of the data directory.
func runExport(ctx *cmd.Context) error {
	dataDir, err := resolveDataDir(ctx.String("data-dir"))
	if err != nil {
		return err
	}

	info, err := os.Stat(dataDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("data directory not found: %s", dataDir)
	}

	output := ctx.String("output")
	if output == "" {
		ts := time.Now().Format("20060102-150405")
		output = fmt.Sprintf("stanza-export-%s.zip", ts)
	}

	// Make output path absolute relative to cwd.
	if !filepath.IsAbs(output) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		output = filepath.Join(cwd, output)
	}

	fmt.Printf("Exporting %s → %s\n", dataDir, output)

	n, totalBytes, err := createZip(output, dataDir)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}

	// Report final size.
	fi, _ := os.Stat(output)
	archiveSize := fi.Size()

	fmt.Printf("Done. %d files (%s) → %s archive\n",
		n, formatBytes(totalBytes), formatBytes(archiveSize))

	return nil
}

// createZip walks the data directory and writes all files into a zip archive.
// It returns the number of files added and total uncompressed bytes.
func createZip(dest, srcDir string) (int, int64, error) {
	f, err := os.Create(dest)
	if err != nil {
		return 0, 0, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	var count int
	var totalBytes int64

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories — they're created implicitly by file paths.
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("file header: %w", err)
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := w.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("create entry: %w", err)
		}

		src, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer src.Close()

		n, err := io.Copy(writer, src)
		if err != nil {
			return fmt.Errorf("write file: %w", err)
		}

		totalBytes += n
		count++
		fmt.Printf("  + %s (%s)\n", relPath, formatBytes(n))

		return nil
	})

	return count, totalBytes, err
}

// formatBytes returns a human-readable byte size.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
