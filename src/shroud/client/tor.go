// +build tor

package client

/*
#cgo CFLAGS: -Icdeps/tor-0.2.4.21/ -Icdeps/tor-0.2.4.21/src/common/ -Icdeps/tor-0.2.4.21/src/or -Icdeps/tor-0.2.4.21/src/ext/
#include "or.h"
#include "main.h"
*/
import "C"

func TorMain() int {
	return int(C.tor_main(C.int(0), nil))
}
