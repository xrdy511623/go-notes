package preparedvsunprepared

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// DB wraps operations for benchmarking prepared vs unprepared statements.
type DB struct {
	db   *sql.DB
	stmt *sql.Stmt
}

// NewDB creates a test database with a simple table.
func NewDB() (*DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE bench (id INTEGER PRIMARY KEY AUTOINCREMENT, value TEXT)")
	if err != nil {
		db.Close()
		return nil, err
	}

	stmt, err := db.Prepare("INSERT INTO bench (value) VALUES (?)")
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db, stmt: stmt}, nil
}

// Close releases resources.
func (d *DB) Close() {
	d.stmt.Close()
	d.db.Close()
}

// InsertPrepared uses the pre-prepared statement.
func (d *DB) InsertPrepared(value string) error {
	_, err := d.stmt.Exec(value)
	return err
}

// InsertUnprepared uses db.Exec directly (no prepare).
func (d *DB) InsertUnprepared(value string) error {
	_, err := d.db.Exec("INSERT INTO bench (value) VALUES (?)", value)
	return err
}

// QueryPreparedRow uses a pre-prepared statement for SELECT.
func (d *DB) QueryPreparedRow(id int) (string, error) {
	stmt, err := d.db.Prepare("SELECT value FROM bench WHERE id = ?")
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	var value string
	err = stmt.QueryRow(id).Scan(&value)
	return value, err
}

// QueryUnpreparedRow uses db.QueryRow directly.
func (d *DB) QueryUnpreparedRow(id int) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM bench WHERE id = ?", id).Scan(&value)
	return value, err
}
