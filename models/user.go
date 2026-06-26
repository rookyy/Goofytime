package models

import (
	"database/sql"
	"time"

	"goofytime/database"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

func GetUserByID(id int) (*User, error) {
	u := &User{}
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, is_admin, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := database.DB.QueryRow(
		"SELECT id, username, password_hash, is_admin, created_at FROM users WHERE username = ?", username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func GetAllUsers() ([]User, error) {
	rows, err := database.DB.Query("SELECT id, username, password_hash, is_admin, created_at FROM users ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func CreateUser(username, passwordHash string, isAdmin bool) (*User, error) {
	result, err := database.DB.Exec(
		"INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)",
		username, passwordHash, isAdmin,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetUserByID(int(id))
}

func UpdateUser(id int, username, passwordHash string, isAdmin bool) error {
	if passwordHash != "" {
		_, err := database.DB.Exec(
			"UPDATE users SET username = ?, password_hash = ?, is_admin = ? WHERE id = ?",
			username, passwordHash, isAdmin, id,
		)
		return err
	}
	_, err := database.DB.Exec(
		"UPDATE users SET username = ?, is_admin = ? WHERE id = ?",
		username, isAdmin, id,
	)
	return err
}

func DeleteUser(id int) error {
	_, err := database.DB.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func UserExists(username string) (bool, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetUserCount() (int, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// UserCount is an exported alias for middleware access
var UserCount = GetUserCount

func GetEntryCountForUser(userID int) (int, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM time_entries WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func GetTotalHoursForUser(userID int) (float64, error) {
	var total sql.NullFloat64
	err := database.DB.QueryRow("SELECT SUM(hours) FROM time_entries WHERE user_id = ?", userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Float64, nil
	}
	return 0, nil
}
