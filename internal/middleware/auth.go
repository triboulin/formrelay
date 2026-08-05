package middleware

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuth protège les routes admin via HTTP Basic Auth, en utilisant une
// comparaison à temps constant pour éviter les attaques de timing.
func BasicAuth(user, pass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()

			userMatch := subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(p), []byte(pass)) == 1

			if !ok || !userMatch || !passMatch {
				w.Header().Set("WWW-Authenticate", `Basic realm="FormRelay Admin"`)
				http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
