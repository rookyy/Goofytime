package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"stundenerfassung/database"
)

type Session struct {
	ID        int
	UserID    int
	Token     string
	ExpiresAt time.Time
}

func CreateSession(userID int) (*Session, error) {
	token := generateToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	_, err := database.DB.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &Session{UserID: userID, Token: token, ExpiresAt: expiresAt}, nil
}

func GetSession(token string) (*Session, error) {
	s := &Session{}
	err := database.DB.QueryRow(
		"SELECT id, user_id, token, expires_at FROM sessions WHERE token = ? AND expires_at > ?",
		token, time.Now(),
	).Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func DeleteSession(token string) {
	database.DB.Exec("DELETE FROM sessions WHERE token = ?", token)
}

func CleanupSessions() {
	database.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
