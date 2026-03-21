package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stanza-go/framework/pkg/cmd"
	"github.com/stanza-go/framework/pkg/sqlite"
)

// runDB prints database statistics and table information.
func runDB(ctx *cmd.Context) error {
	dataDir, err := resolveDataDir(ctx.String("data-dir"))
	if err != nil {
		return err
	}

	noColor := ctx.Bool("no-color")
	p := &statusPrinter{noColor: noColor}

	dbPath := filepath.Join(dataDir, "database.sqlite")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("database not found: %s", dbPath)
	}

	// Open database in read-only mode via query_only pragma.
	db := sqlite.New(dbPath, sqlite.WithPragma("PRAGMA query_only = true"))
	if err := db.Start(context.Background()); err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Stop(context.Background())

	// File sizes.
	p.printHeader("Database Info")
	p.printField("Path", dbPath)

	if fi, err := os.Stat(dbPath); err == nil {
		p.printField("Size", formatBytes(fi.Size()))
	}
	walPath := dbPath + "-wal"
	if fi, err := os.Stat(walPath); err == nil {
		p.printField("WAL size", formatBytes(fi.Size()))
	}
	fmt.Println()

	// SQLite version and settings.
	p.printSection("Engine")
	if version, err := queryString(db, "SELECT sqlite_version()"); err == nil {
		p.printField("  SQLite version", version)
	}
	if mode, err := queryString(db, "PRAGMA journal_mode"); err == nil {
		p.printField("  Journal mode", strings.ToUpper(mode))
	}
	if pageSize, err := queryInt(db, "PRAGMA page_size"); err == nil {
		p.printField("  Page size", formatBytes(int64(pageSize)))
	}
	if pageCount, err := queryInt(db, "PRAGMA page_count"); err == nil {
		p.printField("  Page count", fmt.Sprintf("%d", pageCount))
	}
	if freePages, err := queryInt(db, "PRAGMA freelist_count"); err == nil {
		p.printField("  Free pages", fmt.Sprintf("%d", freePages))
	}
	fmt.Println()

	// Tables and row counts.
	p.printSection("Tables")

	type tableInfo struct {
		name     string
		rowCount int
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	var tables []tableInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, tableInfo{name: name})
	}
	rows.Close()

	// Fetch row counts (separate queries because Rows holds the mutex).
	for i, t := range tables {
		count, err := queryInt(db, fmt.Sprintf("SELECT count(*) FROM \"%s\"", t.name))
		if err == nil {
			tables[i].rowCount = count
		}
	}

	if len(tables) == 0 {
		fmt.Println("  No tables found.")
	} else {
		// Find longest name for alignment.
		maxName := 0
		for _, t := range tables {
			if len(t.name) > maxName {
				maxName = len(t.name)
			}
		}

		for _, t := range tables {
			countStr := fmt.Sprintf("%d rows", t.rowCount)
			if t.rowCount == 1 {
				countStr = "1 row"
			}
			if noColor {
				fmt.Printf("  %-*s  %s\n", maxName, t.name, countStr)
			} else {
				fmt.Printf("  %-*s  %s%s%s\n", maxName, t.name, colorDim, countStr, colorReset)
			}
		}
	}
	fmt.Println()

	// Migration status.
	p.printSection("Migrations")

	// Check if _migrations table exists.
	hasMigrations, _ := queryInt(db, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='_migrations'")
	if hasMigrations == 0 {
		fmt.Println("  No migration history found.")
		return nil
	}

	migrationCount, _ := queryInt(db, "SELECT count(*) FROM _migrations")
	p.printField("  Applied", fmt.Sprintf("%d", migrationCount))

	// Show last 5 migrations.
	type migrationInfo struct {
		version   int64
		name      string
		appliedAt string
	}

	mRows, err := db.Query("SELECT version, name, applied_at FROM _migrations ORDER BY version DESC LIMIT 5")
	if err == nil {
		var migrations []migrationInfo
		for mRows.Next() {
			var m migrationInfo
			if err := mRows.Scan(&m.version, &m.name, &m.appliedAt); err != nil {
				continue
			}
			migrations = append(migrations, m)
		}
		mRows.Close()

		// Reverse to show oldest-first (ascending).
		sort.Slice(migrations, func(i, j int) bool {
			return migrations[i].version < migrations[j].version
		})

		if len(migrations) > 0 {
			if migrationCount > 5 {
				if noColor {
					fmt.Printf("  ... (%d earlier migrations)\n", migrationCount-5)
				} else {
					fmt.Printf("  %s... (%d earlier migrations)%s\n", colorDim, migrationCount-5, colorReset)
				}
			}
			for _, m := range migrations {
				if noColor {
					fmt.Printf("  #%-4d %s  (%s)\n", m.version, m.name, m.appliedAt)
				} else {
					fmt.Printf("  %s#%-4d%s %s  %s(%s)%s\n", colorGreen, m.version, colorReset, m.name, colorDim, m.appliedAt, colorReset)
				}
			}
		}
	}

	return nil
}

// queryString executes a query that returns a single string value.
func queryString(db *sqlite.DB, sql string) (string, error) {
	var val string
	if err := db.QueryRow(sql).Scan(&val); err != nil {
		return "", err
	}
	return val, nil
}

// queryInt executes a query that returns a single integer value.
func queryInt(db *sqlite.DB, sql string) (int, error) {
	var val int
	if err := db.QueryRow(sql).Scan(&val); err != nil {
		return 0, err
	}
	return val, nil
}
