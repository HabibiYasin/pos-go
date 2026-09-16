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
			if !tc.allowed && (err == nil || err.Error() != promoValidDaysMessage(tc.days)) {
				t.Fatalf("expected invalid day, got %v", err)
			}
		})
	}
}

func TestPromoValidDaysMessage(t *testing.T) {
	for _, tc := range []struct {
		days int
		want string
	}{
		{32, "voucher hanya berlaku hari: jumat"},
		{62, "voucher hanya berlaku hari senin-jumat"},
		{65, "voucher hanya berlaku weekend"},
		{10, "voucher hanya berlaku hari: senin, rabu"},
		{33, "voucher hanya berlaku hari: jumat, minggu"},
		{1, "voucher hanya berlaku hari: minggu"},
	} {
		if got := promoValidDaysMessage(tc.days); got != tc.want {
			t.Errorf("days %d: got %q, want %q", tc.days, got, tc.want)
		}
	}
}

func TestPromoBranchValidation(t *testing.T) {
	for _, tc := range []struct {
		allowed      int
		branch, want string
	}{
		{6, "jakarta-selatan", "Voucher hanya bisa digunakan di cabang: Depok dan Tokyo"},
		{2, "tokyo", "Voucher hanya bisa digunakan di cabang: Depok"},
		{5, "depok", "Voucher hanya bisa digunakan di cabang: Jakarta Selatan dan Tokyo"},
		{6, "depok", ""}, {6, "tokyo", ""},
		{7, "jakarta-selatan", ""}, {7, "depok", ""}, {7, "tokyo", ""},
		{7, "", "Pilih cabang yang valid"}, {7, "invalid", "Pilih cabang yang valid"},
	} {
		err := validatePromoBranch(tc.allowed, tc.branch)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != tc.want {
			t.Errorf("mask %d branch %s: got %q want %q", tc.allowed, tc.branch, got, tc.want)
		}
	}
}
