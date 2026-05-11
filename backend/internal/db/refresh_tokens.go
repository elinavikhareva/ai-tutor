package db

import (
	"context"
	"time"
)

type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	CSRFToken string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func CreateRefreshToken(ctx context.Context, q Querier, userID int64, tokenHash, csrfToken string, expiresAt time.Time) error {
	_, err := q.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, csrf_token, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		userID, tokenHash, csrfToken, expiresAt,
	)
	return err
}

func GetRefreshToken(ctx context.Context, q Querier, tokenHash string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := q.QueryRow(ctx,
		`SELECT id, user_id, token_hash, csrf_token, expires_at, used_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.CSRFToken, &rt.ExpiresAt, &rt.UsedAt, &rt.CreatedAt)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func DeleteRefreshToken(ctx context.Context, q Querier, tokenHash string) error {
	_, err := q.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}

func DeleteUserRefreshTokens(ctx context.Context, q Querier, userID int64) error {
	_, err := q.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE user_id = $1`,
		userID,
	)
	return err
}

func MarkRefreshTokenUsed(ctx context.Context, q Querier, tokenHash string) error {
	_, err := q.Exec(ctx,
		`UPDATE refresh_tokens SET used_at = now() WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}

func DeleteExpiredRefreshTokens(ctx context.Context, q Querier, userID int64) error {
	_, err := q.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE user_id = $1 AND expires_at < now()`,
		userID,
	)
	return err
}
