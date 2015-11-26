package main

import (
	"fmt"
	"net/http"
	"bytes"
	"text/template"
	"net/url"
	"time"
)

// basic info about board
type boardInfo struct {
	Name string
	Desc string
	Info string
	mod  bool
}

func (b *boardInfo) setMod(mod bool) {
	b.mod = mod
}

func (b *boardInfo) IsMod() bool {
	return b.mod
}

type threadInfo struct {
	parent *boardInfo
	Id     uint64
}

func (t *threadInfo) Board() string {
	return t.parent.Name
}

func (t *threadInfo) setMod(mod bool) {
	t.parent.setMod(mod)
}

func (t *threadInfo) IsMod() bool {
	return t.parent.IsMod()
}

// single post info
type postInfo struct {
	parent   *threadInfo
	Id       uint64
	Name     string
	Subject  string
	Email    string
	Date     int64
	Message  string
	File     string
	Original string
}

func (p *postInfo) Board() string {
	return p.parent.Board()
}

func (p *postInfo) Thread() uint64 {
	return p.parent.Id
}

func (p *postInfo) HasFile() bool {
	return p.File != ""
}

func (p *postInfo) FullFile() string {
	if p.HasFile() {
		return "/" + p.Board() + "/src/" + p.File
	}
	return ""
}

func (p *postInfo) HasOriginal() bool {
	return p.Original != ""
}

func (p *postInfo) StrOriginal() string {
	return template.HTMLEscapeString(p.Original)
}

func (p *postInfo) FOriginal() string {
	return url.QueryEscape(p.Original)
}

func (p *postInfo) FullOriginal() string {
	if p.HasOriginal() {
		return p.FullFile() + "/" + p.FOriginal()
	}
	return p.FullFile()
}

func (p *postInfo) HasName() bool {
	return p.Name != ""
}

func (p *postInfo) FName() string {
	if p.HasName() {
		return template.HTMLEscapeString(p.Name)
	}
	return "Anonymous"
}

func (p *postInfo) HasSubject() bool {
	return p.Subject != ""
}

func (p *postInfo) FSubject() string {
	if p.HasSubject() {
		return template.HTMLEscapeString(p.Subject)
	}
	return "None"
}

func (p *postInfo) HasEmail() bool {
	return p.Email != ""
}

func (p *postInfo) FEmail() string {
	return url.QueryEscape(p.Email)
}

func (p *postInfo) setMod(mod bool) {
	p.parent.setMod(mod)
}

func (p *postInfo) IsMod() bool {
	return p.parent.IsMod()
}

// prints date in format browser understands
func (p *postInfo) FDate() string {
	t := time.Unix(p.Date, 0)
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()) // Z denotes UTC
}

// format user understands better
func (p *postInfo) StrDate() string {
	t := time.Unix(p.Date, 0)
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func (p *postInfo) HasMessage() bool {
	return p.Message != ""
}

// escapes and formats message
var (
	htmlQuot = []byte("&#34;") // shorter than "&quot;"
	htmlApos = []byte("&#39;") // shorter than "&apos;" and apos was not in HTML until HTML5
	htmlAmp  = []byte("&amp;")
	htmlLt   = []byte("&lt;")
	htmlGt   = []byte("&gt;")
	htmlBr   = []byte("<br />")
)

func (p *postInfo) FMessage() string {
	b := []byte(p.Message)
	var w bytes.Buffer
	src, last := 0, 0
	for src < len(b) {
		c := b[src]
		var inc int
		var esc []byte
		switch c {
		case '"':
			esc = htmlQuot
			inc = 1
		case '\'':
			esc = htmlApos
			inc = 1
		case '&':
			esc = htmlAmp
			inc = 1
		case '<':
			esc = htmlLt
			inc = 1
		case '>':
			esc = htmlGt
			inc = 1
		case '\n':
			esc = htmlBr
			inc = 1
		default:
			src++
			continue
		}
		w.Write(b[last:src])
		w.Write(esc)
		src += inc
		last = src
	}
	w.Write(b[last:])
	return w.String()
}

type fullPostInfo struct {
	postInfo
	// TODO some additional stuff
}

type fullThreadInfo struct {
	threadInfo
	Op      fullPostInfo
	Replies []fullPostInfo
}

type fullBoardInfo struct {
	boardInfo
	Threads []fullThreadInfo
}

// front page info
type fullFrontData struct {
	// TODO: add sth moar
	Boards []boardInfo
}


func renderFront(w http.ResponseWriter, r *http.Request) {
	db := openSQL()
	defer db.Close()

	var f fullFrontData
	inputBoards(db, &f)

	execTemplate(w, "boards", &f)
}

func renderBoard(w http.ResponseWriter, r *http.Request, board string, mod bool) {
	db := openSQL()
	defer db.Close()

	var b fullBoardInfo
	if !inputThreads(db, &b, board) {
		http.NotFound(w, r)
		return
	}
	b.setMod(mod)
	execTemplate(w, "threads", &b)
}

func renderThread(w http.ResponseWriter, r *http.Request, board string, thread uint64, mod bool) {
	db := openSQL()
	defer db.Close()

	var t fullThreadInfo
	if !inputPosts(db, &t, board, thread) {
		http.NotFound(w, r)
		return
	}
	t.setMod(mod)

	execTemplate(w, "posts", &t)
}
