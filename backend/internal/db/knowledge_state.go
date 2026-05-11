package db

import (
	"context"
	"time"
)

type KnowledgeEntry struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"-"`
	Tag        string     `json:"tag"`
	Confidence int        `json:"confidence"`
	LastTested *time.Time `json:"last_tested_at"`
}

func UpsertKnowledge(ctx context.Context, q Querier, userID int64, tag string, confidence int) error {
	_, err := q.Exec(ctx,
		`INSERT INTO knowledge_state (user_id, tag, confidence, last_tested_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (user_id, tag)
		 DO UPDATE SET confidence = excluded.confidence, last_tested_at = excluded.last_tested_at`,
		userID, tag, confidence,
	)
	return err
}

func ListKnowledge(ctx context.Context, q Querier, userID int64) ([]*KnowledgeEntry, error) {
	rows, err := q.Query(ctx,
		`SELECT id, user_id, tag, confidence, last_tested_at
		 FROM knowledge_state WHERE user_id = $1 ORDER BY confidence ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []*KnowledgeEntry{}
	for rows.Next() {
		e := &KnowledgeEntry{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.Tag, &e.Confidence, &e.LastTested); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func ListWeakTags(ctx context.Context, q Querier, userID int64, threshold int) ([]string, error) {
	rows, err := q.Query(ctx,
		`SELECT tag FROM knowledge_state
		 WHERE user_id = $1 AND confidence < $2
		 ORDER BY confidence ASC`,
		userID, threshold,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
