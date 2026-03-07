package txbatchvsindividual

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// NewDB creates a test database.
func NewDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE bench (id INTEGER PRIMARY KEY AUTOINCREMENT, value TEXT)")
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// InsertIndividual inserts n rows with individual Exec calls.
func InsertIndividual(db *sql.DB, n int) error {
	for i := 0; i < n; i++ {
		_, err := db.Exec("INSERT INTO bench (value) VALUES (?)", fmt.Sprintf("val_%d", i))
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertBatchTx inserts n rows within a single transaction.
func InsertBatchTx(db *sql.DB, n int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i := 0; i < n; i++ {
		_, err := tx.Exec("INSERT INTO bench (value) VALUES (?)", fmt.Sprintf("val_%d", i))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertBatchTxPrepared inserts n rows within a transaction using a prepared statement.
func InsertBatchTxPrepared(db *sql.DB, n int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO bench (value) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 0; i < n; i++ {
		_, err := stmt.Exec(fmt.Sprintf("val_%d", i))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
