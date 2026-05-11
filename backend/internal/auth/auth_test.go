package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var testSecret = []byte(strings.Repeat("s", 32))

func newService(t *testing.T) *Service {
	t.Helper()
	s, err := New(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewRejectsShortSecret(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
		t.Fatal("expected an error for a short secret")
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse")) != nil {
		t.Error("hash does not match the password")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrong")) == nil {
		t.Error("hash matches a wrong password")
	}
}

func TestMiddleware(t *testing.T) {
	s := newService(t)
	valid, err := s.IssueAccessToken(42)
	if err != nil {
		t.Fatal(err)
	}
	other, err := New([]byte(strings.Repeat("o", 32)))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := other.IssueAccessToken(42)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "Bearer " + valid, http.StatusOK},
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic " + valid, http.StatusUnauthorized},
		{"garbage", "Bearer not-a-jwt", http.StatusUnauthorized},
		{"other secret", "Bearer " + foreign, http.StatusUnauthorized},
		{"expired", "Bearer " + signed(t, time.Now().Add(-time.Hour), jwt.SigningMethodHS256), http.StatusUnauthorized},
		{"alg none", "Bearer " + signed(t, time.Now().Add(time.Hour), jwt.SigningMethodNone), http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser int64
			h := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = UserID(r.Context())
			}))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if tt.want == http.StatusOK && gotUser != 42 {
				t.Errorf("UserID = %d, want 42", gotUser)
			}
		})
	}
}

func signed(t *testing.T, expires time.Time, method jwt.SigningMethod) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expires)},
		UserID:           42,
	})
	key := any(testSecret)
	if method == jwt.SigningMethodNone {
		key = jwt.UnsafeAllowNoneSignatureType
	}
	s, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRefreshTokenHash(t *testing.T) {
	token, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	got, err := HashRefreshToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if got != hash {
		t.Errorf("HashRefreshToken = %q, want %q", got, hash)
	}
	if _, err := HashRefreshToken("not hex"); err == nil {
		t.Error("expected an error for a malformed token")
	}
}

func TestValidCSRFToken(t *testing.T) {
	token, err := NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidCSRFToken(token, token) {
		t.Error("identical tokens are not valid")
	}
	if ValidCSRFToken(token+"x", token) {
		t.Error("different tokens are valid")
	}
}
