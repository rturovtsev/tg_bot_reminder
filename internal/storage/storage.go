package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем базовую таблицу если её нет
	createTableSQL := `CREATE TABLE IF NOT EXISTS reminders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER,
		text TEXT
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatal(err)
	}

	// Проверяем и добавляем столбец datetime если его нет
	if !columnExists(db, "reminders", "datetime") {
		_, err = db.Exec(`ALTER TABLE reminders ADD COLUMN datetime DATETIME`)
		if err != nil {
			log.Fatal("Failed to add datetime column:", err)
		}
		log.Println("Added datetime column to reminders table")
	}

	// Проверяем и добавляем столбец repeat_type если его нет
	if !columnExists(db, "reminders", "repeat_type") {
		_, err = db.Exec(`ALTER TABLE reminders ADD COLUMN repeat_type TEXT DEFAULT 'none'`)
		if err != nil {
			log.Fatal("Failed to add repeat_type column:", err)
		}
		log.Println("Added repeat_type column to reminders table")
	}

	return db
}

// columnExists проверяет существование столбца в таблице
func columnExists(db *sql.DB, tableName, columnName string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue interface{}
		var pk int

		err = rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			continue
		}

		if name == columnName {
			return true
		}
	}
	return false
}
