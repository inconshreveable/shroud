package discover

// XXX: use DNSSEC for all of the queries

import (
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"io"
	"net/http"
	"strings"
)

var (
	defaultClientConfig *dns.ClientConfig
	defaultNS           string
	opts                struct {
		listenAddr  string
		tlsCrt      string
		tlsKey      string
		proxyDomain string
	}
)

func init() {
	defaultClientConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		panic(err)
	}

	defaultNS = fmt.Sprintf("%s:%s", defaultClientConfig.Servers[0], defaultClientConfig.Port)
}

func Main() {
	// read command line options
	parseArgs()

	// there's just one endpoint for discovering your public proxies
	http.HandleFunc("/proxies", proxiesHandler)

	// run the server
	err := http.ListenAndServeTLS(opts.listenAddr, opts.tlsCrt, opts.tlsKey, nil)
	if err != nil {
		panic(err)
	}
}

func parseArgs() {
	listenAddr := flag.String("listenAddr", ":443", "Address to listen on")
	tlsCrt := flag.String("tlsCrt", "", "Path to TLS certificate")
	tlsKey := flag.String("tlsKey", "", "Path to TLS key")
	proxyDomain := flag.String("proxyDomain", "proxy.v1.shroud.io", "The base domain for the public proxies")
	flag.Parse()
	opts.listenAddr = *listenAddr
	opts.tlsCrt = *tlsCrt
	opts.tlsKey = *tlsKey
	opts.proxyDomain = *proxyDomain
}

func proxiesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	// first things first, check if the CNAME is pointed correctly
	invalid, err := isCNAMECorrect(q, opts.proxyDomain)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error while validating CNAME for '%v': %v", q, err), 500)
		return
	} else if invalid != nil {
		http.Error(w, invalid.Error(), 403)
		return
	}

	// okay, the CNAME is right, let's look up which proxy servers they should connect to
	addrs, err := getProxyServers(q, opts.proxyDomain)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error while getting proxy servers for '%v': %v", q, err), 500)
		return
	}

	// write out the addresses as a simple newline-delimited list
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, strings.Join(addrs, "\n"))
}

// Does a DNS lookup to ensure that the shrouded service's domain is properly CNAME'd.
// If the domain is x.com and the proxy domain is proxy.shroud.io, we verify that
// x.com -> x.com.proxy.shroud.io
func isCNAMECorrect(domain string, proxyDomain string) (invalid error, err error) {
	correctTarget := domain + "." + proxyDomain

	// cname query
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeCNAME)

	// ask the server
	in, err := dns.Exchange(m, defaultNS)
	if err != nil {
		return
	}

	// no cname records at all
	if len(in.Answer) == 0 {
		invalid = fmt.Errorf("No CNAME record for %v, create one which targets %v\n", domain, correctTarget)
		return
	}

	// use the first one (can't have multiple CNAMEs anyways)
	rr, ok := in.Answer[0].(*dns.CNAME)
	if !ok {
		err = fmt.Errorf("CNAME record is not the right type: %v", rr)
		return
	}

	// wrong target?
	target := unFqdn(rr.Target)
	if target != correctTarget {
		invalid = fmt.Errorf("CNAME record points to wrong target '%v', should point to '%v'", target, correctTarget)
	}

	return
}

// returns the addresses of ALL shroud SRV records for the domain
func getProxyServers(domain string, proxyDomain string) (addrs []string, err error) {
	m := new(dns.Msg)
	q := fmt.Sprintf("_shroud._tls.%s.%s", domain, proxyDomain)
	m.SetQuestion(dns.Fqdn(q), dns.TypeSRV)

	in, err := dns.Exchange(m, defaultNS)
	if err != nil {
		return nil, err
	}

	// my DNS provider won't let me create wildcard SRV records, and I'm not even sure they
	// work anyways. As a workaround, just explicitly fallback.
	// If there are no SRV records for _shroud._tls.x.com.proxy.shroud.io just use the
	// ones for _shroud._tls.proxy.shroud.io
	if len(in.Answer) == 0 {
		q = fmt.Sprintf("_shroud._tls.%s", proxyDomain)
		m.SetQuestion(dns.Fqdn(q), dns.TypeSRV)
		in, err = dns.Exchange(m, defaultNS)
		if err != nil {
			return nil, err
		}

		// our DNS is misconfigured if we couldn't get any records, bail out
		if len(in.Answer) == 0 {
			return nil, fmt.Errorf("No shroud SRV records found for '%v'", q)
		}
	}

	// dump out the SRV records as addr:port strings
	// XXX: we are completely ignoring weight and priority
	// weight doesn't make much sense for persistent connections
	// priority might make some sense though
	result := make([]string, len(in.Answer))
	for i, rr := range in.Answer {
		srv, ok := rr.(*dns.SRV)
		if !ok {
			return nil, fmt.Errorf("Couldn't cast to SRV record: %v!", rr)
		}
		result[i] = fmt.Sprintf("%s:%v", unFqdn(srv.Target), srv.Port)
	}

	return result, nil
}

// chop off the trailing period of an FQDN
func unFqdn(d string) string {
	return d[:len(d)-1]
}
