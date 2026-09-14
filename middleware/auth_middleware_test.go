package middleware

import (
	"net/http"
	"net/http/httptest"
	"pos-go/utils"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTabAuthenticationDoesNotFallBackToAnotherAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	kasirToken, err := utils.GenerateToken("kasir-id", "kasir@example.com", "kasir")
	if err != nil {
		t.Fatal(err)
	}
	kokiToken, err := utils.GenerateToken("koki-id", "koki@example.com", "koki")
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/identity", AuthMiddleware(), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("role"))
	})
	for _, tc := range []struct {
		name, header, cookie, role string
		status                     int
	}{
		{"koki tab overrides kasir cookie", "Bearer " + kokiToken, kasirToken, "koki", 200},
		{"kasir tab overrides koki cookie", "Bearer " + kasirToken, kokiToken, "kasir", 200},
		{"logged out tab ignores cookie", "Bearer", kasirToken, "", 401},
		{"invalid tab token ignores cookie", "Bearer invalid", kasirToken, "", 401},
		{"unsupported auth ignores cookie", "Basic invalid", kasirToken, "", 401},
		{"legacy cookie client", "", kasirToken, "kasir", 200},
		{"no credentials", "", "", "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/identity", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "token", Value: tc.cookie})
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status = %d, want %d", res.Code, tc.status)
			}
			if tc.status == 200 && res.Body.String() != tc.role {
				t.Fatalf("role = %q, want %q", res.Body.String(), tc.role)
			}
		})
	}
}
