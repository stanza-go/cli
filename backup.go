package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/stanza-go/framework/pkg/cmd"
	"github.com/stanza-go/framework/pkg/sqlite"
)

// runBackup creates a consistent database backup using VACUUM INTO.
func runBackup(ctx *cmd.Context) error {
	dataDir, err := resolveDataDir(ctx.String("data-dir"))
	if err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "database.sqlite")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("database not found: %s", dbPath)
	}

	compress := ctx.Bool("compress")

	// Determine output path.
	output := ctx.String("output")
	if output == "" {
		ts := time.Now().Format("20060102-150405")
		if compress {
			output = fmt.Sprintf("stanza-backup-%s.sqlite.gz", ts)
		} else {
			output = fmt.Sprintf("stanza-backup-%s.sqlite", ts)
		}
	}

	// Make output path absolute relative to cwd.
	if !filepath.IsAbs(output) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		output = filepath.Join(cwd, output)
	}

	// Open database (VACUUM INTO needs a read-write connection).
	db := sqlite.New(dbPath)
	if err := db.Start(context.Background()); err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Stop(context.Background())

	// Get source size for reporting.
	srcInfo, _ := os.Stat(dbPath)
	srcSize := srcInfo.Size()

	fmt.Printf("Backing up %s (%s)\n", dbPath, formatBytes(srcSize))

	if compress {
		// VACUUM INTO a temp file, then gzip to the output path.
		return backupCompressed(db, output, srcSize)
	}

	// Direct VACUUM INTO.
	start := time.Now()
	if err := db.Backup(output); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	elapsed := time.Since(start)

	fi, err := os.Stat(output)
	if err != nil {
		return fmt.Errorf("stat backup: %w", err)
	}

	fmt.Printf("Done. %s → %s (compacted) in %s\n",
		formatBytes(srcSize), formatBytes(fi.Size()), elapsed.Round(time.Millisecond))
	fmt.Printf("Saved to %s\n", output)

	return nil
}

// backupCompressed creates a VACUUM INTO backup, then gzip-compresses the result.
func backupCompressed(db *sqlite.DB, output string, srcSize int64) error {
	// Create a temp file for the intermediate uncompressed backup.
	tmpFile, err := os.CreateTemp("", "stanza-backup-*.sqlite")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	start := time.Now()

	// VACUUM INTO the temp file.
	if err := db.Backup(tmpPath); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	tmpInfo, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("stat temp backup: %w", err)
	}
	uncompressedSize := tmpInfo.Size()

	// Gzip the temp file to the output path.
	src, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open temp backup: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer dst.Close()

	gz, err := gzip.NewWriterLevel(dst, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	gz.Name = filepath.Base(output[:len(output)-3]) // strip .gz for the gzip header name
	gz.ModTime = time.Now()

	if _, err := io.Copy(gz, src); err != nil {
		gz.Close()
		return fmt.Errorf("compress backup: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("finalize gzip: %w", err)
	}

	elapsed := time.Since(start)

	fi, err := os.Stat(output)
	if err != nil {
		return fmt.Errorf("stat output: %w", err)
	}

	fmt.Printf("Done. %s → %s (compacted) → %s (gzip) in %s\n",
		formatBytes(srcSize), formatBytes(uncompressedSize), formatBytes(fi.Size()), elapsed.Round(time.Millisecond))
	fmt.Printf("Saved to %s\n", output)

	return nil
}
