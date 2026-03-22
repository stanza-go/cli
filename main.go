package main

import (
	"fmt"
	"os"

	"github.com/stanza-go/framework/pkg/cmd"
)

var version = "dev"

func main() {
	app := cmd.New("stanza",
		cmd.WithVersion(version),
		cmd.WithDescription("Stanza CLI — project management for Stanza applications"),
	)

	app.Command("export", "Export the data directory as a zip archive", runExport,
		cmd.StringFlag("output", "", "Output file path (default: stanza-export-{timestamp}.zip)"),
		cmd.StringFlag("data-dir", "", "Data directory path (default: ~/.stanza/ or DATA_DIR env)"),
	)

	app.Command("import", "Import a zip archive into the data directory", runImport,
		cmd.StringFlag("data-dir", "", "Data directory path (default: ~/.stanza/ or DATA_DIR env)"),
		cmd.BoolFlag("force", false, "Skip confirmation prompt"),
	)

	app.Command("logs", "View and tail structured log files", runLogs,
		cmd.StringFlag("data-dir", "", "Data directory path (default: ~/.stanza/ or DATA_DIR env)"),
		cmd.IntFlag("lines", 50, "Number of lines to show"),
		cmd.BoolFlag("follow", false, "Follow new log entries (like tail -f)"),
		cmd.StringFlag("level", "", "Minimum log level filter (debug, info, warn, error)"),
		cmd.StringFlag("file", "", "Log file to read (default: stanza.log)"),
		cmd.BoolFlag("json", false, "Output raw JSON instead of pretty-printed"),
		cmd.BoolFlag("no-color", false, "Disable colored output"),
		cmd.BoolFlag("list", false, "List available log files"),
	)

	app.Command("status", "Show data directory health summary", runStatus,
		cmd.StringFlag("data-dir", "", "Data directory path (default: ~/.stanza/ or DATA_DIR env)"),
		cmd.BoolFlag("no-color", false, "Disable colored output"),
	)

	app.Command("db", "Show database statistics and table information", runDB,
		cmd.StringFlag("data-dir", "", "Data directory path (default: ~/.stanza/ or DATA_DIR env)"),
		cmd.BoolFlag("no-color", false, "Disable colored output"),
	)

	app.Command("backup", "Create a consistent database backup (VACUUM INTO)", runBackup,
		cmd.StringFlag("output", "", "Output file path (default: stanza-backup-{timestamp}.sqlite)"),
		cmd.StringFlag("data-dir", "", "Data directory path (default: ~/.stanza/ or DATA_DIR env)"),
		cmd.BoolFlag("compress", false, "Compress the backup with gzip"),
	)

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
