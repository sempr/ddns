package query

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/sempr/ddns"
)

type DNSQuery struct {
	server *dns.Server
	store  ddns.DNSStorage
	domain string
}

func NewDNSQuery(store ddns.DNSStorage, domain string) *DNSQuery {
	return &DNSQuery{store: store, domain: domain}
}

func (s *DNSQuery) extractHostname(rawQueryName string) (string, error) {
	queryName := strings.TrimRight(strings.ToLower(rawQueryName), ".")
	log.Println(queryName, s.domain)
	hostname := ""
	if strings.HasSuffix(queryName, s.domain) {
		hostname = queryName[:len(queryName)-len(s.domain)]
	}

	if hostname == "" {
		return "", errors.New("query name does not correspond to our domain")
	}

	return hostname, nil
}

func (s *DNSQuery) parseQuery(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			hostname, _ := s.extractHostname(q.Name)
			log.Printf("Query A for %s\n", hostname)

			ans := s.store.Query(context.TODO(), hostname, q.Qtype)

			for _, r := range ans {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, r))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		case dns.TypeTXT:
			log.Printf("Query TXT for %s\n", q.Name)
			rr, err := dns.NewRR(fmt.Sprintf("%s TXT %s", q.Name, "Hello,world!"))
			if err == nil {
				m.Answer = append(m.Answer, rr)
				m.Answer = append(m.Answer, rr)
			}
		case dns.TypeAAAA:
			hostname, _ := s.extractHostname(q.Name)
			log.Printf("Query A for %s\n", hostname)
			ans := s.store.Query(context.TODO(), hostname, q.Qtype)

			for _, r := range ans {
				rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, r))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		default:
			log.Printf("query type: %d %s", q.Qtype, q.Name)
		}
	}
}

func (s *DNSQuery) handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		s.parseQuery(m)
	default:
		log.Println("unknown r.Opcode: ", r.Opcode)
	}

	w.WriteMsg(m)

}

func (s *DNSQuery) Run(port int) error {
	srv := dns.NewServeMux()
	srv.HandleFunc(".", s.handleDnsRequest)
	// start server
	server := &dns.Server{Handler: srv, Addr: ":" + strconv.Itoa(port), Net: "udp"}
	s.server = server
	log.Printf("Starting at %d\n", port)
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
		return err
	}
	return nil
}

func (s *DNSQuery) Stop() error {
	log.Println("DNS Server stopping....")
	defer log.Println("DNS Server stopped....")
	return s.server.Shutdown()
}
