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

	err = mw.PingImage(source)
	if err != nil {
		return err
	}

	l, err := mw.GetImageLength() // get length in bytes
	if err != nil {
		return err
	}

	// moar than 50 megs aint allrait for image...
	if l > (50 << 20) {
		return errors.New("unpacked image is bigger than 50 megabytes")
	}

	err = mw.ReadImage(source)
	if err != nil {
		return err
	}

	// set to first frame incase its gif or sth like that
	//mw.SetIteratorIndex(0)

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

	if bgcolor != "" {
		// flatten image to make transparent shit look allright
		pw := imagick.NewPixelWand()
		defer pw.Destroy()
		pw.SetColor(bgcolor)
		err = mw.SetImageBackgroundColor(pw)
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

	err = mw.ThumbnailImage(needW, needH)
	if err != nil {
		return err
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

func makeConvertThumb(source, destdir, dest, destext, bgcolor string) error {
	tmpfile := destdir + "/" + ".tmp." + dest + "." + destext
	dstfile := destdir + "/" + dest + "." + destext
	cmd := exec.Command("convert", source, "-thumbnail", fmt.Sprintf("%dx%d", thumbMaxW, thumbMaxH), "-auto-orient", "+profile", "*", tmpfile)
	cmd.Run()
	os.Rename(tmpfile, dstfile)
	return nil
}

func makeGmConvertThumb(source, destdir, dest, destext, bgcolor string) error {
	tmpfile := destdir + "/" + ".tmp." + dest + "." + destext
	dstfile := destdir + "/" + dest + "." + destext
	cmd := exec.Command("gm", "convert", source, "-thumbnail", fmt.Sprintf("%dx%d", thumbMaxW, thumbMaxH), "-auto-orient", "+profile", "*", tmpfile)
	cmd.Run()
	os.Rename(tmpfile, dstfile)
	return nil
}

func makeThumb(fullname, fname, board, method string) (string, error) {
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
	switch method {
		case "imagick":
			err = makeIMagickThumb(fullname, pathThumbDir(board), fname, format, "")
			if err != nil {
				return "", err
			}
			return fname + "." + format, nil
		case "convert":
			err = makeConvertThumb(fullname, pathThumbDir(board), fname, format, "")
			if err != nil {
				return "", err
			}
			return fname + "." + format, nil
		case "gm-convert":
			err = makeGmConvertThumb(fullname, pathThumbDir(board), fname, format, "")
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
		ntname, err = makeThumb(pathSrcFile(board, modthumbs[i].file), modthumbs[i].file, board, method)
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