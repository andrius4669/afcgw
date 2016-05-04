package main

type threadInfo struct {
	parent *boardInfo
	Id     uint64
}

func (t *threadInfo) Board() string {
	return t.parent.Name
}

func (t *threadInfo) setMod(mod bool) {
	t.parent.setMod(mod)
}

func (t *threadInfo) IsMod() bool {
	return t.parent.IsMod()
}

func (t *threadInfo) setBoardView(bw bool) {
	t.parent.setBoardView(bw)
}

func (t *threadInfo) IsBoardView() bool {
	return t.parent.IsBoardView()
}

type fullThreadInfo struct {
	threadInfo
	Op      fullPostInfo
	Replies []fullPostInfo
	postMap map[uint64]int
}
