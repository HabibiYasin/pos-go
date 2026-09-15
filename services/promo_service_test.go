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
			promo := promo_model.Promo{IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour), MinPurchase: 100000}
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
