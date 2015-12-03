package main

import (
	"os"
	"errors"
	"github.com/gographics/imagick/imagick"
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
	mw.SetIteratorIndex(0)

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

func makeThumb(fullname, fname, board string) (string, error) {
	var err error

	err = makeIMagickThumb(fullname, pathThumbDir(board), fname, "jpg", "#0000FF")
	if err != nil {
		return "", err
	}
	return fname + ".jpg", nil
}


func initImageMagick() {
	imagick.Initialize()
}

func killImageMagick() {
	imagick.Terminate()
}


func makeThumbs(board, file string) {

}