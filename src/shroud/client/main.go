package client

import (
	"code.google.com/p/go.net/proxy"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/inconshreveable/go-tunnel/client"
	"github.com/inconshreveable/go-tunnel/log"
	"github.com/inconshreveable/go-tunnel/proto"
	tunneltls "github.com/inconshreveable/go-tunnel/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Service struct {
	log.Logger
	addr      string
	domain    string
	tlsConfig *tls.Config
}

func Main() {
	// parse command line opts
	opts, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// load from configuration file or from command line arguments
	var services []*Service
	if opts.configPath == "" {
		s := &Service{
			Logger: log.NewTaggedLogger(opts.domain),
			addr:   opts.localAddr,
			domain: opts.domain,
		}

		if opts.tlsCrt != "" || opts.tlsKey != "" {
			s.tlsConfig, err = tunneltls.ServerConfig(opts.tlsCrt, opts.tlsKey)
			if err != nil {
				fmt.Printf("Failed to load TLS configuration: %v\n", err)
				os.Exit(1)
			}
		}

		services = []*Service{s}
	} else {
		// XXX: implement config file
	}

	// set up logging
	if opts.logto != "" {
		log.LogTo(opts.logto)
	}

	// start Tor unless you specified an address where it's already running
	if opts.torAddr == "" {
		// XXX: actually, you could use the torrc to override the listening address even for
		// the tor we start "internally" so hard-coding the default is not very robust
		// maybe we should ask you to provide the port to the tor control protocol instead since
		// we'll need that eventually for advanced stuff
		opts.torAddr = "127.0.0.1:9050"
		go TorMain()
		// give tor a second to bind its SOCKS5 port
		time.Sleep(time.Second)
	}

	// first, discover the location of the proxies
	var proxyAddrs []string
	if opts.proxyAddrs != "" {
		proxyAddrs = strings.Split(opts.proxyAddrs, ",")
		fmt.Printf("Using explicit proxy addresses: %v\n", proxyAddrs)
	} else {
		// XXX: obviously services[0] doesn't work for multiple services
		proxyAddrs, err = discoverProxies(services[0].domain, opts.discoverUrl, opts.torAddr)
		if err != nil {
			fmt.Printf("Failed to discover proxies: %v\n", err)
			os.Exit(1)
		}
		//proxyAddrs = []string{"0.us.proxy.v1.shroud.io:4443"}
		fmt.Printf("Discovered public proxy servers at: %v\n", proxyAddrs)
	}

	// establish a tunnel to each public proxy
	for _, proxyAddr := range proxyAddrs {
		// set up tls configuration
		tlsName, _, err := net.SplitHostPort(proxyAddr)
		if err != nil {
			fmt.Printf("Failed to parse proxy address: %v: %v\n", proxyAddr, err)
			os.Exit(1)
		}

		tlsCfg, err := tunneltls.ClientTrusted(tlsName)
		if err != nil {
			fmt.Printf("Failed to create TLS configuration for name %v: %v\n", tlsName, err)
			os.Exit(1)
		}

		// create a socks5 proxy dialer that dials through Tor
		torDialer := client.SOCKS5Dialer("tcp", opts.torAddr, "", "", proxyAddr, tlsCfg)

		// create a tunnel to the proxy
		sess, err := client.NewReconnectingSession(torDialer, nil)
		if err != nil {
			fmt.Printf("Failed to setup tunnel connection to %v: %v\n", proxyAddr, err)
			os.Exit(1)
		}

		// start listening for each shrouded service (you can only have one right now)
		// XXX: this is too simplistic for handling multiple services (what if they have
		// different sets of public proxies?)
		for _, service := range services {
			httpOpts := &proto.TLSOptions{Hostname: service.domain}
			tun, err := sess.ListenTLS(httpOpts, nil)
			if err != nil {
				fmt.Printf("Failed to listen on domain '%v': %v\n", service.domain, err)
				os.Exit(1)
			}

			go proxyTunnel(tun, service)
		}
	}

	select {}
}

func discoverProxies(domain, discoverUrl, torAddr string) ([]string, error) {
	// set up a tor dialer for the discovery request
	dialer, err := proxy.SOCKS5("tcp", torAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("Failed to crate socks5 dialer for discover call: %v", err)
	}

	// take the discover URL and add q=domain
	queryUrl, err := url.Parse(discoverUrl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse discover URL: %v", err)
	}
	queryValues := queryUrl.Query()
	queryValues.Set("q", domain)
	queryUrl.RawQuery = queryValues.Encode()

	// custom TLS config
	tlsCfg, err := tunneltls.ClientTrusted(queryUrl.Host)
	if err != nil {
		return nil, fmt.Errorf("Failed to create TLS configuration for name %v: %v\n", queryUrl.Host, err)
	}

	// make an http client that will speak through the tor dialer
	client := http.Client{Transport: &http.Transport{Dial: dialer.Dial, TLSClientConfig: tlsCfg}}

	// issue the request
	resp, err := client.Get(queryUrl.String())
	if err != nil {
		return nil, fmt.Errorf("Failed to discover proxies: %v", err)
	}
	defer resp.Body.Close()

	// read/parse the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read discover response: %v", err)
	}

	return strings.Split(string(body), "\n"), nil
}

func proxyTunnel(tun *client.Tunnel, s *Service) {
	defer tun.Close()

	for {
		c, err := tun.Accept()
		if err != nil {
			s.Error("Failed to accept connection: %v. Closing tunnel.")
			break
		}
		go proxyConnection(c, s)
	}
}

func proxyConnection(c net.Conn, s *Service) {
	defer c.Close()

	// open a new connection to the address
	privateConn, err := net.Dial("tcp", s.addr)
	if err != nil {
		s.Error("Failed to open private connection: %v", err)
		return
	}
	defer privateConn.Close()

	// Optionally terminate TLS if requested to do so
	if s.tlsConfig != nil {
		c = tls.Server(c, s.tlsConfig)
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	splice := func(lhs, rhs net.Conn) {
		n, err := io.Copy(lhs, rhs)
		s.Error("Finished connection splice from %v to %v after %v bytes with error %v", rhs.RemoteAddr().String(), lhs.RemoteAddr().String(), n, err)
		wg.Done()
	}

	go splice(c, privateConn)
	go splice(privateConn, c)

	// wait for both splice calls to finish
	wg.Wait()
}
