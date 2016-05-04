package main

type postReference struct {
	parent *postInfo
	Id     uint64
	Tid    uint64
}

func (r *postReference) Board() string {
	return r.parent.Board()
}

func (r *postReference) Thread() uint64 {
	return r.parent.Thread()
}

func (r *postReference) Post() uint64 {
	return r.parent.Id
}

// formatted url of post it refers to
func (r *postReference) Url() string {
	return urlPost(r.Board(), r.Tid, r.Id)
}
