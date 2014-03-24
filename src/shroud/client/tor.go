// +build tor

package client

/*
#cgo CFLAGS: -I../../vendor/libevent-2.0.21-stable/build/include/ -I../../vendor/tor-0.2.4.21/ -I../../vendor/tor-0.2.4.21/src/common/ -I../../vendor/tor-0.2.4.21/src/or -I../../vendor/tor-0.2.4.21/src/ext/
#include "or.h"
#include "main.h"
*/
import "C"

func TorMain() int {
	return int(C.tor_main(C.int(0), nil))
}
