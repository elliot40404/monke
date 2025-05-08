package main

import (
	"database/sql"
	"log"
	"os"
	"os/user"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db     *sql.DB
	dbPath string
)

type Expense struct {
	ID       int
	Title    string
	Amount   float64
	Day      int
	Category string
}

type CategoryTotal struct {
	Name   string
	Amount float64
}

func initDB() {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user: %v", err)
	}
	configDir := filepath.Join(currentUser.HomeDir, ".config", "monke")
	dbPath = filepath.Join(configDir, "monke.db")
	err = os.MkdirAll(configDir, 0o755)
	if err != nil {
		log.Fatalf("Error creating config directory: %v", err)
	}
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS expenses (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"title" TEXT,
		"amount" REAL,
		"day" INTEGER,
		"category" TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}
