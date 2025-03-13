package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/nxtcoder17/go.pkgs/log"
	"github.com/nxtcoder17/ivy"
)

const domainSuffix = ".my-server.com."

type DNSServer struct{}

func (s *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)

	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			ip, valid := parseQuery(q.Name)
			if valid {
				rr, _ := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				msg.Answer = append(msg.Answer, rr)
			}
		}
	}

	w.WriteMsg(msg)
}

func parseQuery(query string) (string, bool) {
	if !strings.HasSuffix(query, domainSuffix) {
		return "", false
	}

	trimmed := strings.TrimSuffix(query, domainSuffix)
	parts := strings.Split(trimmed, ".")
	if len(parts) != 4 {
		return "", false
	}

	ip := net.ParseIP(strings.Join(parts, "."))
	if ip == nil {
		return "", false
	}

	return ip.String(), true
}

func startDoH() *ivy.Router {
	router := ivy.NewRouter()

	router.Post("/dns-query", func(c *ivy.Context) error {
		body, err := io.ReadAll(c.Body())
		if err != nil {
			return err
		}
		defer c.Body().Close()

		msg := new(dns.Msg)
		if err := msg.Unpack(body); err != nil {
			return fmt.Errorf("invalid DNS query")
			// http.Error(w, "Invalid DNS query", http.StatusBadRequest)
			// return
		}

		resp := new(dns.Msg)
		resp.SetReply(msg)
		for _, q := range msg.Question {
			if q.Qtype == dns.TypeA {
				ip, valid := parseQuery(q.Name)
				if valid {
					rr, _ := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
					resp.Answer = append(resp.Answer, rr)
				}
			}
		}

		packed, err := resp.Pack()
		if err != nil {
			return errors.Join(err, fmt.Errorf("failed to pack dns response"))
		}
		c.SetHeader("Content-Type", "application/dns-message")
		return c.SendBytes(packed)
	})

	return router
}

var logger log.Logger

func main() {
	tcpAddr := flag.String("tcp-addr", ":5953", "--tcp-addr [host]:<port>")
	udpAddr := flag.String("udp-addr", ":5953", "--udp-addr [host]:<port>")
	httpAddr := flag.String("http-addr", ":8053", "--http-addr [host]:<port>")

	flag.Parse()

	logger = log.New()

	srv := &DNSServer{}

	udpServer := &dns.Server{Addr: *udpAddr, Net: "udp", Handler: srv}
	tcpServer := &dns.Server{Addr: *tcpAddr, Net: "tcp", Handler: srv}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("starting UDP server at", "addr", *udpAddr)
		if err := udpServer.ListenAndServe(); err != nil {
			logger.Error(err, "failed to start udp server")
			os.Exit(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("starting TCP server at", "addr", *tcpAddr)
		if err := tcpServer.ListenAndServe(); err != nil {
			logger.Error(err, "failed to start tcp server")
			os.Exit(2)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		router := startDoH()
		logger.Info("starting HTTP server at", "addr", *httpAddr)
		http.ListenAndServe(*httpAddr, router)
	}()

	wg.Wait()
}
