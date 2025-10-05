// ABOUTME: SQLite storage implementation for distributed SQLite system
// ABOUTME: Provides persistent key-value storage using SQLite database with native JSON type
package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db   *sql.DB
	path string
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	storage := &SQLiteStorage{
		db:   db,
		path: dbPath,
	}

	if err := storage.createSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return storage, nil
}

func (s *SQLiteStorage) createSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS kv_store (
			key TEXT PRIMARY KEY,
			value JSON NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TRIGGER IF NOT EXISTS update_timestamp
		AFTER UPDATE ON kv_store
		BEGIN
			UPDATE kv_store SET updated_at = CURRENT_TIMESTAMP WHERE key = NEW.key;
		END;
	`

	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStorage) Set(ctx context.Context, key string, value []byte) error {
	query := `
		INSERT INTO kv_store (key, value) VALUES (?, json(?))
		ON CONFLICT(key) DO UPDATE SET
			value = json(excluded.value),
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := s.db.ExecContext(ctx, query, key, string(value))
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

func (s *SQLiteStorage) Get(ctx context.Context, key string) ([]byte, error) {
	query := `SELECT value FROM kv_store WHERE key = ?`

	var jsonValue string
	err := s.db.QueryRowContext(ctx, query, key).Scan(&jsonValue)
	if err == sql.ErrNoRows {
		return nil, nil // Key not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return []byte(jsonValue), nil
}

func (s *SQLiteStorage) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM kv_store WHERE key = ?`

	_, err := s.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

func (s *SQLiteStorage) List(ctx context.Context) ([]string, error) {
	query := `SELECT key FROM kv_store ORDER BY key`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return keys, nil
}

func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetDBPath returns the database file path for debugging
func (s *SQLiteStorage) GetDBPath() string {
	return s.path
}