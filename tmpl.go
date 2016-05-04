package main

import (
	"io"
	"io/ioutil"
	"text/template"
)

var loadedTemplates *template.Template

var templateNames []struct{ n, f string } = []struct{ n, f string }{
	{"front", "front.tmpl"},
	{"board", "board.tmpl"},
	{"thread", "thread.tmpl"},
	{"post", "post.tmpl"},
	{"posted", "posted.tmpl"},
	{"threadcreated", "threadcreated.tmpl"},
	{"deleted", "deleted.tmpl"},
	{"boardcreated", "boardcreated.tmpl"},
	{"boarddeleted", "boarddeleted.tmpl"},
}

func parseFromFile(t *template.Template, fname string) (*template.Template, error) {
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	return t.Parse(string(buf))
}

func addTemplate(t *template.Template, tname, fname string) (*template.Template, error) {
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	var nt *template.Template
	if t != nil {
		nt = t.New(tname)
	} else {
		nt = template.New(tname)
	}

	return nt.Parse(string(buf))
}

func loadTemplates() {
	t := template.New("main")
	for i := range templateNames {
		if templateNames[i].n == "main" {
			template.Must(parseFromFile(t, templateNames[i].f))
		} else {
			template.Must(addTemplate(t, templateNames[i].n, templateNames[i].f))
		}
	}
	loadedTemplates = t
}

func execTemplate(w io.Writer, name string, data interface{}) {
	t := loadedTemplates
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		panic(err)
	}
}
