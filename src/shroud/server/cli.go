package server

import (
	"flag"
)

type Options struct {
	tlsAddr      string
	logto        string
	tunnelAddr   string
	tunnelTLSCrt string
	tunnelTLSKey string
}

func parseArgs() *Options {
	tunnelAddr := flag.String("tunnelAddr", ":4443", "Public address listening for shroud clients")
	tlsAddr := flag.String("tlsAddr", ":443", "Address to listen for TLS connections from the public internet")
	tunnelTLSCrt := flag.String("tunnelTLSCrt", "", "Path to a TLS certificate file")
	tunnelTLSKey := flag.String("tunnelTLSKey", "", "Path to a TLS key file")
	logto := flag.String("log", "stdout", "Write log messages to this file. 'stdout' and 'none' have special meanings")

	flag.Parse()

	return &Options{
		tunnelAddr:   *tunnelAddr,
		tlsAddr:      *tlsAddr,
		tunnelTLSCrt: *tunnelTLSCrt,
		tunnelTLSKey: *tunnelTLSKey,
		logto:        *logto,
	}
}
