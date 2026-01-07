package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func UserIDFromContext(ctx context.Context) (uint64, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(uint64)
	return id, ok
}

func RequireAuth(jwtSvc *JWT) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")

			uid, err := jwtSvc.Verify(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
