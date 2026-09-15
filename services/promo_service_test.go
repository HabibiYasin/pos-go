package services

import (
	promo_model "pos-go/models/promo_model"
	"testing"
	"time"
)

func TestPromoEligibility(t *testing.T) {
	now := time.Date(2026, 9, 15, 5, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		change   func(*promo_model.Promo)
		subtotal float64
		want     string
	}{
		{"valid at minimum", func(p *promo_model.Promo) {}, 100000, ""},
		{"expired", func(p *promo_model.Promo) { p.EndDate = now.Add(-time.Second) }, 100000, "Voucher expired"},
		{"not started", func(p *promo_model.Promo) { p.StartDate = now.Add(time.Second) }, 100000, "Promo belum dimulai"},
		{"quota exhausted", func(p *promo_model.Promo) { p.UsageLimit = 5; p.UsageCount = 5 }, 100000, "Batas kuota voucher habis"},
		{"unlimited quota", func(p *promo_model.Promo) { p.UsageCount = 100 }, 100000, ""},
		{"inactive", func(p *promo_model.Promo) { p.IsActive = false }, 100000, "Promo tidak aktif"},
		{"shortfall", func(p *promo_model.Promo) {}, 20000, "Promo tidak terpenuhi (tambahkan Rp80.000)"},
		{"start boundary", func(p *promo_model.Promo) { p.StartDate = now }, 100000, ""},
		{"end boundary", func(p *promo_model.Promo) { p.EndDate = now }, 100000, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			promo := promo_model.Promo{ValidDays: 127, IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour), MinPurchase: 100000}
			tc.change(&promo)
			err := validatePromoEligibility(promo, tc.subtotal, now)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPromoWeekdays(t *testing.T) {
	cases := []struct {
		name, date string
		days       int
		allowed    bool
	}{
		{"weekday Monday", "2026-09-14T12:00:00+07:00", 62, true},
		{"weekday Friday", "2026-09-18T12:00:00+07:00", 62, true},
		{"weekday Saturday", "2026-09-19T12:00:00+07:00", 62, false},
		{"weekday Sunday", "2026-09-20T12:00:00+07:00", 62, false},
		{"Friday only", "2026-09-18T12:00:00+07:00", 32, true},
		{"Friday only Thursday", "2026-09-17T12:00:00+07:00", 32, false},
		{"WIB Friday while UTC Thursday", "2026-09-17T17:00:00Z", 32, true},
		{"WIB Saturday while UTC Friday", "2026-09-18T17:00:00Z", 32, false},
		{"all days Sunday", "2026-09-20T12:00:00+07:00", 127, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, tc.date)
			promo := promo_model.Promo{IsActive: true, ValidDays: tc.days, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour)}
			err := validatePromoEligibility(promo, 100000, now)
			if tc.allowed && err != nil {
				t.Fatal(err)
			}
			if !tc.allowed && err != ErrPromoInvalidDay {
				t.Fatalf("expected invalid day, got %v", err)
			}
		})
	}
}
