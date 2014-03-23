// +build !tor

package client

func TorMain() int {
	panic("Tor not compiled in. Install Tor separately and specify -torAddr, or compile with 'make TOR=1'")
}
