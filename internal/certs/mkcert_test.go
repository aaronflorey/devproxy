package certs

import (
	"errors"
	"strings"
	"testing"
)

func TestMKCertIssueReturnsExplicitFailure(t *testing.T) {
	issuer := MKCertIssuer{
		Runner: func(binary string, args ...string) error {
			return errors.New("mkcert failed: permission denied")
		},
	}

	_, err := issuer.Issue("/tmp/devproxy-certs", []string{"acme.test", "*.acme.test"})
	if err == nil {
		t.Fatalf("expected explicit error when mkcert command fails")
	}
	if !strings.Contains(err.Error(), "mkcert") {
		t.Fatalf("expected error to mention mkcert failure, got %v", err)
	}
}
