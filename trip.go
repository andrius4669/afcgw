package main

import (
	   "regexp"
	ej "golang.org/x/text/encoding/japanese"
	tr "golang.org/x/text/transform"
	ud "./unixdes"
)

func encodeSJIS(src []byte) []byte {
	enc := ej.ShiftJIS.NewEncoder()
	var dst = make([]byte, 16)
	var ndst int
	for {
		enc.Reset()
		var err  error
		ndst, _, err = enc.Transform(dst, src, true)
		if err == tr.ErrShortDst {
			newlen := 2 * len(dst)
			if newlen > 1024 {
				break
			}
			dst = make([]byte, newlen)
			continue
		}
		break
	}
	return dst[:ndst]
}



// the most fucking basic tr-like functionality, because fuck go std shit
func myTr(buf []byte, old, rep string) {
	buflen, oldlen, replen := len(buf), len(old), len(rep)
	if replen == 0 {
		return // get the fuck out, dunno how to handle this shit yet
	}
	for i := 0; i < buflen; i++ {
		for j := 0; j < oldlen; j++ {
			if buf[i] == old[j] {
				buf[i] = rep[j % replen]
				break
			}
		}
	}
}

func MakeTrip(src string) (string, string) {
	match := regexp.MustCompile("^([^#]+)?#(.+)$").FindSubmatch([]byte(src))
	if match == nil {
		return src, ""
	}
	name := string(match[1])
	trip := encodeSJIS(match[2])
	salt := append(trip, []byte("H..")...)[1:3]
	salt = regexp.MustCompile("[^.-z]").ReplaceAll(salt, []byte{'.'})
	myTr(salt, ":;<=>?@[\\]^_`", `ABCDEFGabcdef`)
	res := ud.Crypt(string(trip), string(salt))
	return name, "!" + res[len(res)-10:]
}
