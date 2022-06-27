package ddns

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type MemoryDNSStoreage struct {
	store map[string]*MemoryRecord
	lk    *sync.RWMutex
}

type MemoryRecord struct {
	Token string
	A     []string
	AAAA  []string
}

func NewMemoryDNSStorage() *MemoryDNSStoreage {
	m := make(map[string]*MemoryRecord)
	var lk sync.RWMutex
	return &MemoryDNSStoreage{store: m, lk: &lk}
}

func (m *MemoryDNSStoreage) genToken(hostname string) string {
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	hash.Write([]byte(hostname))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (m *MemoryDNSStoreage) New(ctx context.Context, qname string) string {
	if _, ok := (m.store)[qname]; !ok {
		a := new(MemoryRecord)
		a.Token = m.genToken(qname)
		(m.store)[qname] = a
		return a.Token
	}
	return ""
}

func (m *MemoryDNSStoreage) Valid(ctx context.Context, qname string) bool {
	_, ok := (m.store)[qname]
	return !ok
}

func (m *MemoryDNSStoreage) Delete(ctx context.Context, qname, token string) error {
	if r, ok := (m.store)[qname]; ok && r != nil && r.Token == token {
		delete(m.store, qname)
		return nil
	}
	return errors.New("token not match")
}

func (m *MemoryDNSStoreage) Query(ctx context.Context, qname string, qtype uint16) []string {
	if r, ok := (m.store)[qname]; ok {
		switch qtype {
		case dns.TypeA:
			return r.A
		case dns.TypeAAAA:
			return r.AAAA
		}
	}
	return []string{}
}

func (m *MemoryDNSStoreage) Update(ctx context.Context, qname, token string, qtype uint16, val []string) []string {
	if r, ok := (m.store)[qname]; ok && r != nil && r.Token == token {
		switch qtype {
		case dns.TypeA:
			r.A = val
			return val
		case dns.TypeAAAA:
			r.AAAA = val
			return val
		}
	}
	return []string{}
}

func (m *MemoryDNSStoreage) Append(ctx context.Context, qname, token string, qtype uint16, val []string) []string {
	if r, ok := (m.store)[qname]; ok && r != nil && r.Token == token {
		switch qtype {
		case dns.TypeA:
			r.A = append(r.A, val...)
			return r.A
		case dns.TypeAAAA:
			r.AAAA = append(r.AAAA, val...)
			return r.AAAA
		}
	}
	return []string{}
}
