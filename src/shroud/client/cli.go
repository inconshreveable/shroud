package client

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

type Options struct {
	configPath  string
	proxyAddrs  string
	discoverUrl string
	domain      string
	torAddr     string
	tlsCrt      string
	tlsKey      string
	logto       string
	localAddr   string
}

func parseArgs() (*Options, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "shroud [options] <domain> <port | address>")
	}

	logto := flag.String("log", "stdout", "File to log to or 'stdout' for console")
	torAddr := flag.String(
		"torAddr",
		"",
		"Address of the Tor SOCKS5 proxy port to use. If empty, shroud will start its own Tor relay")
	tlsCrt := flag.String(
		"tlsCrt",
		"",
		"Optional path to a TLS certificate to decrypt incoming traffic before forwarding to your local service. If empty, no decryption is attempted before forwarding.")
	tlsKey := flag.String(
		"tlsKey",
		"",
		"Optional path to a TLS private key to decrypt incoming traffic before forwarding to your local service. If empty, no decryption is attempted before forwarding.")
	discoverUrl := flag.String("discoverUrl", "https://discover.v1.shroud.io/proxies", "URL to hit when starting up to discover the location of the proxy servers to use. You may skip this by setting it to the empty string and setting proxyAddrs explicitly.")
	proxyAddrs := flag.String("proxyAddrs", "", "Explicit comma-delimited list of public proxies to tunnel through. You probably want to auto-discover with the discoverUrl.")

	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		return nil, fmt.Errorf("You must supply exactly two arguments: a domain to accept connections for and an address or local port to forward to.\n")
	}

	domain, localAddr := args[0], args[1]
	localAddr, err := normalizeAddress(localAddr)
	if err != nil {
		return nil, err
	}

	if *discoverUrl == "" && *proxyAddrs == "" {
		return nil, fmt.Errorf("You must specify -discoverUrl or -proxyAddrs!")
	}

	if *discoverUrl != "" && *proxyAddrs != "" {
		return nil, fmt.Errorf("You must specify only one of -discoverUrl and -proxyAddrs, not both.")
	}

	return &Options{
		discoverUrl: *discoverUrl,
		proxyAddrs:  *proxyAddrs,
		torAddr:     *torAddr,
		logto:       *logto,
		domain:      domain,
		tlsKey:      *tlsKey,
		tlsCrt:      *tlsCrt,
		localAddr:   localAddr,
	}, nil
}

// shamelessly lifted from ngrok's code
func normalizeAddress(addr string) (string, error) {
	// normalize port to address
	if _, err := strconv.Atoi(addr); err == nil {
		addr = ":" + addr
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("'%s' is not a valid address or local port: %s", addr, err.Error())
	}

	if host == "" {
		host = "127.0.0.1"
	}

	return fmt.Sprintf("%s:%s", host, port), nil
}
