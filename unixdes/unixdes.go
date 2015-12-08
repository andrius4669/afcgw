package unixdes

import (
	"unsafe"
	"sync"
)

// #cgo LDFLAGS: -lcrypt
// #define _XOPEN_SOURCE
// #include <crypt.h>
// #include <stdlib.h>
import "C"

var crypt_m sync.Mutex

func Crypt(key, salt string) string {
	ckey := C.CString(key)
	csalt := C.CString(salt)
	crypt_m.Lock()
	out := C.GoString(C.crypt(ckey, csalt))
	crypt_m.Unlock()
	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(csalt))
	return out
}