package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", "series.db")
	if err != nil {
		panic(err)
	}
	return db
}