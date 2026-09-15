package controllers

import "testing"

func TestPromoDebugTime(t *testing.T) {
	for _, value := range []string{"2026-09-15", "2028-02-29"} {
		got, err := promoDebugTime(value)
		if err != nil || got.Format("2006-01-02") != value {
			t.Fatalf("date %s: %v, %v", value, got, err)
		}
		_, offset := got.Zone()
		if offset != 7*60*60 {
			t.Fatalf("unexpected timezone: %d", offset)
		}
	}
	for _, value := range []string{"2026-02-29", "invalid", "2026-13-01"} {
		if _, err := promoDebugTime(value); err == nil {
			t.Errorf("accepted invalid date %s", value)
		}
	}
	if got, err := promoDebugTime(""); err != nil || got.IsZero() {
		t.Fatal("default date unavailable")
	}
}
