package main

import (
	"fmt"
	"os"
	"database/sql"
)

func pruneFiles(board, fname, tname string) {
	if fname != "" && fname[0] != '/' {
		os.Remove(pathSrcFile(board, fname))
	}
	if tname != "" && tname[0] != '/' {
		os.Remove(pathThumbFile(board, tname))
	}
}

func pruneReplies(db *sql.DB, board string, thread uint64) {
	rows, err := db.Query(fmt.Sprintf("DELETE FROM %s.posts WHERE thread=$1 RETURNING file, thumb", board), thread)
	panicErr(err)
	for rows.Next() {
		var fname, tname sql.NullString
		err = rows.Scan(&fname, &tname)
		panicErr(err)
		pruneFiles(board, fname.String, tname.String)
	}
}

func pruneOp(db *sql.DB, board string, thread uint64) {
	var fname, tname sql.NullString
	err := db.QueryRow(fmt.Sprintf("DELETE FROM %s.posts WHERE id=$1 RETURNING file, thumb", board), thread).Scan(&fname, &tname)
	if err == sql.ErrNoRows {
		return
	}
	panicErr(err)
	pruneFiles(board, fname.String, tname.String)
}

func pruneThread(db *sql.DB, board string, thread uint64) {
	stmt, err := db.Prepare(fmt.Sprintf("DELETE FROM %s.threads WHERE id=$1", board))
	panicErr(err)
	_, err = stmt.Exec(thread)
	panicErr(err)
}

func prunePosts(db *sql.DB, board string, thread uint64) {
	pruneReplies(db, board, thread)
	pruneOp(db, board, thread)
}

func pruneThreadReplies(db *sql.DB, board string, thread uint64) {
	pruneThread(db, board, thread)
	pruneReplies(db, board, thread)
}
