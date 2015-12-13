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


func inputBoards(db *sql.DB, f *fullFrontData) {
	rows, err := db.Query("SELECT name, description, info FROM boards")
	panicErr(err)

	for rows.Next() {
		var b boardInfo
		rows.Scan(&b.Name, &b.Desc, &b.Info)
		f.Boards = append(f.Boards, b)
	}
}

func inputThreads(db *sql.DB, b *fullBoardInfo, board string) bool {
	err := db.QueryRow("SELECT name, description, info FROM boards WHERE name=$1", board).Scan(&b.Name, &b.Desc, &b.Info)
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	// TODO: ordering & limiting
	rows, err := db.Query(fmt.Sprintf("SELECT id FROM %s.threads ORDER BY bump DESC", board))
	panicErr(err)
	for rows.Next() {
		var t fullThreadInfo
		t.parent = &b.boardInfo
		t.postMap = make(map[uint64]int)
		rows.Scan(&t.Id)
		b.Threads = append(b.Threads, t)
	}

	for i := range b.Threads {
		{
			var op fullPostInfo
			op.parent = &b.Threads[i].threadInfo
			op.fparent = &b.Threads[i]
			// expliclty fetch OP
			err = db.QueryRow(fmt.Sprintf("SELECT id, name, trip, subject, email, date, message, file, original, thumb FROM %s.posts WHERE id=$1", board), b.Threads[i].Id).
		                     Scan(&op.Id, &op.Name, &op.Trip, &op.Subject, &op.Email, &op.Date, &op.Message, &op.File, &op.Original, &op.Thumb)
			if err == sql.ErrNoRows {
				// thread without OP, it broke. TODO: remove from list
			} else {
				panicErr(err)
			}
			b.Threads[i].Op = op
			b.Threads[i].postMap[op.Id] = 0
		}

		// TODO sorting and limiting (we need to show only few posts in board view)
		rows, err = db.Query(fmt.Sprintf("SELECT id, name, trip, subject, email, date, message, file, original, thumb FROM %s.posts WHERE thread=$1", board), b.Threads[i].Id)
		panicErr(err)
		for rows.Next() {
			var p fullPostInfo
			p.parent = &b.Threads[i].threadInfo
			p.fparent = &b.Threads[i]
			err = rows.Scan(&p.Id, &p.Name, &p.Trip, &p.Subject, &p.Email, &p.Date, &p.Message, &p.File, &p.Original, &p.Thumb)
			panicErr(err)
			if p.Id == b.Threads[i].Id {
				continue // OP already included -- shouldn't normally happen
			}
			b.Threads[i].Replies = append(b.Threads[i].Replies, p)
			b.Threads[i].postMap[p.Id] = len(b.Threads[i].Replies)
		}
	}

	return true
}

func inputPosts(db *sql.DB, t *fullThreadInfo, board string, thread uint64) bool {
	t.parent = &boardInfo{}
	err := db.QueryRow("SELECT name, description, info FROM boards WHERE name=$1", board).Scan(&t.parent.Name, &t.parent.Desc, &t.parent.Info)
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	err = db.QueryRow(fmt.Sprintf("SELECT id FROM %s.threads WHERE id=$1", board), thread).Scan(&t.Id)
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	t.Op.parent = &t.threadInfo
	t.Op.fparent = t
	err = db.QueryRow(fmt.Sprintf("SELECT id, name, trip, subject, email, date, message, file, original, thumb FROM %s.posts WHERE id=$1", board), thread).
	                 Scan(&t.Op.Id, &t.Op.Name, &t.Op.Trip, &t.Op.Subject, &t.Op.Email, &t.Op.Date, &t.Op.Message, &t.Op.File, &t.Op.Original, &t.Op.Thumb);
	if err == sql.ErrNoRows {
		return false
	}
	panicErr(err)

	t.postMap[t.Op.Id] = 0

	rows, err := db.Query(fmt.Sprintf("SELECT id, name, trip, subject, email, date, message, file, original, thumb FROM %s.posts WHERE thread=$1", board), thread)
	panicErr(err)
	for rows.Next() {
		var p fullPostInfo
		p.parent = &t.threadInfo
		p.fparent = t
		err = rows.Scan(&p.Id, &p.Name, &p.Trip, &p.Subject, &p.Email, &p.Date, &p.Message, &p.File, &p.Original, &p.Thumb)
		panicErr(err)
		if p.Id == thread {
			continue // OP already included
		}
		t.Replies = append(t.Replies, p)
		t.postMap[p.Id] = len(t.Replies)
	}

	return true
}