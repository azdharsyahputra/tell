package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

func CORS(allowedOrigins []string, allowCredentials bool) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Idempotency-Key"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	})
}
