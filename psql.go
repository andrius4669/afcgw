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

func inputThreads(db *sql.DB, b *threadsInfo, board string) bool {
	err := db.QueryRow("SELECT name, description, info FROM boards WHERE name=$1", board).Scan(&b.Name, &b.Desc, &b.Info)
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	// TODO: ordering & limiting
	rows, err := db.Query(fmt.Sprintf("SELECT id FROM %s.threads", board))
	panicErr(err)
	for rows.Next() {
		var t threadInfo
		rows.Scan(&t.Id)
		b.Threads = append(b.Threads, t)
	}

	for i := range b.Threads {
		{
			var op postInfo
			// expliclty fetch OP
			err = db.QueryRow(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE id=$1", board), b.Threads[i].Id).
		                     Scan(&op.Id, &op.Name, &op.Subject, &op.Email, &op.Date, &op.Message, &op.File)
			if err == sql.ErrNoRows {
				// thread without OP, it broke. TODO: remove from list
			}
			b.Threads[i].Op = op
		}

		// TODO sorting and limiting (we need to show only few posts in board view)
		rows, err = db.Query(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE thread=$1", board), b.Threads[i].Id)
		panicErr(err)
		for rows.Next() {
			var p postInfo
			rows.Scan(&p.Id, &p.Name, &p.Subject, &p.Email, &p.Date, &p.Message, &p.File)
			if p.Id == b.Threads[i].Id {
				continue // OP already included
			}
			b.Threads[i].Replies = append(b.Threads[i].Replies, p)
		}
	}

	return true
}

func inputPosts(db *sql.DB, t *postsInfo, board string, thread uint64) bool {
	err := db.QueryRow("SELECT name, description, info FROM boards WHERE name=$1", board).Scan(&t.Name, &t.Desc, &t.Info)
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	err = db.QueryRow(fmt.Sprintf("SELECT id FROM %s.threads WHERE id=$1", board), thread).Scan(&t.Id)
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	err = db.QueryRow(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE id=$1", board), thread).
	                 Scan(&t.Op.Id, &t.Op.Name, &t.Op.Subject, &t.Op.Email, &t.Op.Date, &t.Op.Message, &t.Op.File);
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	rows, err := db.Query(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE thread=$1", board), thread)
	panicErr(err)
	for rows.Next() {
		var p postInfo
		rows.Scan(&p.Id, &p.Name, &p.Subject, &p.Email, &p.Date, &p.Message, &p.File)
		if p.Id == thread {
			continue // OP already included
		}
		t.Replies = append(t.Replies, p)
	}

	return true
}