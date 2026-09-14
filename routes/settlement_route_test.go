package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http/httptest"
	"pos-go/utils"
	"strings"
	"testing"
)

func TestSettlementMutationGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SETTLEMENT_DEBUG_RESET", "")
	r := gin.New()
	SettlementRoutes(r)
	for _, tc := range []struct {
		name, method, path, role, body string
		status                         int
	}{
		{"anonymous", "PUT", "/settlement", "", `{}`, 401},
		{"koki cannot edit", "PUT", "/settlement", "koki", `{}`, 403},
		{"koki cannot reset", "DELETE", "/settlement/debug-reset?date=2026-09-14", "koki", ``, 403},
		{"debug disabled", "DELETE", "/settlement/debug-reset?date=2026-09-14", "kasir", ``, 403},
		{"negative cash", "PUT", "/settlement", "kasir", `{"date":"2026-09-14","actual_cash":-1}`, 400},
		{"missing cash", "PUT", "/settlement", "kasir", `{"date":"2026-09-14"}`, 400},
		{"null cash", "PUT", "/settlement", "kasir", `{"date":"2026-09-14","actual_cash":null}`, 400},
		{"create negative", "POST", "/settlement", "kasir", `{"date":"2026-09-14","actual_cash":-1}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.role != "" {
				token, err := utils.GenerateToken(uuid.NewString(), "test@example.com", tc.role)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("got %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}
