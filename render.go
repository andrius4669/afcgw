package main

import (
	"fmt"
	"net/http"
	"bytes"
	"text/template"
	"net/url"
	"time"
	"strconv"
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

func (p *postInfo) FullOriginal() string {
	if p.HasOriginal() {
		var u = url.URL{Path: p.FullFile() + "/" + p.Original}
		return u.EscapedPath()
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

// check existence of cross-linking, ex: >>>/b/ >>>/pol/13548
func checkCrossPattern(b []byte, src int, end *int, board *string, post *uint64) bool {
	// shortest crosslink: >>>/a/ - 6 chars
	if src + 6 >= len(b) {
		return false
	}
	if b[src+1] != '>' || b[src+2] != '>' || b[src+3] != '/' {
		return false
	}
	src += 4
	idx := src
	for ;; idx++ {
		if idx >= len(b) {
			return false
		}
		if b[idx] == '/' {
			if idx > src {
				break
			} else {
				return false
			}
		}
		if (b[idx] < 'a' || b[idx] > 'z') && (b[idx] < 'A' || b[idx] > 'Z') && (b[idx] < '0' || b[idx] > '9') {
			return false
		}
	}
	// can only break out with syntaxically correct board name
	*board = string(b[src:idx])
	idx ++
	src = idx
	for ;; idx++ {
		if idx >= len(b) || b[idx] < '0' || b[idx] > '9' {
			break
		}
	}
	if idx > src {
		v, e := strconv.ParseUint(string(b[src:idx]), 10, 64)
		if e == nil {
			*post = v
		}
	}
	*end = idx
	return true
}

func checkLinkPattern(b []byte, src int, end *int, post *uint64) bool {
	return false
}

func (p *postInfo) FMessage() string {
	b := []byte(p.Message)
	var w bytes.Buffer
	src, last := 0, 0

	const (
		tagGreentext = iota
	)
	var tagMap = map[uint]struct{ start, end []byte } {
		tagGreentext: { []byte("<span style=\"color:green\">"), []byte("</span>") },
	}
	var tagList []uint

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
			var board string
			var post uint64
			var end int
			if checkCrossPattern(b, src, &end, &board, &post) {
				if post != 0 {
					esc = []byte(fmt.Sprintf("<a href=\"/%s/thread/%d\">%s%s%s/%s/%d</a>", board, post, htmlGt, htmlGt, htmlGt, board, post))
				} else {
					esc = []byte(fmt.Sprintf("<a href=\"/%s/\">%s%s%s/%s/</a>", board, htmlGt, htmlGt, htmlGt, board))
				}
				inc = end - src
			} else if checkLinkPattern(b, src, &end, &post) {
				esc = []byte(fmt.Sprintf("<a href=\"#%d\">%s%s%d</a>", post, htmlGt, htmlGt, post))
				inc = end - src
			} else if src == 0 || b[src-1] == '\n' {
				esc = append(tagMap[tagGreentext].start, htmlGt...)
				inc = 1
			} else {
				esc = htmlGt
				inc = 1
			}
		case '\n':
			for i := int(len(tagList)-1); i >= 0; i-- {
				if tagList[i] == tagGreentext {
					for j := int(len(tagList)-1); j >= i; j-- {
						esc = append(esc, tagMap[tagList[j]].end...)
					}
					tagList = tagList[:i]
				}
			}
			esc = append(esc, htmlBr...)
			inc = 1
		case '\r':
			inc = 1 // just skip it
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
	for i := int(len(tagList)-1); i >= 0; i-- {
		w.Write(tagMap[tagList[i]].end)
	}
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
