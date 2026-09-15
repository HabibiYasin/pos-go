package utils

import (
	"testing"
	"time"
)

func TestOrderAccessIsScopedAndExpires(t *testing.T) {
	token := OrderAccessToken("order-a", "secret", time.Now().Add(time.Hour))
	if !ValidOrderAccessToken("order-a", "secret", token) {
		t.Fatal("valid access rejected")
	}
	for _, tc := range []struct{ id, key, token string }{
		{"order-b", "secret", token}, {"order-a", "other", token}, {"order-a", "", token},
		{"order-a", "secret", token + "0"}, {"order-a", "secret", OrderAccessToken("order-a", "secret", time.Now().Add(-time.Minute))},
	} {
		if ValidOrderAccessToken(tc.id, tc.key, tc.token) {
			t.Fatal("invalid access accepted")
		}
	}
}
