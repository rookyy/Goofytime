package models

import (
	"time"

	"goofytime/database"
)

type ActivityLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

func LogActivity(userID int, action, details string) {
	database.DB.Exec(
		"INSERT INTO activity_log (user_id, action, details) VALUES (?, ?, ?)",
		userID, action, details,
	)
}

func GetActivityLogs(userID int, actionFilter string, limit int) ([]ActivityLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := database.DB.Query(
		"SELECT id, user_id, action, details, created_at FROM activity_log WHERE user_id = ? AND (? = '' OR action = ?) ORDER BY created_at DESC LIMIT ?",
		userID, actionFilter, actionFilter, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ActivityLog
	for rows.Next() {
		var l ActivityLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Details, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func GetDistinctActions(userID int) ([]string, error) {
	rows, err := database.DB.Query(
		"SELECT DISTINCT action FROM activity_log WHERE user_id = ? ORDER BY action",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, nil
}
