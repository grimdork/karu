package karu

import (
	"context"
	"database/sql"
	"fmt"
)

func (k *Karu) GetPreference(ctx context.Context, userID, key string) (string, error) {
	var value string
	err := k.db.QueryRowContext(ctx,
		`SELECT value FROM preferences WHERE user_id = $1 AND key = $2`,
		userID, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("getting preference: %w", err)
	}
	return value, nil
}

func (k *Karu) SetPreference(ctx context.Context, userID, key, value string) error {
	_, err := k.db.ExecContext(ctx,
		`INSERT INTO preferences (user_id, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = $3`,
		userID, key, value)
	if err != nil {
		return fmt.Errorf("setting preference: %w", err)
	}
	return nil
}

func (k *Karu) DeletePreference(ctx context.Context, userID, key string) error {
	_, err := k.db.ExecContext(ctx,
		`DELETE FROM preferences WHERE user_id = $1 AND key = $2`,
		userID, key)
	if err != nil {
		return fmt.Errorf("deleting preference: %w", err)
	}
	return nil
}

func (k *Karu) ListPreferences(ctx context.Context, userID string) (map[string]string, error) {
	rows, err := k.db.QueryContext(ctx,
		`SELECT key, value FROM preferences WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing preferences: %w", err)
	}
	defer rows.Close()

	prefs := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning preference: %w", err)
		}
		prefs[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prefs, nil
}
