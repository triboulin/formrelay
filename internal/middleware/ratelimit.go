package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// IPRateLimiter bloque les requêtes émanant de la même IP si elles arrivent
// à moins de `interval` d'intervalle. Implémentation en mémoire, thread-safe,
// avec nettoyage périodique pour éviter une fuite mémoire.
type IPRateLimiter struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
	interval time.Duration
}

// NewIPRateLimiter crée un limiteur avec la fenêtre donnée (ex: 5s).
func NewIPRateLimiter(interval time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		lastSeen: make(map[string]time.Time),
		interval: interval,
	}
	go rl.cleanupLoop()
	return rl
}

// Allow retourne true si l'IP est autorisée à effectuer une requête maintenant.
func (rl *IPRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if last, ok := rl.lastSeen[ip]; ok {
		if now.Sub(last) < rl.interval {
			return false
		}
	}
	rl.lastSeen[ip] = now
	return true
}

func (rl *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for ip, last := range rl.lastSeen {
			if last.Before(cutoff) {
				delete(rl.lastSeen, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit renvoie un middleware HTTP appliquant la limite par IP.
// En cas de dépassement, répond 429 Too Many Requests.
func (rl *IPRateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r)
		if !rl.Allow(ip) {
			http.Error(w, "429 Too Many Requests: veuillez patienter avant de réessayer", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP extrait l'adresse IP du client, en tenant compte de X-Forwarded-For
// (utile derrière un reverse proxy).
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Peut contenir une liste "client, proxy1, proxy2" - on prend le premier.
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
