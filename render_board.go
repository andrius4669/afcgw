package main

// basic info about board
type boardInfo struct {
	Name      string
	Desc      string
	Info      string
	modView   bool
	boardView bool
}

func (b *boardInfo) setMod(mod bool) {
	b.modView = mod
}

func (b *boardInfo) IsMod() bool {
	return b.modView
}

func (b *boardInfo) setBoardView(bw bool) {
	b.boardView = bw
}

func (b *boardInfo) IsBoardView() bool {
	return b.boardView
}