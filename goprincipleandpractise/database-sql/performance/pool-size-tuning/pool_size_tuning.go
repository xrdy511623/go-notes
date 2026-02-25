package poolsizetuning

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// NewDB creates a test database with the given pool configuration.
func NewDB(maxOpen, maxIdle int) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	_, err = db.Exec("CREATE TABLE bench (id INTEGER PRIMARY KEY AUTOINCREMENT, value TEXT)")
	if err != nil {
		db.Close()
		return nil, err
	}

	// Pre-populate with some data for read benchmarks.
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO bench (value) VALUES (?)", "seed_data")
	}

	return db, nil
}

// DoWork simulates a typical database operation (read + write).
func DoWork(db *sql.DB, id int) error {
	var value string
	err := db.QueryRow("SELECT value FROM bench WHERE id = ?", (id%100)+1).Scan(&value)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO bench (value) VALUES (?)", value)
	return err
}
