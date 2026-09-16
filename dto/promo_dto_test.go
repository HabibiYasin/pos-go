package dto

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestPromoUsageLimit(t *testing.T) {
	v := validator.New()
	v.SetTagName("binding")
	for _, tc := range []struct {
		name, payload string
		want          *int
		invalid       bool
	}{
		{"omitted", `{}`, nil, false},
		{"unlimited", `{"usage_limit":null}`, nil, false},
		{"exhausted", `{"usage_limit":0}`, promoLimit(0), false},
		{"limited", `{"usage_limit":5}`, promoLimit(5), false},
		{"negative", `{"usage_limit":-1}`, promoLimit(-1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var create CreatePromoDTO
			var update UpdatePromoDTO
			if err := json.Unmarshal([]byte(tc.payload), &create); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.payload), &update); err != nil {
				t.Fatal(err)
			}
			for _, got := range []*int{create.UsageLimit, update.UsageLimit} {
				if (got == nil) != (tc.want == nil) || (got != nil && tc.want != nil && *got != *tc.want) {
					t.Fatalf("usage limit did not preserve null/zero/value: %v", got)
				}
			}
			for _, input := range []any{create, update} {
				if err := v.StructPartial(input, "UsageLimit"); (err != nil) != tc.invalid {
					t.Fatalf("unexpected validation result: %v", err)
				}
			}
		})
	}
}

func promoLimit(value int) *int { return &value }
