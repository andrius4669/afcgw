package main

import (
//	"fmt"
	"net/http"
)

// basic info about board
type boardInfo struct {
	Name string
	Desc string
	Info string
}

// front page info
type frontData struct {
	Boards []boardInfo
}

// single post info
type postInfo struct {
	Board   string // redundant, but fuck templates
	Thread  uint64 // again, redundant
	Id      uint64
	Name    string
	Subject string
	Email   string
	Date    uint64
	Message string
	File    string
}

func (p *postInfo) HasFile() bool {
	return p.File != ""
}

// thread with all its posts
type threadInfo struct {
	Board   string
	Id      uint64
	Op      postInfo
	Replies []postInfo
}

// all threads in board + info about board
type threadsInfo struct {
	boardInfo
	Threads []threadInfo
}

// all posts in thread + info about board + info about thread
type postsInfo struct {
	boardInfo
	threadInfo
}


func renderFront(w http.ResponseWriter, r *http.Request) {
	db := openSQL()
	defer db.Close()

	var f frontData
	inputBoards(db, &f)

	execTemplate(w, "boards", &f)
}

func renderBoard(w http.ResponseWriter, r *http.Request, board string) {
	db := openSQL()
	defer db.Close()

	var b threadsInfo
	if !inputThreads(db, &b, board) {
		http.NotFound(w, r)
		return
	}
	execTemplate(w, "threads", &b)
}

func renderThread(w http.ResponseWriter, r *http.Request, board string, thread uint64) {
	db := openSQL()
	defer db.Close()

	var t postsInfo
	if !inputPosts(db, &t, board, thread) {
		http.NotFound(w, r)
		return
	}

	execTemplate(w, "posts", &t)
}
