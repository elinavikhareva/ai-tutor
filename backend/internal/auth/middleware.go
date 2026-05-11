package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey struct{}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found {
			unauthorized(w)
			return
		}
		c, err := s.parseAccessToken(raw)
		if err != nil {
			unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, c.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserID must only be called from handlers behind Middleware.
func UserID(ctx context.Context) int64 {
	return ctx.Value(ctxKey{}).(int64)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}` + "\n"))
}
