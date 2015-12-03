package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
	"errors"
	"github.com/gographics/imagick/imagick"
	"database/sql"
	"strings"
)

const (
	thumbBgOp    = "red"
	thumbBgReply = "blue"
)

const (
	thumbMaxW = 128
	thumbMaxH = 128
)

const (
	thumbIMagick = iota
	thumbConvert
	thumbGMConvert
)

const (
	thumbDefImageMethod = "imagick/jpg"
)

var thumbDefMethodFormat = map[string]string{
	"imagick":    "jpg",
	"convert":    "jpg",
	"gm-convert": "jpg",
}

func makeIMagickThumb(source, destdir, dest, destext, bgcolor string) error {
	var err error

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	mw.SetResourceLimit(imagick.RESOURCE_MEMORY, 50)

	err = mw.ReadImage(source + "[0]")
	if err != nil {
		return err
	}

	// calculate needed width and height. keep aspect ratio
	w, h := mw.GetImageWidth(), mw.GetImageHeight()
	if w < 1 || h < 1 {
		return errors.New("this image a shit")
	}
	var needW, needH uint
	ratio := float64(w)/float64(h)
	if ratio > 1 {
		needW = thumbMaxW
		needH = uint((thumbMaxH / ratio) + 0.5) // round to near
		if needH < 1 {
			needH = 1
		}
	} else {
		needH = thumbMaxH
		needW = uint((thumbMaxW * ratio) + 0.5) // round to near
		if needW < 1 {
			needW = 1
		}
	}

	err = mw.ThumbnailImage(needW, needH)
	if err != nil {
		return err
	}

	if bgcolor != "" {
		// flatten image to make transparent shit look allright
		pw := imagick.NewPixelWand()
		pw.SetColor(bgcolor)
		err = mw.SetImageBackgroundColor(pw)
		pw.Destroy()
		if err != nil {
			return err
		}
		nmw := mw.MergeImageLayers(imagick.IMAGE_LAYER_FLATTEN)
		if nmw == nil {
			return errors.New("MergeImageLayers failed")
		}
		mw.Destroy()
		mw = nmw
	}

	err = mw.SetImageCompressionQuality(95)
	if err != nil {
		return err
	}

	tmpdest := destdir + "/" + ".tmp." + dest + "." + destext
	fnldest := destdir + "/" + dest + "." + destext
	err = mw.WriteImage(tmpdest)
	if err != nil {
		return err
	}

	os.Rename(tmpdest, fnldest)

	return nil
}

func runConvertCmd(gm bool, source, destdir, dest, destext, bgcolor string) error {
	tmpfile := destdir + "/" + ".tmp." + dest + "." + destext
	dstfile := destdir + "/" + dest + "." + destext

	var runfile string
	var args []string

	if !gm {
		runfile = "convert"
	} else {
		runfile = "gm"
		args = append(args, "convert")
	}

	args = append(args, source + "[0]", "-thumbnail", fmt.Sprintf("%dx%d", thumbMaxW, thumbMaxH))
	if bgcolor != "" {
		args = append(args, "-background", bgcolor, "-flatten")
	}
	args = append(args, "-auto-orient", tmpfile)

	cmd := exec.Command(runfile, args...)
	cmd.Run()
	os.Rename(tmpfile, dstfile)

	return nil // TODO
}

func makeConvertThumb(source, destdir, dest, destext, bgcolor string) error {
	return runConvertCmd(false, source, destdir, dest, destext, bgcolor)
}

func makeGmConvertThumb(source, destdir, dest, destext, bgcolor string) error {
	return runConvertCmd(true, source, destdir, dest, destext, bgcolor)
}

func makeThumb(fullname, fname, board, method string, isop bool) (string, error) {
	var err error

	// empty = automatic
	if method == "" {
		// TODO: determine default method depening on mime type/extension
		method = thumbDefImageMethod
	}

	var format string
	if i := strings.IndexByte(method, '/'); i != -1 {
		method, format = method[:i], method[i+1:]
	}

	if format == "" {
		format = thumbDefMethodFormat[method]
	}

	var bgcolor string
	if isop {
		bgcolor = thumbBgOp
	} else {
		bgcolor = thumbBgReply
	}

	switch method {
		case "imagick":
			err = makeIMagickThumb(fullname, pathThumbDir(board), fname, format, bgcolor)
			if err != nil {
				return "", err
			}
			return fname + "." + format, nil
		case "convert":
			err = makeConvertThumb(fullname, pathThumbDir(board), fname, format, bgcolor)
			if err != nil {
				return "", err
			}
			return fname + "." + format, nil
		case "gm-convert":
			err = makeGmConvertThumb(fullname, pathThumbDir(board), fname, format, bgcolor)
			if err != nil {
				return "", err
			}
			return fname + "." + format, nil
		default:
			return "", errors.New("unknown thumb generation method")
	}
}


func initImageMagick() {
	imagick.Initialize()
}

func killImageMagick() {
	imagick.Terminate()
}

func makeThumbs(method, board, file string) {
	var err error

	db := openSQL()
	defer db.Close()

	var bname string
	err = db.QueryRow("SELECT name FROM boards WHERE name=$1", board).Scan(&bname)
	if err == sql.ErrNoRows {
		fmt.Printf("error: board does not exist")
		return
	}
	panicErr(err)

	var rows *sql.Rows
	if file == "" {
		rows, err = db.Query(fmt.Sprintf("SELECT id, thread, file, thumb FROM %s.posts", board))
	} else {
		rows, err = db.Query(fmt.Sprintf("SELECT id, thread, file, thumb FROM %s.posts WHERE file=$1", board), file)
	}
	panicErr(err)

	type tpost struct {
		id, thread uint64
		file, thumb string
	}

	var modthumbs []tpost

	for rows.Next() {
		var p tpost
		var pthread sql.NullInt64
		err = rows.Scan(&p.id, &pthread, &p.file, &p.thumb)
		panicErr(err)
		// if file does not exist or has special meaning or thumb already has special meaning assigned, don't regenerate
		if p.file != "" && p.file[0] != '/' && (len(p.thumb) < 1 || p.thumb[0] != '/') {
			if !pthread.Valid || pthread.Int64 == 0 || uint64(pthread.Int64) == p.id {
				p.thread = p.id
			} else {
				p.thread = uint64(pthread.Int64)
			}
			modthumbs = append(modthumbs, p)
		}
	}

	fmt.Printf("will regenerate %d thumbs\n", len(modthumbs))

	var total_time uint64 = 0
	for i := range modthumbs {
		var ntname string
		fmt.Printf(">%s", modthumbs[i].file)
		st_time := time.Now().UnixNano()
		ntname, err = makeThumb(pathSrcFile(board, modthumbs[i].file), modthumbs[i].file, board, method, modthumbs[i].id == modthumbs[i].thread)
		ed_time := time.Now().UnixNano()
		spent := uint64(ed_time-st_time)
		if err == nil {
			fmt.Printf(" done: %.3fms\n", float64(spent)/1000000.0)
		} else {
			fmt.Printf(" fail[%s]: %.3fms\n", err, float64(spent)/1000000.0)
		}
		total_time += spent
		if ntname != modthumbs[i].thumb {
			stmt, err := db.Prepare(fmt.Sprintf("UPDATE %s.posts SET thumb = $1 WHERE id = $2", board))
			panicErr(err)
			_, err = stmt.Exec(ntname, modthumbs[i].id)
			panicErr(err)

			if modthumbs[i].thumb != "" {
				os.Remove(pathThumbFile(board, modthumbs[i].thumb))
			}
		}
	}
	fmt.Printf("done. total time spent generating %d thumbs: %.6fs\n", len(modthumbs), float64(total_time)/1000000000.0)
}