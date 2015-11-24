package main

import (
	"fmt"
	"net/http"
)

func postNewBoard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "supposed to create new board...")
}

func postNewThread(w http.ResponseWriter, r *http.Request, board string) {
	fmt.Fprintf(w, "supposed to create new thread in /%s/", board)
}

func postNewPost(w http.ResponseWriter, r *http.Request, board string, thread uint64) {
	fmt.Fprintf(w, "supposed to create new post in /%s/ #%d", board, thread)
}
