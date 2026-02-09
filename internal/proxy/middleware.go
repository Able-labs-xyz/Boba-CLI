package proxy

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// withAuth wraps an http.HandlerFunc with Bearer-token authentication. The
// incoming request must carry an Authorization header whose Bearer value
// matches the proxy's session token. If the token is missing or does not match,
// a 403 Forbidden JSON response is returned.
func (s *ProxyServer) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader || subtle.ConstantTimeCompare([]byte(token), []byte(s.sessionToken)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden"})
			return
		}

		next(w, r)
	}
}
