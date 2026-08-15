package middleware

import (
	"net/http"
	"os"
)

func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set global headers

		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("FrontEnd_URL"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		// Handle preflight OPTIONS request globally
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass the request down to your actual http.HandleFunc endpoints
		next.ServeHTTP(w, r)
	})
}
