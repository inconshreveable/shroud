package server

import (
	"github.com/inconshreveable/go-tunnel/log"
	"github.com/inconshreveable/go-tunnel/server"
	"github.com/inconshreveable/go-tunnel/server/binder"
	tunneltls "github.com/inconshreveable/go-tunnel/tls"
	"net/http"
	"time"
)

const (
	muxTimeout = 10 * time.Second
)

func Main() {
	opts := parseArgs()

	// set up logging
	log.LogTo(opts.logto)

	// load the tunnel TLS
	tunnelTLSConfig, err := tunneltls.ServerConfig(opts.tunnelTLSCrt, opts.tunnelTLSKey)
	if err != nil {
		panic(err)
	}

	// setup tls binders
	var binders server.Binders = make(server.Binders)
	if binders["tls"], err = binder.NewTLSBinder(opts.tlsAddr, "", muxTimeout); err != nil {
		panic(err)
	}

	// auto redirect everything from http -> https
	go httpRedirect()

	server, err := server.ServeTLS("tcp", opts.tunnelAddr, tunnelTLSConfig, binders)
	if err != nil {
		panic(err)
	}

	server.Run()
}

func httpRedirect() {
	// permanently redirect all http requests to https
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		u := *(r.URL)
		u.Scheme = "https"
		u.Host = r.Host
		http.Redirect(w, r, u.String(), 301)
	})

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
