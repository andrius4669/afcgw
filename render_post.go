package main

import (
	"fmt"
	"net/url"
	"text/template"
	"time"
)

// single post info
type postInfo struct {
	parent   *threadInfo
	Id       uint64
	Name     string
	Trip     string
	Subject  string
	Email    string
	Date     int64
	Message  string
	File     string
	Original string
	Thumb    string
}

func (p *postInfo) Board() string {
	return p.parent.Board()
}

func (p *postInfo) Thread() uint64 {
	return p.parent.Id
}

func (p *postInfo) IsOp() bool {
	return p.parent.Id == p.Id
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
	if p.HasOriginal() {
		return template.HTMLEscapeString(p.Original)
	}
	return template.HTMLEscapeString(p.File)
}

func (p *postInfo) FullOriginal() string {
	if p.HasOriginal() {
		var u = url.URL{Path: p.FullFile() + "/" + p.Original}
		return u.EscapedPath()
	}
	return p.FullFile()
}

// whether thumb can be displayed for this file
func (p *postInfo) CanThumb() bool {
	return p.Thumb != ""
}

// bit diferent.. whether thumb can be displayed AND is generated from file itself
func (p *postInfo) HasThumb() bool {
	return len(p.Thumb) > 0 && p.Thumb[0] != '/'
}

func (p *postInfo) FullThumb() string {
	if p.HasThumb() {
		return urlThumb(p.Board(), p.Thumb)
	} else {
		return urlStaticThumb(p.Board(), p.Thumb)
	}
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

func (p *postInfo) HasTrip() bool {
	return p.Trip != ""
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

func (p *postInfo) setBoardView(bw bool) {
	p.parent.setBoardView(bw)
}

func (p *postInfo) IsBoardView() bool {
	return p.parent.IsBoardView()
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

type fullPostInfo struct {
	postInfo
	FMessage   string
	fparent    *fullThreadInfo
	References []postReference
}
