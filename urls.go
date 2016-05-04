package main

import "fmt"

var staticThumbsUrls = map[string]string{
	"audio":   "/static/wubz.png",
	"deleted": "/static/removed.jpg",
	"spoiler": "/static/spoiler.jpg",
	"":        "/static/weird.jpg",
}

func urlStaticThumb(board, s string) string {
	if len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	u, ok := staticThumbsUrls[s]
	if !ok {
		fmt.Printf("warning: no static thumb for type \"%s\"\n", s)
	}
	return u
}

func urlThumb(board, s string) string {
	return "/" + board + "/thumb/" + s
}

func urlPost(board string, thread, post uint64) string {
	return fmt.Sprintf("/%s/thread/%d#%d", board, thread, post)
}
