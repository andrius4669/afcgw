package main

import (
	"database/sql"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	maxImageSize = 8 << 20
	maxMusicSize = 50 << 20 // :^)
)

// nice place to also include file sizes
var allowedTypes = map[string]int64{
	"image/gif":  maxImageSize,
	"image/jpeg": maxImageSize,
	"image/png":  maxImageSize,
	"image/bmp":  maxImageSize,
	"audio/mpeg": maxMusicSize,
	"audio/ogg":  maxMusicSize,
	"audio/flac": maxMusicSize,
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
	unixnow := (t.Unix() * 1000) + ((t.UnixNano() / 1000000) % 1000)
	if unixnow > lastTime {
		lastTime = unixnow
		return unixnow
	} else {
		lastTime++
		return lastTime
	}
}

func utcUnixTime() int64 {
	return time.Now().UTC().Unix()
}

type newBoardInfo struct {
	Name, Desc, Info string
}

func initDatabase(db *sql.DB) {
	err := os.MkdirAll(pathBaseDir(), os.ModePerm)
	panicErr(err)
	err = os.MkdirAll(pathStaticDir(""), os.ModePerm)
	panicErr(err)

	create_q := `CREATE TABLE IF NOT EXISTS boards (
		name        text    PRIMARY KEY,
		description text    NOT NULL,
		info        text    NOT NULL,
		maxthreads  integer,
		bumplimit   integer
	)`
	stmt, err := db.Prepare(create_q)
	panicErr(err)
	_, err = stmt.Exec()
	panicErr(err)

	create_q = `CREATE TABLE IF NOT EXISTS ip_bans (
		ip_addr inet PRIMARY KEY,
		reason  text NOT NULL
	)`
	stmt, err = db.Prepare(create_q)
	panicErr(err)
	_, err = stmt.Exec()
	panicErr(err)

	create_q = `CREATE TABLE IF NOT EXISTS admins (
		username text PRIMARY KEY,
		password text NOT NULL
	)`
	stmt, err = db.Prepare(create_q)
	panicErr(err)
	_, err = stmt.Exec()
	panicErr(err)

	// only these tables so far...
}

func initDbCmd() {
	fmt.Print("initialising database...")

	db := openSQL()
	defer db.Close()

	initDatabase(db)

	fmt.Print(" done.\n")
}

func validBoardName(name string) bool {
	ok, _ := regexp.MatchString("^[a-z0-9]{1,10}$", name)
	if !ok {
		return false
	}
	switch name {
	case "static":
	case "mod":
		return false
	}
	return true
}

func makeNewBoard(db *sql.DB, dbi *newBoardInfo) {
	// prepare schema
	stmt, err := db.Prepare("CREATE SCHEMA IF NOT EXISTS $1")
	panicErr(err)
	_, err = stmt.Exec(dbi.Name) // result isn't very meaningful for us, we check err regardless
	panicErr(err)

	// prepare tables
	create_q := `CREATE TABLE IF NOT EXISTS %s.posts (
		id       bigserial PRIMARY KEY,
		thread   bigint,
		name     text      NOT NULL,
		trip     text      NOT NULL,
		subject  text      NOT NULL,
		email    text      NOT NULL,
		date     bigint    NOT NULL,
		message  text      NOT NULL,
		file     text      NOT NULL,
		original text      NOT NULL,
		thumb    text      NOT NULL,
		ip_addr  inet
	)`
	stmt, err = db.Prepare(fmt.Sprintf(create_q, dbi.Name))
	panicErr(err)
	_, err = stmt.Exec()
	panicErr(err)

	create_q = `CREATE INDEX ON %s.posts (thread)`
	stmt, err = db.Prepare(fmt.Sprintf(create_q, dbi.Name))
	panicErr(err)
	_, err = stmt.Exec()
	panicErr(err)

	create_q = `CREATE TABLE IF NOT EXISTS %s.threads (
		id      bigint  PRIMARY KEY,
		bump    bigint  NOT NULL,
		bumpnum integer NOT NULL
	)`
	stmt, err = db.Prepare(fmt.Sprintf(create_q, dbi.Name))
	panicErr(err)
	_, err = stmt.Exec()
	panicErr(err)

	// create dir tree
	err = os.MkdirAll(pathBoardDir(dbi.Name), os.ModePerm)
	panicErr(err)
	err = os.MkdirAll(pathSrcDir(dbi.Name), os.ModePerm)
	panicErr(err)
	err = os.MkdirAll(pathThumbDir(dbi.Name), os.ModePerm)
	panicErr(err)
	err = os.MkdirAll(pathStaticDir(dbi.Name), os.ModePerm)
	panicErr(err)

	// insert to board list
	create_q = `INSERT INTO boards (name, description, info) VALUES ($1, $2, $3)`
	stmt, err = db.Prepare(create_q)
	panicErr(err)
	_, err = stmt.Exec(dbi.Name, dbi.Desc, dbi.Info)
	panicErr(err)

	// we're done
}

func deleteBoard(db *sql.DB, name string) bool {
	var bname string
	err := db.QueryRow("DELETE FROM boards WHERE name=$1 RETURNING name", name).Scan(&bname)
	if err == sql.ErrNoRows {
		// already deleted or invalid name, we have nothing to do there
		return false
	}
	panicErr(err)

	stmt, err := db.Prepare("DROP SCHEMA IF EXISTS $1")
	panicErr(err)
	_, err = stmt.Exec(bname)
	panicErr(err)

	os.RemoveAll(pathBoardDir(name))

	return true
}

func postNewBoard(w http.ResponseWriter, r *http.Request) {
	var nbi newBoardInfo

	r.ParseForm()

	bname, ok := r.Form["name"]
	if !ok {
		http.Error(w, "400 bad request: no name field", 400)
		return
	}
	nbi.Name = bname[0]
	if !validBoardName(nbi.Name) {
		http.Error(w, "400 bad request: invalid board name", 400)
		return
	}

	bdesc, ok := r.Form["desc"]
	if !ok {
		http.Error(w, "400 bad request: no desc field", 400)
		return
	}
	nbi.Desc = bdesc[0]

	binfo, ok := r.Form["info"]
	if !ok {
		http.Error(w, "400 bad request: no info field", 400)
		return
	}
	nbi.Info = binfo[0]

	db := openSQL()
	defer db.Close()

	makeNewBoard(db, &nbi)
	execTemplate(w, "boardcreated", &nbi)
}

func postDelBoard(w http.ResponseWriter, r *http.Request) {
	var board string

	bname, ok := r.Form["name"]
	if !ok {
		http.Error(w, "400 bad request: no name field", 400)
		return
	}
	board = bname[0]

	db := openSQL()
	defer db.Close()

	ok = deleteBoard(db, board)
	if !ok {
		http.Error(w, "500 internal server error: board deletion failed", 500)
		return
	}

	execTemplate(w, "boarddeleted", &board)
}

// postinfo for writing
type wPostInfo struct {
	Name     string
	Trip     string
	Subject  string
	Email    string
	Message  string
	File     string
	Original string // original filename
	Thumb    string
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

func acceptPost(w http.ResponseWriter, r *http.Request, p *wPostInfo, board string, isop bool) bool {
	var err error

	err = r.ParseMultipartForm(1 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("400 bad request: ParseMultipartForm failed: %s", err), 400)
		return false
	}

	pname, ok := r.Form["name"]
	if !ok {
		http.Error(w, "400 bad request: has no name field", 400)
		return false
	}
	p.Name, p.Trip = MakeTrip(pname[0])

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
		fullname := pathSrcFile(board, fname)
		tmpname := pathSrcFile(board, ".tmp."+fname)
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

		tname, err := makeThumb(fullname, fname, board, ext, mt, isop)
		if err != nil {
			fmt.Printf("error generating thumb for %s: %s\n", fname, err)
		}
		p.Thumb = tname
	}

	return true
}

func postNewThread(w http.ResponseWriter, r *http.Request, board string) {
	var p wPostInfo

	db := openSQL()
	defer db.Close()

	var bname string
	var maxthreads sql.NullInt64
	err := db.QueryRow("SELECT name, maxthreads FROM boards WHERE name=$1", board).Scan(&bname, &maxthreads)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	if !acceptPost(w, r, &p, board, true) {
		return
	}

	nowtime := utcUnixTime()

	var lastInsertId uint64
	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s.posts (name, trip, subject, email, date, message, file, original, thumb) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id;", board),
		p.Name, p.Trip, p.Subject, p.Email, nowtime, p.Message, p.File, p.Original, p.Thumb).Scan(&lastInsertId)
	panicErr(err)

	stmt, err := db.Prepare(fmt.Sprintf("INSERT INTO %s.threads (id, bump, bumpnum) VALUES ($1, $2, $3)", board))
	panicErr(err)
	_, err = stmt.Exec(lastInsertId, nowtime, 0)
	panicErr(err)

	// prune excess threads if limit exists
	if maxthreads.Valid && maxthreads.Int64 != 0 {
		//var numrows uint64
		//err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s.threads", board)).Scan(&numrows)
		//panicErr(err)
		delq := `
			DELETE FROM %s.threads
			WHERE id = any (array(
				SELECT id FROM %s.threads
				ORDER BY bump DESC
				OFFSET $1))
			RETURNING id`
		rows, err := db.Query(fmt.Sprintf(delq, board, board), uint64(maxthreads.Int64))
		panicErr(err)
		for rows.Next() {
			var tid uint64
			err = rows.Scan(&tid)
			panicErr(err)
			prunePosts(db, board, tid)
		}
	}

	var pr = postResult{Board: board, Thread: lastInsertId, Post: lastInsertId}
	execTemplate(w, "threadcreated", pr)
}

func bumpThread(db *sql.DB, board string, thread uint64, t int64) {
	q := `UPDATE %s.threads
SET bump = $1, bumpnum = bumpnum + 1
WHERE id = $2`
	stmt, err := db.Prepare(fmt.Sprintf(q, board))
	panicErr(err)
	_, err = stmt.Exec(t, thread)
	panicErr(err)
}

func postNewPost(w http.ResponseWriter, r *http.Request, board string, thread uint64) {
	var p wPostInfo

	db := openSQL()
	defer db.Close()

	var bumplimit sql.NullInt64
	err := db.QueryRow("SELECT bumplimit FROM boards WHERE name=$1", board).Scan(&bumplimit)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	var bumpnum uint32
	err = db.QueryRow(fmt.Sprintf("SELECT bumpnum FROM %s.threads WHERE id=$1", board), thread).Scan(&bumpnum)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	panicErr(err)

	if !acceptPost(w, r, &p, board, false) {
		return
	}

	nowtime := utcUnixTime()

	var lastInsertId uint64
	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s.posts (thread, name, trip, subject, email, date, message, file, original, thumb) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id;", board),
		thread, p.Name, p.Trip, p.Subject, p.Email, nowtime, p.Message, p.File, p.Original, p.Thumb).Scan(&lastInsertId)
	panicErr(err)

	// TODO: check for sage
	if !bumplimit.Valid || bumpnum < uint32(bumplimit.Int64) {
		bumpThread(db, board, thread, nowtime)
	}

	var pr = postResult{Board: board, Thread: thread, Post: lastInsertId}
	execTemplate(w, "posted", pr)
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
	var fname sql.NullString
	var tname sql.NullString
	err = db.QueryRow(fmt.Sprintf("DELETE FROM %s.posts WHERE id=$1 RETURNING thread, file, thumb", board), post).Scan(&thread, &fname, &tname)
	if err == sql.ErrNoRows {
		return true // already deleted
	}
	panicErr(err)

	pruneFiles(board, fname.String, tname.String)

	// if it was OP, prune whole thread
	if !thread.Valid || thread.Int64 == 0 || uint64(thread.Int64) == post {
		pr.Thread = post
		pruneThreadReplies(db, board, post)
	} else {
		pr.Thread = uint64(thread.Int64)
	}

	return true
}

func postDelete(w http.ResponseWriter, r *http.Request, board string) {
	r.ParseForm()
	post, ok := r.PostForm["id"]
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
