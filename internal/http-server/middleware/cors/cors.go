package cors

import (
	"net/http"
)

func New() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func allowOrigin(origin string) bool {
	return true
	switch origin {
	case "http://localhost:8080", "http://127.0.0.1:8080", "http://217.16.16.26:80":
		return true
	default:
		return false
	}
}
