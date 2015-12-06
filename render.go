package main

import (
	"net/http"
)


type fullPostInfo struct {
	postInfo
	FMessage   string
	fparent    *fullThreadInfo
	References []uint64 // posts IDs who refered to this post
}

type fullThreadInfo struct {
	threadInfo
	Op      fullPostInfo
	Replies []fullPostInfo
	postMap map[uint64]*fullPostInfo
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

	execTemplate(w, "front", &f)
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
	b.setBoardView(true)
	for i := range b.Threads {
		processThread(&b.Threads[i], db)
	}

	execTemplate(w, "board", &b)
}

func renderThread(w http.ResponseWriter, r *http.Request, board string, thread uint64, mod bool) {
	db := openSQL()
	defer db.Close()

	var t fullThreadInfo
	t.postMap = make(map[uint64]*fullPostInfo)
	if !inputPosts(db, &t, board, thread) {
		http.NotFound(w, r)
		return
	}
	t.setMod(mod)
	t.setBoardView(false)
	processThread(&t, db)

	execTemplate(w, "thread", &t)
}
