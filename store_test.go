package ddns

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/miekg/dns"
)

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func test1(t *testing.T, s DNSStorage) {
	name := "abc"
	ctx := context.TODO()
	if !s.Valid(ctx, name) {
		t.Errorf("%s valid 1 failed", name)
	}
	token := s.New(ctx, name)
	if s.Valid(ctx, name) {
		t.Errorf("%s valid 2 failed", name)
	}
	defer s.Delete(ctx, name, token)

	s.Update(ctx, name, token, dns.TypeA, []string{"128.0.0.1"})
	r := s.Query(ctx, name, dns.TypeA)
	if !same(r, []string{"128.0.0.1"}) {
		t.Errorf("query 1 failed")
	}

	r = s.Append(ctx, name, token, dns.TypeA, []string{"128.0.0.2"})
	if !same(r, []string{"128.0.0.1", "128.0.0.2"}) {
		t.Errorf("query 2 failed")
	}

	s.Update(ctx, name, token, dns.TypeA, []string{"128.0.0.3"})
	r = s.Query(ctx, name, dns.TypeA)
	if !same(r, []string{"128.0.0.3"}) {
		t.Errorf("query 3 failed")
	}

	s.Append(ctx, name, token, dns.TypeAAAA, []string{"2600::"})
	if !same(s.Query(ctx, name, dns.TypeAAAA), []string{"2600::"}) {
		t.Error("query 4 failed")
	}
	if err := s.Delete(ctx, name, token); err != nil {
		t.Error("delete failed", err)
	}

	if !s.Valid(ctx, name) {
		t.Error("valid 3 failed")
	}
}

func TestRedis(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})
	var s DNSStorage = NewRedisDNSStorage(rdb)
	test1(t, s)

	s = NewMemoryDNSStorage()
	test1(t, s)
}
