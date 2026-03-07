package queryrowvsquery

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// NewDB creates a test database with sample data.
func NewDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE bench (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		db.Close()
		return nil, err
	}

	// Insert sample data.
	for i := 1; i <= 1000; i++ {
		db.Exec("INSERT INTO bench (id, name) VALUES (?, ?)", i, "test_name")
	}

	return db, nil
}

// FetchWithQueryRow uses db.QueryRow to fetch a single row.
func FetchWithQueryRow(db *sql.DB, id int) (string, error) {
	var name string
	err := db.QueryRow("SELECT name FROM bench WHERE id = ?", id).Scan(&name)
	return name, err
}

// FetchWithQuery uses db.Query to fetch a single row (requires manual Close).
func FetchWithQuery(db *sql.DB, id int) (string, error) {
	rows, err := db.Query("SELECT name FROM bench WHERE id = ?", id)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var name string
	if rows.Next() {
		err = rows.Scan(&name)
	}
	if err2 := rows.Err(); err2 != nil {
		return "", err2
	}
	return name, err
}
