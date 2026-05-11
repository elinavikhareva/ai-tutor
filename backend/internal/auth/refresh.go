package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
)

const (
	refreshTokenBytes = 32
	RefreshCookieName = "refresh_token"
)

func NewRefreshToken() (token, hash string, err error) {
	b := make([]byte, refreshTokenBytes)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth: generate refresh token: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(b), hex.EncodeToString(sum[:]), nil
}

func HashRefreshToken(token string) (string, error) {
	b, err := hex.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("auth: decode refresh token: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func SetRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, refreshCookie(token, int(RefreshTokenTTL.Seconds())))
}

func ClearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, refreshCookie("", -1))
}

func refreshCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     RefreshCookieName,
		Value:    value,
		Path:     "/api/v1/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}
