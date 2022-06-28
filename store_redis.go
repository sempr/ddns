package ddns

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	TokenString string = "token"
)

type RedisDNSStorage struct {
	cli *redis.Client
	lk  *sync.RWMutex
}

func NewRedisDNSStorage(cli *redis.Client) *RedisDNSStorage {
	var lk sync.RWMutex
	return &RedisDNSStorage{cli: cli, lk: &lk}
}

func (m *RedisDNSStorage) genToken(hostname string) string {
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	hash.Write([]byte(hostname))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (m *RedisDNSStorage) New(ctx context.Context, qname string) string {
	m.lk.Lock()
	defer m.lk.Unlock()
	if rs, err := m.cli.HGet(ctx, qname, TokenString).Result(); err != nil || rs == "" {
		token := m.genToken(qname)
		if ok, err := m.cli.HSetNX(ctx, qname, TokenString, token).Result(); ok && err == nil {
			defer m.cli.Expire(ctx, qname, time.Hour*24*180)
			return token
		}
	}
	return ""
}

func (m *RedisDNSStorage) Valid(ctx context.Context, qname string) bool {
	m.lk.RLock()
	defer m.lk.RUnlock()
	rs, err := m.cli.HGet(ctx, qname, TokenString).Result()
	if err != nil || rs == "" {
		return true
	}
	return false
}

func qtypeToStr(qtype uint16) string {
	return fmt.Sprintf("x%04x", qtype)
}

func (m *RedisDNSStorage) Query(ctx context.Context, qname string, qtype uint16) []string {
	m.lk.RLock()
	defer m.lk.RUnlock()
	if ans, err := m.cli.HGet(ctx, qname, qtypeToStr(qtype)).Result(); err == nil {
		return strings.Split(ans, "|")
	}
	return []string{}
}

func (m *RedisDNSStorage) Update(ctx context.Context, qname, token string, qtype uint16, val []string) []string {
	m.lk.Lock()
	defer m.lk.Unlock()
	r, err := m.cli.HGet(ctx, qname, TokenString).Result()
	if err == nil && r == token {
		qtypeStr := qtypeToStr(qtype)
		if len(val) > 10 {
			val = val[len(val)-10:]
		}
		m.cli.HSet(ctx, qname, qtypeStr, strings.Join(val, "|")).Result()
		defer m.cli.Expire(ctx, qname, time.Hour*24*180)
		return val
	}
	return []string{}
}

func (m *RedisDNSStorage) Append(ctx context.Context, qname, token string, qtype uint16, val []string) []string {
	m.lk.Lock()
	defer m.lk.Unlock()
	r, err := m.cli.HMGet(ctx, qname, TokenString, qtypeToStr(qtype)).Result()
	if err == nil && r[0].(string) == token {
		var old []string
		if r[1] != nil {
			old = strings.Split(r[1].(string), "|")
		}
		s := make(map[string]bool)
		for _, v := range old {
			s[v] = true
		}
		for _, v := range val {
			if _, ok := s[v]; !ok {
				old = append(old, v)
				s[v] = true
			}
		}
		if len(old) > 10 {
			old = old[len(old)-10:]
		}
		m.cli.HSet(ctx, qname, qtypeToStr(qtype), strings.Join(old, "|")).Result()
		defer m.cli.Expire(ctx, qname, time.Hour*24*180)
		return old
	}
	return []string{}
}

func (m *RedisDNSStorage) Delete(ctx context.Context, qname, token string) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	r, err := m.cli.HGet(ctx, qname, TokenString).Result()
	if err != nil {
		return err
	}
	if r == token {
		_, err = m.cli.Del(ctx, qname).Result()
		return err
	} else {
		return errors.New("token not match")
	}
}
