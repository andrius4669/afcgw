package main

import (
	"fmt"
	"net/http"
	"sync"
	"database/sql"
	"mime"
	"path/filepath"
	"os"
	"time"
	"io"
	"strconv"
)

const (
	maxImageSize =  8<<20
	maxMusicSize = 50<<20 // :^)
)

// nice place to also include file sizes
var allowedTypes = map[string]int64{
	"image/gif":      maxImageSize,
	"image/jpeg":     maxImageSize,
	"image/png":      maxImageSize,
	"image/bmp":      maxImageSize,
	"audio/mpeg":     maxMusicSize,
	"audio/ogg":      maxMusicSize,
	"audio/flac":     maxMusicSize,
}

// add our own mime stuff since golang's parser erroreusly overwrites image/bmp with image/x-ms-bmp
func initMime() {
	mime.AddExtensionType(".bmp", "image/bmp")
	mime.AddExtensionType(".ogg", "audio/ogg")
	mime.AddExtensionType(".flac", "audio/flac")
}

// timestamps returned by this are guaranteed to be unique
var lastTimeMutex sync.Mutex
var lastTime int64 = 0
func uniqueTimestamp() int64 {
	lastTimeMutex.Lock()
	defer lastTimeMutex.Unlock()

	t := time.Now().UTC()
	unixnow := (t.Unix() * 1000) + (t.UnixNano() / 1000000)
	if unixnow > lastTime {
		lastTime = unixnow
		return unixnow
	} else {
		lastTime ++
		return lastTime
	}
}

func utcUnixTime() int64 {
	return time.Now().UTC().Unix()
}

func postNewBoard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "supposed to create new board...")
}

// postinfo for writing
type wPostInfo struct {
	Name     string
	Subject  string
	Email    string
	Message  string
	File     string
	Original string // original filename
}

type postResult struct {
	Board  string
	Thread uint64
	Post   uint64
}

func (r *postResult) HasThread() bool {
	return r.Thread != 0
}

func (r *postResult) IsThread() bool {
	return r.Thread == r.Post
}

// deletes file and stuff cached from it
func delFile(board, fname string) {
	fullname := "files/" + board + "/src/" + fname
	os.Remove(fullname)
}

func acceptPost(w http.ResponseWriter, r *http.Request, p *wPostInfo, board string) bool {
	err := r.ParseMultipartForm(1 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("400 bad request: ParseMultipartForm failed: %s", err), 400)
		return false
	}

	pname, ok := r.Form["name"]
	if !ok {
		http.Error(w, "400 bad request: has no name field", 400)
		return false
	}
	p.Name = pname[0]

	psubject, ok := r.Form["subject"]
	if !ok {
		http.Error(w, "400 bad request: has no subject field", 400)
		return false
	}
	p.Subject = psubject[0]

	pemail, ok := r.Form["email"]
	if !ok {
		http.Error(w, "400 bad request: has no email field", 400)
		return false
	}
	p.Email = pemail[0]

	pmessage, ok := r.Form["message"]
	if !ok {
		http.Error(w, "400 bad request: has no message field", 400)
		return false
	}
	p.Message = pmessage[0]

	f, h, err := r.FormFile("file")
	if err == nil {
		defer f.Close()
		size, err := f.Seek(0, os.SEEK_END)
		if err != nil {
			http.Error(w, fmt.Sprintf("500 internal server error: %s", err), 500)
			return false
		}
		_, err = f.Seek(0, os.SEEK_SET)
		if err != nil {
			http.Error(w, fmt.Sprintf("500 internal server error: %s", err), 500)
			return false
		}

		ext := filepath.Ext(h.Filename)
		mt := mime.TypeByExtension(ext)
		if mt != "" {
			mt, _, _ = mime.ParseMediaType(mt)
		}
		if mt == "" {
			http.Error(w, "file type not allowed", 403) // 403 Forbidden
			return false
		}
		maxSize, ok := allowedTypes[mt]
		if !ok {
			http.Error(w, "file type not allowed", 403) // 403 Forbidden
			return false
		}
		if size > maxSize {
			http.Error(w, "file too big", 403) // 403 Forbidden
			return false
		}
		fname := strconv.FormatInt(uniqueTimestamp(), 10) + ext
		fullname := "files/" + board + "/src/" + fname
		tmpname := "files/" + board + "/src/tmp_" + fname
		nf, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			http.Error(w, fmt.Sprintf("500 internal server error: %s", err), 500)
			return false
		}
		io.Copy(nf, f)
		nf.Close()
		os.Rename(tmpname, fullname) // atomic :^)

		p.File = fname
		p.Original = h.Filename
	}

	return true
}

func postNewThread(w http.ResponseWriter, r *http.Request, board string) {
	var p wPostInfo

	db := openSQL()
	defer db.Close()

	var bname string
	err := db.QueryRow("SELECT name FROM boards WHERE name=$1", board).Scan(&bname)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	if !acceptPost(w, r, &p, board) {
		return
	}

	nowtime := utcUnixTime()

	var lastInsertId uint64
	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s.posts (name, subject, email, date, message, file, original) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;", board),
                      p.Name, p.Subject, p.Email, nowtime, p.Message, p.File, p.Original).Scan(&lastInsertId)
	panicErr(err)

	stmt, err := db.Prepare(fmt.Sprintf("INSERT INTO %s.threads (id, bump) VALUES ($1, $2)", board))
	panicErr(err)

	_, err = stmt.Exec(lastInsertId, nowtime) // result isn't very meaningful for us, we check err regardless
	panicErr(err)

	var pr = postResult{Board: board, Thread: lastInsertId, Post: lastInsertId}
	execTemplate(w, "newthread", pr)
}

func postNewPost(w http.ResponseWriter, r *http.Request, board string, thread uint64) {
	var p wPostInfo

	db := openSQL()
	defer db.Close()

	var bname string
	err := db.QueryRow("SELECT name FROM boards WHERE name=$1", board).Scan(&bname)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	var bid uint64
	err = db.QueryRow(fmt.Sprintf("SELECT id FROM %s.threads WHERE id=$1", board), thread).Scan(&bid)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	if !acceptPost(w, r, &p, board) {
		return
	}

	var lastInsertId uint64
	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s.posts (thread, name, subject, email, date, message, file, original) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;", board),
                      thread, p.Name, p.Subject, p.Email, utcUnixTime(), p.Message, p.File, p.Original).Scan(&lastInsertId)
	panicErr(err)

	var pr = postResult{Board: board, Thread: thread, Post: lastInsertId}
	execTemplate(w, "posted", pr)
}

func pruneThread(db *sql.DB, board string, thread uint64) {
	stmt, err := db.Prepare(fmt.Sprintf("DELETE FROM %s.threads WHERE id=$1", board))
	panicErr(err)

	_, err = stmt.Exec(thread) // result isn't very meaningful for us, but we check err regardless
	panicErr(err)

	rows, err := db.Query(fmt.Sprintf("DELETE FROM %s.posts WHERE thread=$1 RETURNING file", board), thread)
	panicErr(err)
	for rows.Next() {
		var fname sql.NullString
		err = rows.Scan(&fname)
		panicErr(err)
		if fname.Valid && fname.String != "" {
			delFile(board, fname.String)
		}
	}
}

func removePost(w http.ResponseWriter, r *http.Request, pr *postResult, board string, post uint64) bool {
	db := openSQL()
	defer db.Close()

	var bname string
	err := db.QueryRow("SELECT name FROM boards WHERE name=$1", board).Scan(&bname)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return false
	}
	panicErr(err)

	pr.Board = board
	pr.Post = post

	var thread sql.NullInt64
	var fname  sql.NullString
	err = db.QueryRow(fmt.Sprintf("DELETE FROM %s.posts WHERE id=$1 RETURNING thread, file", board), post).Scan(&thread, &fname)
	if err == sql.ErrNoRows {
		return true // already deleted
	}
	panicErr(err)

	if fname.Valid && fname.String != "" {
		delFile(board, fname.String)
	}

	// if it was OP, prune whole thread
	if !thread.Valid || uint64(thread.Int64) == post {
		pr.Thread = post
		pruneThread(db, board, post)
	} else {
		pr.Thread = uint64(thread.Int64)
	}

	return true
}

func postDelete(w http.ResponseWriter, r *http.Request, board string) {
	r.ParseForm()
	post, ok := r.Form["id"]
	if !ok {
		http.Error(w, "400 bad request: no post id specified", 400)
		return
	}
	n, err := strconv.ParseUint(post[0], 10, 64)
	if err != nil {
		http.Error(w, "400 bad request: bad post id", 400)
		return
	}
	var pr postResult
	if !removePost(w, r, &pr, board, n) {
		return
	}

	execTemplate(w, "deleted", &pr)
}
