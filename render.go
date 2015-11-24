package main

import (
	"fmt"
	"net/http"
	"database/sql"
	_ "github.com/lib/pq"
)

func openSQL() (*sql.DB) {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", "postgres", "postgres", "chin")
	db, err := sql.Open("postgres", dbinfo)
	panicErr(err)
	return db
}

// TODO
func renderFront(w http.ResponseWriter, r *http.Request) {
	db := openSQL()
	rows, err := db.Query("SELECT name, description, info FROM boards")
	panicErr(err)
	//fmt.Fprint(w, "<html><head><title>HUEHUEHUE</title></head><body><b>Boards</b><br />")

	type boardInfo struct {
		Name string
		Desc string
		Info string
	}
	var frontData struct {
		Boards []boardInfo
	}

	for rows.Next() {
		var b boardInfo
		rows.Scan(&b.Name, &b.Desc, &b.Info)
		frontData.Boards = append(frontData.Boards, b)
	}

	execTemplate(w, "boards", frontData)
}

type postInfo struct {
	Id      uint64
	Name    string
	Subject string
	Email   string
	Date    uint64
	Message string
	File    string
}

// TODO
func renderBoard(w http.ResponseWriter, r *http.Request, board string) {
	db := openSQL()

	type threadInfo struct {
		Id    uint64
		Posts []postInfo
	}

	type boardInfo struct {
		Name    string
		Desc    string
		Info    string
		Threads []threadInfo
	}

	var b boardInfo

	err := db.QueryRow("SELECT name, description, info FROM boards WHERE name=$1", board).Scan(&b.Name, &b.Desc, &b.Info)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

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
			e := db.QueryRow(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE id=$1", board), b.Threads[i].Id).Scan(&op.Id, &op.Name, &op.Subject, &op.Email, &op.Date, &op.Message, &op.File)
			if e == sql.ErrNoRows {
				// thread without OP, it broke
				continue
			}
			b.Threads[i].Posts = append(b.Threads[i].Posts, op)
		}

		// TODO sorting and limiting (we need to show only few posts in board view)
		r, e := db.Query(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE thread=$1", board), b.Threads[i].Id)
		panicErr(e)
		for r.Next() {
			var p postInfo
			r.Scan(&p.Id, &p.Name, &p.Subject, &p.Email, &p.Date, &p.Message, &p.File)
			if p.Id == b.Threads[i].Id {
				continue // OP already included
			}
			b.Threads[i].Posts = append(b.Threads[i].Posts, p)
		}
	}

	execTemplate(w, "threads", b)
/*
	fmt.Fprintf(w, "<html><head><title>jej</title></head><body><b>Threads in /%s/</b><br />", board)

	//fmt.Fprintf(w, "<form action=\"/%s/thread/new\" method=\"post\">New thread<br />Name: <input type=\"text\" name=\"name\" />"

	c := 0
	for rows.Next() {
		if c > 0 {
			fmt.Fprint(w, "<br />")
		}
		c++
		var id uint64
		rows.Scan(&id)
		fmt.Fprintf(w, "<a href=\"/%s/thread/%d\">#%d</a>", board, id, id)
	}
	fmt.Fprint(w, "</body></head>")*/
	//fmt.Fprintf(w, "<html><head><title>JEJ</title></head><body><b>/b/ board</b><br /><a href=\"thread/1\">1</a><br /><a href=\"https://soundcloud.com/fearless1406/bizarre-contact-vs-electro-sun-out-of-your-love-sesto-sento-vs-dror-rmx\">:^)</a></body></html>")
}

// TODO
func renderThread(w http.ResponseWriter, r *http.Request, board string, thread uint64) {
	db := openSQL()
	var b string
	// check existence
	err := db.QueryRow("SELECT name FROM boards WHERE name=$1", board).Scan(&b)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	// actual query
	rows, err := db.Query(fmt.Sprintf("SELECT id, name, subject, email, date, message, file FROM %s.posts WHERE thread=$1", board), thread)
	panicErr(err)

	fmt.Fprintf(w, "<html><head><title>wew lad</title></head><body><b>/%s/ #%d</b><br />", board, thread)

	for rows.Next() {
		fmt.Fprint(w, "<div style=\"backgroud-color:green;\">")
		fmt.Fprint(w, "<p class=\"meta\">")
		var name, subject, email, message, file string
		var id, date uint64
		rows.Scan(&id, &name, &subject, &email, &date, &message, &file)
		fmt.Fprintf(w, "ID: %d", id)
		if name == "" {
			name = "Anonymous" // we le anonymouse haxorz now
		}
		fmt.Fprintf(w, " Name: %s", name)
		if subject != "" {
			fmt.Fprintf(w, " Subject: %s", subject)
		}
		if email != "" {
			fmt.Fprintf(w, " Email: %s", email)
		}
		fmt.Fprintf(w, " Date (in UNIX): %d", date)
		fmt.Fprint(w, "</p>")
		fmt.Fprint(w, "<div>")
		fmt.Fprint(w, message)
		fmt.Fprint(w, "</div></div>")
	}
	fmt.Fprint(w, "<br />")



	fmt.Fprint(w, "</body></html>")

	/*
	s := `
	<html><head><title>WEW</title></head><body>
	<b>THRID NO 1</b>
	<br />
	<a href="/b/src/file1.png">do not click</a>
	<br />
	<a href="https://www.youtube.com/watch?v=YFZ8aO1noU8">LAD</a><!-- this is some good shit -->
	</body></html>
	`
	fmt.Fprint(w, s)*/
}
