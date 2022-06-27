package ddns

import (
	"context"
)

type DNSStorage interface {
	Query(ctx context.Context, qname string, qtype uint16) []string
	New(ctx context.Context, qname string) string
	Valid(ctx context.Context, qname string) bool
	Delete(ctx context.Context, qname, token string) error
	Update(ctx context.Context, qname, token string, qtype uint16, record []string) []string
	Append(ctx context.Context, qname, token string, qtype uint16, record []string) []string
}
