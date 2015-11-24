package main

import (
//	"fmt"
	"net/http"
	"strings"
	"os"
	"time"
	"strconv"
)

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, fname string) {
	f, err := os.Open("files/" + fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		http.ServeContent(w, r, fname, time.Time{}, f)
	}
}

type HandlerType struct{}

func (HandlerType) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// shoudln't normally happen, but handle this gracefully
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	if r.Method == "GET" {
		if r.URL.Path == "/" {
			renderFront(w, r)
			return
		}

		board := r.URL.Path[1:]

		idx := strings.IndexByte(board, '/')
		if idx == -1 {
			http.Redirect(w, r, "/" + board + "/", http.StatusFound)
			return
		}

		restype := board[idx+1:]
		board = board[:idx]

		var subinfo string
		if i := strings.IndexByte(restype, '/'); i != -1 {
			restype, subinfo = restype[:i], restype[i:]
		}

		switch restype {
			case "":
				renderBoard(w, r, board)
			case "thread":
				if subinfo == "" || subinfo == "/" {
					http.Redirect(w, r, "/" + board + "/", http.StatusFound)
					return
				}
				n, err := strconv.ParseUint(subinfo[1:], 10, 64)
				if err != nil {
					http.NotFound(w, r)
					return
				}
				renderThread(w, r, board, n)
			case "src":
				if subinfo == "" || subinfo == "/" {
					http.Redirect(w, r, "/" + board + "/", http.StatusFound)
					return
				}
				subinfo = subinfo[1:]
				if i := strings.IndexByte(subinfo, '/'); i != -1 {
					subinfo = subinfo[:i]
				}
				if subinfo == "." || subinfo == ".." {
					http.NotFound(w, r)
					return
				}
				serveFile(w, r, board + "/src/" + subinfo)
			default:
				http.NotFound(w, r)
		}
	} else if r.Method == "POST" {
		board := r.URL.Path[1:]
		var nfunc string
		if i := strings.IndexByte(board, '/'); i != -1 {
			board, nfunc = board[:i], board[i:]
		}
		if board == "newboard" && nfunc == "" {
			postNewBoard(w, r)
			return
		}
		if nfunc == "" || nfunc == "/" {
			http.NotFound(w, r)
			return
		}
		nfunc = nfunc[1:]
		var tfunc string
		if i := strings.IndexByte(nfunc, '/'); i != -1 {
			nfunc, tfunc = nfunc[:i], nfunc[i:]
		}
		if nfunc != "thread" || tfunc == "" || tfunc == "/" {
			http.NotFound(w, r)
			return
		}
		if tfunc == "/new" {
			postNewThread(w, r, board)
		} else {
			n, err := strconv.ParseUint(tfunc[1:], 10, 64)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			postNewPost(w, r, board, n)
		}
	} else {

	}
}

func main() {
	loadTemplates()
	http.ListenAndServe(":1337", &HandlerType{})
}
