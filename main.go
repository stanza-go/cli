package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("stanza — project management CLI")
		fmt.Println()
		fmt.Println("Usage: stanza <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  export    Export the data directory as a zip archive")
		fmt.Println("  import    Restore from a zip archive")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "export":
		fmt.Println("stanza export: not yet implemented")
	case "import":
		fmt.Println("stanza import: not yet implemented")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
