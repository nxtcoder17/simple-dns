package main

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func newRequestID() string {
	b := make([]byte, 3)
	rand.Read(b)
	return string(fmt.Sprintf("%x", b))
}

type DNSServer struct{}

func (s *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)

	if len(r.Question) > 1 {
		logger.Warn("picking first as multiple DNS questions in a query", "len(questions)", len(r.Question))
	}

	q := r.Question[0]

	query := strings.TrimSuffix(q.Name, ".")

	logger := logger.With("query.type", dns.Type(q.Qtype).String(), "query.host", query, "request.id", newRequestID())

	start := time.Now()
	logger.Debug("[REQUEST] started")
	defer func() {
		logger.Debug("[REPLIED]", "took", fmt.Sprintf("%.2fs", time.Since(start).Seconds()))
	}()

	if q.Qtype != dns.TypeA {
		msg.SetRcode(r, dns.RcodeNotImplemented)
		w.WriteMsg(msg)
		return
	}

	logger.Debug("[step/upstream] dns servers")
	for k, addr := range cfg.UpstreamDNSServers {
		if strings.HasSuffix(query, k) {
			logger.Debug("[step/upstream] found correct upstream", "upstream", k, "@", addr)
			if !strings.HasSuffix(addr, ":53") {
				addr += ":53"
			}
			reply, err := dns.Exchange(r, addr)
			if err != nil {
				logger.Error("[step/upstream] failed to exchange dns with upstream", "err", err, "upstream", k)
				msg.SetRcode(r, dns.RcodeNameError)
				w.WriteMsg(msg)
				return
			}
			logger.Debug("[REPLY]", "reply", reply)
			w.WriteMsg(reply)
			return
		}
	}

	logger.Debug("[step/simple-dns] inbuilt")
	ip, err := resolve(query)
	if err != nil {
		logger.Debug("[step/simple-dns] failed to resolve", "err", err)
		msg.SetRcode(r, int(dns.ExtendedErrorCodeStaleNXDOMAINAnswer))
		rr, _ := dns.NewRR(fmt.Sprintf("%s SOA ", q.Name))
		msg.Answer = append(msg.Answer, rr)
		w.WriteMsg(msg)
		return
	}

	if ip != "" {
		logger.Debug("[step/simple-dns] inbuilt found", "ip", ip)
		rr, err := dns.NewRR(fmt.Sprintf("%s\t3600\t%s\t%s\t%s", q.Name, dns.Class(q.Qclass).String(), dns.Type(q.Qtype).String(), ip))
		if err != nil {
			msg.SetRcode(r, dns.RcodeFormatError)
			w.WriteMsg(msg)
		}
		msg.Answer = append(msg.Answer, rr)
		w.WriteMsg(msg)
		return
	}

	dnsAddr := cfg.FallbackDNSServers[0]
	if !strings.HasSuffix(dnsAddr, ":53") {
		dnsAddr += ":53"
	}
	logger.Debug("trying default dns servers", "dns-server", dnsAddr)
	reply, err := dns.Exchange(r, dnsAddr)
	if err != nil {
		logger.Error("[step/fallback] failed to exchange dns with upstream", "err", err, "fallback", dnsAddr)
		msg.SetRcode(r, dns.RcodeNameError)
		w.WriteMsg(msg)
		return
	}
	w.WriteMsg(reply)
}

func resolve(dnsQuery string) (ip string, err error) {
	// STEP 2: check for hosts
	if ip, ok := cfg.Hosts[dnsQuery]; ok {
		return ip, nil
	}

	// STEP 3: check for wildcard hosts
	for k, ip := range cfg.WildcardHosts {
		if strings.HasSuffix(dnsQuery, k) {
			return ip, nil
		}
	}

	// STEP 1: check for echo request
	sp := strings.SplitN(dnsQuery, ".", 5)
	if len(sp) != 5 {
		return "", nil
		// return "", fmt.Errorf("invalid echo dns request must of format AA.BB.CC.DD.<echo-host>")
	}

	echoHost := sp[4]

	if _, ok := cfg.EchoHosts[echoHost]; ok {
		return dnsQuery[:len(dnsQuery)-len(echoHost)], nil
	}

	return "", nil
}
