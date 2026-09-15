package routes

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentRoutesRejectUnconfiguredCheckoutAndInvalidAccess(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "")
	t.Setenv("server_key_mid", "")
	t.Setenv("MIDTRANS_CLIENT_KEY", "")
	r := gin.New()
	TransactionRoutes(r)
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/transaction/payment-config", "", 200},
		{"GET", "/transaction/00000000-0000-0000-0000-000000000001/customer", "", 403},
		{"POST", "/transaction/notification", `{"order_id":"00000000-0000-0000-0000-000000000001","signature_key":"fake","transaction_status":"settlement"}`, 403},
		{"POST", "/transaction", `{"customer_name":"Test","customer_phone":"081234567890","order_type":"take_away","payment_method":"e_wallet","items":[{"menu_id":"00000000-0000-0000-0000-000000000001","quantity":1}]}`, 503},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("%s %s: got %d want %d", tc.method, tc.path, w.Code, tc.status)
		}
		if tc.path == "/transaction/payment-config" && !strings.Contains(w.Body.String(), `"enabled":false`) {
			t.Fatal("unconfigured payment enabled")
		}
	}
}
