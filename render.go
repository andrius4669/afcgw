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

func renderFront(w http.ResponseWriter, r *http.Request) {
	db := openSQL()

	var f frontData
	inputBoards(db, &f)

	execTemplate(w, "boards", &f)
}

// single post info
type postInfo struct {
	Id      uint64
	Name    string
	Subject string
	Email   string
	Date    uint64
	Message string
	File    string
}

// thread with all its posts
type threadInfo struct {
	Id      uint64
	Op      postInfo
	Replies []postInfo
}

// all threads in board + info about board
type threadsInfo struct {
	boardInfo
	Threads []threadInfo
}

func renderBoard(w http.ResponseWriter, r *http.Request, board string) {
	db := openSQL()

	var b threadsInfo

	if !inputThreads(db, &b, board) {
		http.NotFound(w, r)
		return
	}

	execTemplate(w, "threads", &b)
}

// all posts in thread + info about board + info about thread
type postsInfo struct {
	boardInfo
	threadInfo
}

// TODO
func renderThread(w http.ResponseWriter, r *http.Request, board string, thread uint64) {
	db := openSQL()

	var t postsInfo
	if !inputPosts(db, &t, board, thread) {
		http.NotFound(w, r)
		return
	}

	execTemplate(w, "posts", &t)
}
