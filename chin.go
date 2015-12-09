package main

import (
	"fmt"
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
	f, err := os.Open(fname)
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

		// some special non-board names
		switch board {
			case "static":
				serveFile(w, r, pathStaticSafeFile("", restype))
				return
		}

		var subinfo string
		if i := strings.IndexByte(restype, '/'); i != -1 {
			restype, subinfo = restype[:i], restype[i:]
		}

		switch restype {
			case "":
				renderBoard(w, r, board, false)
			case "thread":
				if subinfo == "" || subinfo == "/" {
					http.Redirect(w, r, "/" + board + "/", http.StatusFound)
					return
				}
				subinfo = subinfo[1:]
				if i := strings.IndexByte(subinfo, '/'); i != -1 {
					subinfo = subinfo[:i] // ignore / and anything after it
				}
				n, err := strconv.ParseUint(subinfo, 10, 64)
				if err != nil {
					http.NotFound(w, r)
					return
				}
				renderThread(w, r, board, n, false)
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
				serveFile(w, r, pathSrcFile(board, subinfo))
			case "thumb":
				if subinfo == "" || subinfo == "/" {
					http.Redirect(w, r, "/" + board + "/", http.StatusFound)
					return
				}
				subinfo = subinfo[1:]
				if i := strings.IndexByte(subinfo, '/'); i != -1 || subinfo == "." || subinfo == ".." {
					http.NotFound(w, r)
					return
				}
				serveFile(w, r, pathThumbFile(board, subinfo))
			case "mod":
				if subinfo == "" {
					http.Redirect(w, r, "/" + board + "/mod/", http.StatusFound)
					return
				}
				if subinfo == "/" {
					renderBoard(w, r, board, true)
					return
				}
				subinfo = subinfo[1:]
				if i := strings.IndexByte(subinfo, '/'); i != -1 {
					subinfo = subinfo[:i] // ignore / and anything after it
				}
				n, err := strconv.ParseUint(subinfo, 10, 64)
				if err != nil {
					http.NotFound(w, r)
					return
				}
				renderThread(w, r, board, n, true)
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
			http.Error(w, "501 not implemented", 501)
			return
		}
		nfunc = nfunc[1:]
		var tfunc string
		if i := strings.IndexByte(nfunc, '/'); i != -1 {
			nfunc, tfunc = nfunc[:i], nfunc[i:]
		}
		if !(nfunc == "thread" || nfunc == "mod") || tfunc == "" || tfunc == "/" {
			http.NotFound(w, r)
			return
		}
		if tfunc == "/new" {
			postNewThread(w, r, board)
		} else {
			tfunc = tfunc[1:]
			var ttfunc string
			if i := strings.IndexByte(tfunc, '/'); i != -1 {
				tfunc, ttfunc = tfunc[:i], tfunc[i:]
			}
			if !(ttfunc == "/post" || ttfunc == "/deleted")  {
				http.NotFound(w, r)
				return
			}
			if ttfunc == "/post" {
				n, err := strconv.ParseUint(tfunc, 10, 64)
				if err != nil {
					http.NotFound(w, r)
					return
				}
				postNewPost(w, r, board, n)
			}
			if ttfunc == "/deleted" {
				postDelete(w, r, board)
			}
		}
	} else {
		http.Error(w, "501 not implemented", 501)
	}
}

func init() {
	initMime()
}

func main() {
	if len(os.Args) < 2 {
		loadTemplates()

		initImageMagick()
		defer killImageMagick()

		http.ListenAndServe(":1337", &HandlerType{})
	} else {
		cmd := os.Args[1]
		var method string
		if i := strings.IndexByte(cmd, '/'); i != -1 {
			cmd, method = cmd[:i], cmd[i+1:]
		}
		switch(cmd) {
			case "thumb":
				var board string
				if len(os.Args) > 2 {
					board = os.Args[2]
				}
				var file string
				if len(os.Args) > 3 {
					board = os.Args[3]
				}
				makeThumbs(method, board, file)
			case "initdb":
				initDbCmd()
			default:
				fmt.Printf("unknown command: %s\n", cmd)
		}
		return
	}
}
