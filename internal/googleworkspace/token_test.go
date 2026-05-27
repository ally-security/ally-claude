package googleworkspace

import (
	"testing"
	"time"
)

func TestTokenNeedsRefresh(t *testing.T) {
	tok := &Token{
		AccessToken: "x",
		ExpiresAt:   float64(time.Now().Add(2 * time.Minute).Unix()),
	}
	if tok.NeedsRefresh(60 * time.Second) {
		t.Fatal("expected no refresh yet")
	}
	tok.ExpiresAt = float64(time.Now().Add(30 * time.Second).Unix())
	if !tok.NeedsRefresh(60 * time.Second) {
		t.Fatal("expected refresh within skew")
	}
}
