package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/miekg/dns"
	"github.com/nxtcoder17/fastlog"
	flag "github.com/spf13/pflag"
)

var logger *fastlog.Logger

var cfg struct {
	// EchoHosts are those for when `1.2.3.4.host.com` should resolve to `1.2.3.4`, then `host.com` is a echo Host
	EchoHosts map[string]struct{}

	// Hosts is a /etc/hosts like entry mapping where key is a `hostname` and value being an IP address
	Hosts map[string]string

	// WildcardHosts is much like Hosts, but here a hostname could is a wildcard as if it starts with `*`
	WildcardHosts map[string]string

	// UpstreamDNSServers are those where requests are forwarded for a particular host
	UpstreamDNSServers map[string]string

	// FallbackDNSServers are those which are queried when processing a request for a non-configured host
	FallbackDNSServers []string
}

func main() {
	debug := flag.Bool("debug", os.Getenv("DEBUG") == "true", "--debug")
	echoHosts := flag.StringSlice("echo-host", nil, "--echo-host [hostname]")
	hosts := flag.StringSlice("host", nil, "--host [hostname=IP]")
	wildcardHosts := flag.StringSlice("wildcard-host", nil, "--wildcard-host [hostname=IP]")
	upstreamAddr := flag.StringSlice("upstream", nil, "--upstream [HOST=DNS_ADDR]")
	fallbackDNSAddr := flag.StringSlice("fallback-dns", []string{"1.1.1.1", "1.0.0.1"}, "--upstream [DNS_ADDR]")

	addr := flag.String("addr", ":5953", "--addr [host]:<port>")
	flag.Parse()

	cfg.EchoHosts = make(map[string]struct{})
	cfg.Hosts = make(map[string]string)
	cfg.WildcardHosts = make(map[string]string)
	cfg.UpstreamDNSServers = make(map[string]string)

	for _, eh := range *echoHosts {
		cfg.EchoHosts[eh] = struct{}{}
	}

	for _, host := range *hosts {
		sp := strings.SplitN(host, "=", 2)
		if len(sp) != 2 {
			panic(fmt.Sprintf("bad host value (%s), must be of KEY=VALUE format", host))
		}
		cfg.Hosts[sp[0]] = sp[1]
	}

	for _, host := range *wildcardHosts {
		sp := strings.SplitN(host, "=", 2)
		if len(sp) != 2 {
			panic(fmt.Sprintf("bad host value (%s), must be of KEY=VALUE format", host))
		}
		cfg.WildcardHosts[sp[0]] = sp[1]
	}

	for _, host := range *upstreamAddr {
		sp := strings.SplitN(host, "=", 2)
		if len(sp) != 2 {
			panic(fmt.Sprintf("bad upstream addr value (%s), must be of KEY=VALUE format", host))
		}
		cfg.UpstreamDNSServers[sp[0]] = sp[1]
	}

	cfg.FallbackDNSServers = *fallbackDNSAddr

	logger = fastlog.New(fastlog.Options{
		Format:        fastlog.ConsoleFormat,
		ShowDebugLogs: *debug,
		ShowCaller:    true,
		EnableColors:  true,
	})

	srv := &DNSServer{}

	udpServer := &dns.Server{Addr: *addr, Net: "udp", Handler: srv}

	logger.Info("STARTING dns server", "addr", *addr)
	if err := udpServer.ListenAndServe(); err != nil {
		logger.Error("failed to start udp server", "err", err, "addr", *addr)
		os.Exit(1)
	}
}
