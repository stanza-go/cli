package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stanza-go/framework/pkg/cmd"
)

// runStatus prints a health summary of the Stanza data directory.
func runStatus(ctx *cmd.Context) error {
	dataDir, err := resolveDataDir(ctx.String("data-dir"))
	if err != nil {
		return err
	}

	noColor := ctx.Bool("no-color")
	p := &statusPrinter{noColor: noColor}

	info, err := os.Stat(dataDir)
	if err != nil || !info.IsDir() {
		p.printHeader("Stanza Status")
		p.printField("Data directory", dataDir)
		p.printStatus("NOT FOUND", "red")
		return nil
	}

	p.printHeader("Stanza Status")
	p.printField("Data directory", dataDir)
	fmt.Println()

	// Database file.
	p.printSection("Database")
	dbPath := filepath.Join(dataDir, "database.sqlite")
	if fi, err := os.Stat(dbPath); err == nil {
		p.printField("  database.sqlite", formatBytes(fi.Size()))
		p.printFieldDim("  Modified", fi.ModTime().Format("2006-01-02 15:04:05"))

		// WAL file.
		walPath := dbPath + "-wal"
		if wfi, err := os.Stat(walPath); err == nil {
			p.printField("  database.sqlite-wal", formatBytes(wfi.Size()))
		} else {
			p.printFieldDim("  database.sqlite-wal", "not present")
		}

		// SHM file.
		shmPath := dbPath + "-shm"
		if sfi, err := os.Stat(shmPath); err == nil {
			p.printField("  database.sqlite-shm", formatBytes(sfi.Size()))
		} else {
			p.printFieldDim("  database.sqlite-shm", "not present")
		}
	} else {
		p.printStatus("  database.sqlite: MISSING", "red")
	}
	fmt.Println()

	// Logs directory.
	p.printSection("Logs")
	logsDir := filepath.Join(dataDir, "logs")
	if fi, err := os.Stat(logsDir); err == nil && fi.IsDir() {
		count, totalSize := dirStats(logsDir, ".log")
		p.printField("  Log files", fmt.Sprintf("%d", count))
		p.printField("  Total size", formatBytes(totalSize))

		// Show most recent log file.
		if newestName, newestMod := newestFile(logsDir, ".log"); newestName != "" {
			p.printFieldDim("  Latest", fmt.Sprintf("%s (%s)", newestName, relativeTime(newestMod)))
		}
	} else {
		p.printFieldDim("  logs/", "not present")
	}
	fmt.Println()

	// Uploads directory.
	p.printSection("Uploads")
	uploadsDir := filepath.Join(dataDir, "uploads")
	if fi, err := os.Stat(uploadsDir); err == nil && fi.IsDir() {
		count, totalSize := dirStatsRecursive(uploadsDir)
		p.printField("  Files", fmt.Sprintf("%d", count))
		p.printField("  Total size", formatBytes(totalSize))
	} else {
		p.printFieldDim("  uploads/", "not present")
	}
	fmt.Println()

	// Backups directory.
	p.printSection("Backups")
	backupsDir := filepath.Join(dataDir, "backups")
	if fi, err := os.Stat(backupsDir); err == nil && fi.IsDir() {
		count, totalSize := dirStatsRecursive(backupsDir)
		p.printField("  Files", fmt.Sprintf("%d", count))
		p.printField("  Total size", formatBytes(totalSize))

		if newestName, newestMod := newestFile(backupsDir, ""); newestName != "" {
			p.printFieldDim("  Latest", fmt.Sprintf("%s (%s)", newestName, relativeTime(newestMod)))
		}
	} else {
		p.printFieldDim("  backups/", "not present")
	}
	fmt.Println()

	// Config file.
	p.printSection("Config")
	configPath := filepath.Join(dataDir, "config.yaml")
	if fi, err := os.Stat(configPath); err == nil {
		p.printField("  config.yaml", formatBytes(fi.Size()))
		p.printFieldDim("  Modified", fi.ModTime().Format("2006-01-02 15:04:05"))
	} else {
		p.printFieldDim("  config.yaml", "not present")
	}

	return nil
}

// dirStats counts files with the given extension in a directory (non-recursive).
func dirStats(dir, ext string) (int, int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}

	var count int
	var totalSize int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext != "" && !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		if fi, err := e.Info(); err == nil {
			count++
			totalSize += fi.Size()
		}
	}
	return count, totalSize
}

// dirStatsRecursive counts all files and total size in a directory tree.
func dirStatsRecursive(dir string) (int, int64) {
	var count int
	var totalSize int64
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		count++
		totalSize += info.Size()
		return nil
	})
	return count, totalSize
}

// newestFile returns the name and mod time of the most recently modified file
// in dir, optionally filtered by extension.
func newestFile(dir, ext string) (string, time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", time.Time{}
	}

	var newest string
	var newestMod time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext != "" && !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if fi.ModTime().After(newestMod) {
			newest = e.Name()
			newestMod = fi.ModTime()
		}
	}
	return newest, newestMod
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// statusPrinter handles formatted output for the status command.
type statusPrinter struct {
	noColor bool
}

func (p *statusPrinter) printHeader(title string) {
	if p.noColor {
		fmt.Printf("=== %s ===\n", title)
	} else {
		fmt.Printf("%s=== %s ===%s\n", colorCyan, title, colorReset)
	}
}

func (p *statusPrinter) printSection(title string) {
	if p.noColor {
		fmt.Printf("[%s]\n", title)
	} else {
		fmt.Printf("%s[%s]%s\n", colorCyan, title, colorReset)
	}
}

func (p *statusPrinter) printField(label, value string) {
	fmt.Printf("%-24s %s\n", label, value)
}

func (p *statusPrinter) printFieldDim(label, value string) {
	if p.noColor {
		fmt.Printf("%-24s %s\n", label, value)
	} else {
		fmt.Printf("%-24s %s%s%s\n", label, colorDim, value, colorReset)
	}
}

func (p *statusPrinter) printStatus(text, color string) {
	if p.noColor {
		fmt.Println(text)
		return
	}
	c := colorReset
	switch color {
	case "red":
		c = colorRed
	case "green":
		c = colorGreen
	case "yellow":
		c = colorYellow
	}
	fmt.Printf("%s%s%s\n", c, text, colorReset)
}
