// +build !tor

package client

func TorMain() int {
	panic("Tor not compiled in!")
}
