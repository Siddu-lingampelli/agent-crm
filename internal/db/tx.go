// Package db — transactions helpers.
package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Transaction runs fn inside a single transaction and rolls back on error.
// Used for multi-record operations like lead conversion.
func (d *DB) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
