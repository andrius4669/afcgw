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

const maxFileSize = 10<<20 // 10 megs

var allowedTypes = map[string]bool{
	"image/gif":  true,
	"image/jpeg": true,
	"image/png":  true,
	"image/bmp":  true,
}


// timestamps returned by this are guaranteed to be unique
var lastTimeMutex sync.Mutex
var lastTime int64 = 0
func uniqueUnixTime() int64 {
	lastTimeMutex.Lock()
	defer lastTimeMutex.Unlock()

	unixnow := time.Now().Unix()
	if unixnow > lastTime {
		lastTime = unixnow
		return unixnow
	} else {
		lastTime ++
		return lastTime
	}
}

func postNewBoard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "supposed to create new board...")
}

// postinfo for writing
type wPostInfo struct {
	Name    string
	Subject string
	Email   string
	Message string
	File    string
}

type postResult struct {
	Board  string
	Thread uint64
	Post   uint64
}

func postNewThread(w http.ResponseWriter, r *http.Request, board string) {
	fmt.Fprintf(w, "supposed to create new thread in /%s/", board)
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

	err = r.ParseMultipartForm(1 << 20)
	if err != nil {
		http.Error(w, "400 bad request", 400)
		return
	}

	pname, ok := r.Form["name"]
	if !ok {
		http.Error(w, "400 bad request", 400)
		return
	}
	p.Name = pname[0]

	psubject, ok := r.Form["subject"]
	if !ok {
		http.Error(w, "400 bad request", 400)
		return
	}
	p.Subject = psubject[0]

	pemail, ok := r.Form["email"]
	if !ok {
		http.Error(w, "400 bad request", 400)
		return
	}
	p.Email = pemail[0]

	pmessage, ok := r.Form["message"]
	if !ok {
		http.Error(w, "400 bad request", 400)
		return
	}
	p.Message = pmessage[0]

	f, h, err := r.FormFile("file")
	if err == nil {
		defer f.Close()
		size, err := f.Seek(0, os.SEEK_END)
		if err != nil {
			http.Error(w, fmt.Sprintf("500 internal server error: %s", err), 500)
			return
		}

		if size > maxFileSize {
			http.Error(w, "file too big", 403) // 403 Forbidden
			return
		}

		f.Seek(0, os.SEEK_SET)

		mt := mime.TypeByExtension(filepath.Ext(h.Filename))
		if mt != "" {
			mt, _, _ = mime.ParseMediaType(mt)
		}
		if mt == "" || !allowedTypes[mt] {
			http.Error(w, "file type not allowed", 403) // 403 Forbidden
			return
		}
		ext, _ := mime.ExtensionsByType(mt) // shouldn't fail
		fname := strconv.FormatInt(uniqueUnixTime(), 10) + ext[0]
		fullname := "files/" + board + "/src/" + fname
		tmpname := "files/" + board + "/src/tmp_" + fname
		nf, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0666)
        if err != nil {
			http.Error(w, fmt.Sprintf("500 internal server error: %s", err), 500)
			return
        }
		io.Copy(nf, f)
		nf.Close()
		os.Rename(tmpname, fullname) // atomic :^)

		p.File = fname
	}

	var lastInsertId uint64
	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s.posts (thread, name, subject, email, date, message, file) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;", board),
                      thread, p.Name, p.Subject, p.Email, time.Now().Unix(), p.Message, p.File).Scan(&lastInsertId)
	panicErr(err)

	var pr = postResult{Board: board, Thread: thread, Post: lastInsertId}
	execTemplate(w, "posted", pr)
}
