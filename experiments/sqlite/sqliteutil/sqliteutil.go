package sqliteutil

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const DBPath = "sqlite/data/demo.db"

type User struct {
	ID    int
	Name  string
	Email string
}

func Open() (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(DBPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

func EnsureSchema(db *sql.DB) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

func SeedUsers(db *sql.DB) error {
	rows := []User{
		{Name: "Ava Li", Email: "ava@example.com"},
		{Name: "Noah Diaz", Email: "noah@example.com"},
		{Name: "Lena Ortiz", Email: "lena@example.com"},
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM users`); err != nil {
		return fmt.Errorf("clear users: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO users(name, email) VALUES(?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(r.Name, r.Email); err != nil {
			return fmt.Errorf("insert user %s: %w", r.Email, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func QueryUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT id, name, email FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func Drop() error {
	err := os.Remove(DBPath)
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("remove db: %w", err)
}
