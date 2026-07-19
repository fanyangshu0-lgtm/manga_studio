package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func migrateMySQL(ctx context.Context, db *sql.DB) error {
	var locked int
	if err := db.QueryRowContext(ctx, "SELECT GET_LOCK('manga_drama_studio_migrations', 30)").Scan(&locked); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	if locked != 1 {
		return fmt.Errorf("acquire migration lock: timed out")
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), "SELECT RELEASE_LOCK('manga_drama_studio_migrations')")
	}()

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		applied_at DATETIME(6) NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create schema migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versionText := strings.SplitN(entry.Name(), "_", 2)[0]
		version, parseErr := strconv.ParseInt(versionText, 10, 64)
		if parseErr != nil {
			return fmt.Errorf("invalid migration filename %s", entry.Name())
		}
		var exists int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if exists > 0 {
			continue
		}
		contents, readErr := migrationFiles.ReadFile("migrations/" + entry.Name())
		if readErr != nil {
			return fmt.Errorf("read migration %d: %w", version, readErr)
		}
		tx, txErr := db.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("begin migration %d: %w", version, txErr)
		}
		for _, statement := range strings.Split(string(contents), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, txErr = tx.ExecContext(ctx, statement); txErr != nil {
				_ = tx.Rollback()
				return fmt.Errorf("apply migration %d: %w", version, txErr)
			}
		}
		if _, txErr = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, applied_at) VALUES (?, UTC_TIMESTAMP(6))", version); txErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, txErr)
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("commit migration %d: %w", version, txErr)
		}
	}
	return nil
}
