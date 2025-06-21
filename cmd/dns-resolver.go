package main

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

type DNSServer struct{}

func (s *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)

	if len(r.Question) > 1 {
		logger.Warn("picking first as multiple DNS questions in a query", "len(questions)", len(r.Question))
	}

	q := r.Question[0]

	query := strings.TrimSuffix(q.Name, ".")

	logger.Debug("querying for ", "query", q.String())
	defer logger.Debug("[REPLIED]", "query", q.String())

	switch q.Qtype {
	case dns.TypeA:
		{
			logger.Debug("querying for ", "host", q.Name)
			logger.Debug("trying with upstream dns servers", "host", q.Name)
			for k, addr := range cfg.UpstreamDNSServers {
				if strings.HasSuffix(query, k) {
					if !strings.HasSuffix(addr, ":53") {
						addr += ":53"
					}
					reply, err := dns.Exchange(r, addr)
					if err != nil {
						logger.Error("failed to exchange dns with upstream", "err", err, "upstream", addr)
						msg.SetRcode(r, dns.RcodeNameError)
						w.WriteMsg(msg)
						return
					}
					w.WriteMsg(reply)
					return
				}
			}

			logger.Debug("trying with custom resolver", "host", query)
			ip, err := resolve(query)
			if err != nil {
				logger.Debug("custom resolver", "host", q.Name, "ip", ip, "err", err)
				msg.SetRcode(r, int(dns.ExtendedErrorCodeStaleNXDOMAINAnswer))
				rr, _ := dns.NewRR(fmt.Sprintf("%s SOA ", q.Name))
				msg.Answer = append(msg.Answer, rr)
				w.WriteMsg(msg)
				return
			}

			logger.Debug("custom resolver", "host", q.Name, "ip", ip)
			if ip != "" {
				rr, _ := dns.NewRR(fmt.Sprintf("%s A %s ", q.Name, ip))
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
				logger.Error("failed to exchange dns", "err", err)
			}
			w.WriteMsg(reply)
			// if err != nil {
			// 	msg.SetRcode(r, int(dns.ExtendedErrorCodeStaleNXDOMAINAnswer))
			// 	rr, _ := dns.NewRR(fmt.Sprintf("%s SOA ", q.Name))
			// 	msg.Answer = append(msg.Answer, rr)
			// 	w.WriteMsg(msg)
			// 	return
			// }
			// msg.Answer = append(msg.Answer, reply.Answer...)
			// w.WriteMsg(msg)
			return
		}
	default:
		{
		}
	}

	// w.WriteMsg(msg)
}

func resolve(dnsQuery string) (ip string, err error) {
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

	return "", nil
}
