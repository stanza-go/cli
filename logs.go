package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stanza-go/framework/pkg/cmd"
)

// logLevels maps level names to numeric severity for filtering.
var logLevels = map[string]int{
	"debug": 0,
	"info":  1,
	"warn":  2,
	"error": 3,
}

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// runLogs tails structured JSON log files from the data directory.
func runLogs(ctx *cmd.Context) error {
	dataDir, err := resolveDataDir(ctx.String("data-dir"))
	if err != nil {
		return err
	}

	logsDir := filepath.Join(dataDir, "logs")
	if info, err := os.Stat(logsDir); err != nil || !info.IsDir() {
		return fmt.Errorf("logs directory not found: %s", logsDir)
	}

	follow := ctx.Bool("follow")
	lines := ctx.Int("lines")
	if lines <= 0 {
		lines = 50
	}
	minLevel := strings.ToLower(ctx.String("level"))
	rawJSON := ctx.Bool("json")
	listFiles := ctx.Bool("list")
	noColor := ctx.Bool("no-color")

	// List available log files and exit.
	if listFiles {
		return printLogFiles(logsDir)
	}

	// Determine which log file to read.
	logFile := ctx.String("file")
	if logFile == "" {
		logFile = "stanza.log"
	}
	logPath := filepath.Join(logsDir, logFile)

	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("log file not found: %s", logPath)
	}

	// Validate level filter.
	if minLevel != "" {
		if _, ok := logLevels[minLevel]; !ok {
			return fmt.Errorf("invalid level %q (use: debug, info, warn, error)", minLevel)
		}
	}

	printer := &logPrinter{
		rawJSON:  rawJSON,
		noColor:  noColor,
		minLevel: minLevel,
	}

	// Print the last N lines.
	if err := tailLines(logPath, lines, printer); err != nil {
		return err
	}

	// If --follow, keep watching for new lines.
	if follow {
		return followLog(logPath, printer)
	}

	return nil
}

// printLogFiles lists all log files in the logs directory with sizes.
func printLogFiles(logsDir string) error {
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return fmt.Errorf("read logs directory: %w", err)
	}

	type logFileInfo struct {
		name string
		size int64
		mod  time.Time
	}

	var files []logFileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, logFileInfo{
			name: e.Name(),
			size: info.Size(),
			mod:  info.ModTime(),
		})
	}

	if len(files) == 0 {
		fmt.Println("No log files found.")
		return nil
	}

	// Sort: current log first, then by name descending (newest rotated first).
	sort.Slice(files, func(i, j int) bool {
		if files[i].name == "stanza.log" {
			return true
		}
		if files[j].name == "stanza.log" {
			return false
		}
		return files[i].name > files[j].name
	})

	fmt.Printf("Log files in %s:\n\n", logsDir)
	for _, f := range files {
		label := f.name
		if f.name == "stanza.log" {
			label += " (current)"
		}
		fmt.Printf("  %-40s %8s   %s\n", label, formatBytes(f.size), f.mod.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// tailLines reads the last N lines from a file and prints them through the printer.
// Lines that don't pass the level filter are not counted toward N.
func tailLines(path string, n int, p *logPrinter) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	// Read all lines (log files are rarely huge — this is simple and correct).
	var allLines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max line
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if p.matchesLevel(line) {
			allLines = append(allLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read log file: %w", err)
	}

	// Take last N lines.
	start := 0
	if len(allLines) > n {
		start = len(allLines) - n
	}

	for _, line := range allLines[start:] {
		p.printLine(line)
	}

	return nil
}

// followLog watches a log file for new content and prints new lines.
func followLog(path string, p *logPrinter) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	// Seek to end.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("seek to end: %w", err)
	}

	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(300 * time.Millisecond)
				continue
			}
			return fmt.Errorf("read log: %w", err)
		}

		line = strings.TrimRight(line, "\n\r")
		if line == "" {
			continue
		}

		if p.matchesLevel(line) {
			p.printLine(line)
		}
	}
}

// logPrinter handles formatting and filtering of log lines.
type logPrinter struct {
	rawJSON  bool
	noColor  bool
	minLevel string
}

// matchesLevel checks if a JSON log line meets the minimum level threshold.
func (p *logPrinter) matchesLevel(line string) bool {
	if p.minLevel == "" {
		return true
	}

	minSev, ok := logLevels[p.minLevel]
	if !ok {
		return true
	}

	// Quick extraction of "level" field without full JSON parse.
	level := extractJSONString(line, "level")
	if level == "" {
		return true // Non-JSON or missing level — include by default.
	}

	lineSev, ok := logLevels[level]
	if !ok {
		return true
	}

	return lineSev >= minSev
}

// printLine formats and prints a single log line.
func (p *logPrinter) printLine(line string) {
	if p.rawJSON {
		fmt.Println(line)
		return
	}

	// Parse JSON.
	var entry map[string]any
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		// Not valid JSON — print as-is.
		fmt.Println(line)
		return
	}

	// Extract core fields.
	ts, _ := entry["time"].(string)
	level, _ := entry["level"].(string)
	msg, _ := entry["msg"].(string)

	// Remove core fields from extra.
	delete(entry, "time")
	delete(entry, "level")
	delete(entry, "msg")

	// Format timestamp.
	if ts != "" {
		// Try to parse and reformat for readability.
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", ts); err == nil {
			ts = t.Local().Format("15:04:05.000")
		} else if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			ts = t.Local().Format("15:04:05.000")
		}
	}

	// Format level with color.
	levelStr := strings.ToUpper(level)
	if !p.noColor {
		levelStr = p.colorLevel(level)
	}

	// Build extra fields string.
	var extras []string
	for k, v := range entry {
		extras = append(extras, fmt.Sprintf("%s=%v", k, formatValue(v)))
	}
	// Sort for deterministic output.
	sort.Strings(extras)

	// Print formatted line.
	if p.noColor {
		if len(extras) > 0 {
			fmt.Printf("%s  %-5s  %s  %s\n", ts, levelStr, msg, strings.Join(extras, " "))
		} else {
			fmt.Printf("%s  %-5s  %s\n", ts, levelStr, msg)
		}
	} else {
		extraStr := ""
		if len(extras) > 0 {
			extraStr = "  " + colorDim + strings.Join(extras, " ") + colorReset
		}
		fmt.Printf("%s%s%s  %s  %s%s\n", colorDim, ts, colorReset, levelStr, msg, extraStr)
	}
}

// colorLevel returns an ANSI-colored level string.
func (p *logPrinter) colorLevel(level string) string {
	upper := strings.ToUpper(level)
	padded := fmt.Sprintf("%-5s", upper)
	switch level {
	case "debug":
		return colorDim + padded + colorReset
	case "info":
		return colorGreen + padded + colorReset
	case "warn":
		return colorYellow + padded + colorReset
	case "error":
		return colorRed + padded + colorReset
	default:
		return padded
	}
}

// formatValue formats a JSON value for display.
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		if strings.ContainsAny(val, " \t\n\"") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case float64:
		// JSON numbers are always float64. Show as int if whole.
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// extractJSONString quickly extracts a string value for a given key from a JSON line.
// This avoids a full JSON parse for level filtering.
func extractJSONString(line, key string) string {
	needle := fmt.Sprintf("%q:", key)
	idx := strings.Index(line, needle)
	if idx < 0 {
		return ""
	}

	// Skip past the key and colon.
	rest := line[idx+len(needle):]

	// Skip whitespace.
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		rest = rest[1:]
	}

	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}

	// Find closing quote (simple — no escaped quotes in level values).
	end := strings.IndexByte(rest[1:], '"')
	if end < 0 {
		return ""
	}

	return rest[1 : end+1]
}
