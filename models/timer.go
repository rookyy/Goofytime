package models

import (
	"goofytime/database"
)

type ActiveTimer struct {
	UserID    int    `json:"user_id"`
	StartMs   int64  `json:"start_ms"`
	ElapsedMs int64  `json:"elapsed_ms"`
	IsRunning bool   `json:"is_running"`
	Purpose   string `json:"purpose"`
	ClientID  *int   `json:"client_id"`
}

func GetActiveTimer(userID int) (*ActiveTimer, error) {
	var t ActiveTimer
	var running int
	var clientID *int
	err := database.DB.QueryRow(
		"SELECT user_id, start_ms, elapsed_ms, is_running, purpose, client_id FROM active_timers WHERE user_id = ?",
		userID,
	).Scan(&t.UserID, &t.StartMs, &t.ElapsedMs, &running, &t.Purpose, &clientID)
	if err != nil {
		return nil, err
	}
	t.IsRunning = running == 1
	t.ClientID = clientID
	return &t, nil
}

func SaveActiveTimer(userID int, startMs, elapsedMs int64, isRunning bool, purpose string, clientID *int) error {
	run := 0
	if isRunning {
		run = 1
	}
	_, err := database.DB.Exec(
		`INSERT INTO active_timers (user_id, start_ms, elapsed_ms, is_running, purpose, client_id)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		 start_ms = excluded.start_ms,
		 elapsed_ms = excluded.elapsed_ms,
		 is_running = excluded.is_running,
		 purpose = excluded.purpose,
		 client_id = excluded.client_id`,
		userID, startMs, elapsedMs, run, purpose, clientID,
	)
	return err
}

func DeleteActiveTimer(userID int) {
	database.DB.Exec("DELETE FROM active_timers WHERE user_id = ?", userID)
}
