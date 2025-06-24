package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS reminders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER,
		text TEXT,
		datetime DATETIME,
		repeat_type TEXT DEFAULT 'none',
		repeat_enabled BOOLEAN DEFAULT 0
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatal(err)
	}

	// Migrate existing table if needed
	migrationSQL := `
		ALTER TABLE reminders ADD COLUMN repeat_type TEXT DEFAULT 'none';
		ALTER TABLE reminders ADD COLUMN repeat_enabled BOOLEAN DEFAULT 0;
	`
	// Ignore errors for migration as columns might already exist
	db.Exec(migrationSQL)

	return db
}
