package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiter_Allow(t *testing.T) {
	rl := NewIPRateLimiter(50 * time.Millisecond)

	if !rl.Allow("1.2.3.4") {
		t.Fatal("première requête devrait être autorisée")
	}
	if rl.Allow("1.2.3.4") {
		t.Fatal("deuxième requête immédiate devrait être refusée")
	}
	if !rl.Allow("5.6.7.8") {
		t.Fatal("une autre IP ne devrait pas être affectée par le rate limit de la première")
	}

	time.Sleep(60 * time.Millisecond)
	if !rl.Allow("1.2.3.4") {
		t.Fatal("après expiration de la fenêtre, la requête devrait être autorisée")
	}
}

func TestIPRateLimiter_RateLimitMiddleware(t *testing.T) {
	rl := NewIPRateLimiter(50 * time.Millisecond)
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.RateLimit(next)

	req := httptest.NewRequest(http.MethodPost, "/f/x", nil)
	req.RemoteAddr = "9.9.9.9:1234"

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("première requête: attendu 200, obtenu %d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("deuxième requête immédiate: attendu 429, obtenu %d", rec2.Code)
	}

	if called != 1 {
		t.Fatalf("le handler suivant ne devrait être appelé qu'une fois, appelé %d fois", called)
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		xrip       string
		want       string
	}{
		{"remote addr simple", "1.2.3.4:5678", "", "", "1.2.3.4"},
		{"x-forwarded-for unique", "1.2.3.4:5678", "10.0.0.1", "", "10.0.0.1"},
		{"x-forwarded-for liste", "1.2.3.4:5678", "10.0.0.1, 10.0.0.2, 10.0.0.3", "", "10.0.0.1"},
		{"x-real-ip", "1.2.3.4:5678", "", "10.0.0.9", "10.0.0.9"},
		{"remote addr sans port", "1.2.3.4", "", "", "1.2.3.4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xrip != "" {
				req.Header.Set("X-Real-IP", tc.xrip)
			}
			got := ClientIP(req)
			if got != tc.want {
				t.Errorf("ClientIP() = %q, attendu %q", got, tc.want)
			}
		})
	}
}
