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

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
