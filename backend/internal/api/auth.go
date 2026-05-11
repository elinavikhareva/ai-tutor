package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	CSRFToken   string `json:"csrf_token"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := db.GetUserByUsername(r.Context(), h.db, req.Username)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		h.respondError(w, r, err)
		return
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		h.logger(r).Warn("failed login attempt", "remote_addr", r.RemoteAddr)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	sess, err := h.newSession(user.ID)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	if err := sess.store(r.Context(), h.db, user.ID); err != nil {
		h.respondError(w, r, err)
		return
	}
	sess.write(w)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.RefreshCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf == "" {
		writeError(w, http.StatusForbidden, "missing csrf token")
		return
	}
	tokenHash, err := auth.HashRefreshToken(cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	stored, err := db.GetRefreshToken(r.Context(), h.db, tokenHash)
	if err != nil {
		h.logger(r).Warn("refresh token lookup failed", "err", err)
		auth.ClearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if stored.UsedAt != nil {
		h.logger(r).Warn("refresh token reuse detected, revoking all user tokens", "user_id", stored.UserID)
		if err := db.DeleteUserRefreshTokens(r.Context(), h.db, stored.UserID); err != nil {
			h.logger(r).Error("revoke refresh tokens", "err", err)
		}
		auth.ClearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if time.Now().After(stored.ExpiresAt) {
		_ = db.DeleteRefreshToken(r.Context(), h.db, tokenHash)
		auth.ClearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}
	if !auth.ValidCSRFToken(csrf, stored.CSRFToken) {
		writeError(w, http.StatusForbidden, "invalid csrf token")
		return
	}

	sess, err := h.newSession(stored.UserID)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	err = h.db.WithTx(r.Context(), func(tx db.Querier) error {
		if err := db.MarkRefreshTokenUsed(r.Context(), tx, tokenHash); err != nil {
			return err
		}
		if err := sess.store(r.Context(), tx, stored.UserID); err != nil {
			return err
		}
		return db.DeleteExpiredRefreshTokens(r.Context(), tx, stored.UserID)
	})
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	sess.write(w)
}

type session struct {
	accessToken  string
	refreshToken string
	refreshHash  string
	csrfToken    string
}

func (h *Handler) newSession(userID int64) (*session, error) {
	var s session
	var err error
	if s.accessToken, err = h.auth.IssueAccessToken(userID); err != nil {
		return nil, err
	}
	if s.refreshToken, s.refreshHash, err = auth.NewRefreshToken(); err != nil {
		return nil, err
	}
	if s.csrfToken, err = auth.NewCSRFToken(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *session) store(ctx context.Context, q db.Querier, userID int64) error {
	expiresAt := time.Now().Add(auth.RefreshTokenTTL)
	return db.CreateRefreshToken(ctx, q, userID, s.refreshHash, s.csrfToken, expiresAt)
}

func (s *session) write(w http.ResponseWriter) {
	auth.SetRefreshCookie(w, s.refreshToken)
	writeJSON(w, http.StatusOK, tokenResponse{AccessToken: s.accessToken, CSRFToken: s.csrfToken})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf == "" {
		writeError(w, http.StatusForbidden, "missing csrf token")
		return
	}
	if cookie, err := r.Cookie(auth.RefreshCookieName); err == nil {
		tokenHash, err := auth.HashRefreshToken(cookie.Value)
		if err == nil {
			stored, err := db.GetRefreshToken(r.Context(), h.db, tokenHash)
			if err == nil {
				if !auth.ValidCSRFToken(csrf, stored.CSRFToken) {
					writeError(w, http.StatusForbidden, "invalid csrf token")
					return
				}
				_ = db.DeleteRefreshToken(r.Context(), h.db, tokenHash)
			}
		}
	}
	auth.ClearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
