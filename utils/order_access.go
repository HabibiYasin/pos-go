package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// Separate capability for one customer's order; never usable as a staff session.
func OrderAccessToken(id, key string, expires time.Time) string {
	stamp := strconv.FormatInt(expires.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte("order-access:" + id + ":" + stamp))
	return stamp + "." + hex.EncodeToString(mac.Sum(nil))
}

func ValidOrderAccessToken(id, key, token string) bool {
	if key == "" {
		return false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	stamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || stamp <= time.Now().Unix() {
		return false
	}
	return hmac.Equal([]byte(token), []byte(OrderAccessToken(id, key, time.Unix(stamp, 0))))
}
