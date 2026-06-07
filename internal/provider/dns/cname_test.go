package dns

import (
	"context"
	"fmt"
	"testing"
)

type fakeResolver struct {
	cname string
	err   error
}

func (r fakeResolver) LookupCNAME(context.Context, string) (string, error) {
	return r.cname, r.err
}

func TestCheckCNAMEAcceptsMatchingCanonicalName(t *testing.T) {
	err := CheckCNAME(context.Background(), fakeResolver{cname: "Gateway.Example.Com."}, "app.example.com", "gateway.example.com")
	if err != nil {
		t.Fatalf("CheckCNAME returned error: %v", err)
	}
}

func TestCheckCNAMERejectsMismatch(t *testing.T) {
	err := CheckCNAME(context.Background(), fakeResolver{cname: "other.example.com."}, "app.example.com", "gateway.example.com")
	if err == nil {
		t.Fatal("expected mismatch to fail")
	}
}

func TestCheckCNAMEReturnsLookupError(t *testing.T) {
	err := CheckCNAME(context.Background(), fakeResolver{err: fmt.Errorf("not found")}, "app.example.com", "gateway.example.com")
	if err == nil {
		t.Fatal("expected lookup error to fail")
	}
}
