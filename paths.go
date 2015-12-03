package main

import "path"

// src - where received original files are stored
func pathSrcDir(board string) string {
	return "files/" + board + "/src"
}
func pathSrcFile(board, file string) string {
	return pathSrcDir(board) + "/" + file
}

// thumb - where generated thumbnails are stored
func pathThumbDir(board string) string {
	return "files/" + board + "/thumb"
}
func pathThumbFile(board, file string) string {
	return pathThumbDir(board) + "/" + file
}

// static - where static (non-changing images, css, etc) files are stored
func pathStaticDir(board string) string {
	if board == "" {
		return "files/static"
	} else {
		return "files/" + board + "/static"
	}
}
func pathStaticFile(board, file string) string {
	return pathStaticDir(board) + "/" + file
}
func pathStaticSafeFile(board, file string) string {
	// return file safelly rooted in static dir
	return pathStaticDir(board) + path.Clean("/" + file)
}