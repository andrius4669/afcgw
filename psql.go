package main

import (
	"fmt"
	"database/sql"
	_ "github.com/lib/pq"
)

func openSQL() (*sql.DB) {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", "postgres", "postgres", "chin")
	db, err := sql.Open("postgres", dbinfo)
	panicErr(err)
	return db
}

func inputBoards(db *sql.DB, f *frontData) {
	rows, err := db.Query("SELECT name, description, info FROM boards")
	panicErr(err)

	for rows.Next() {
		var b boardInfo
		rows.Scan(&b.Name, &b.Desc, &b.Info)
		f.Boards = append(f.Boards, b)
	}
}
