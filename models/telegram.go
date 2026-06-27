package models

import "goofytime/database"

type TelegramToken struct {
	ID     int
	UserID int
	ChatID string
}

func GetTelegramChatID(userID int) (string, error) {
	var chatID string
	err := database.DB.QueryRow("SELECT chat_id FROM telegram_tokens WHERE user_id = ?", userID).Scan(&chatID)
	if err != nil {
		return "", err
	}
	return chatID, nil
}

func SetTelegramChatID(userID int, chatID string) error {
	_, err := database.DB.Exec(
		"INSERT INTO telegram_tokens (user_id, chat_id) VALUES (?, ?) ON CONFLICT(user_id) DO UPDATE SET chat_id = excluded.chat_id",
		userID, chatID,
	)
	return err
}

func GetAllTelegramTokens() ([]TelegramToken, error) {
	rows, err := database.DB.Query("SELECT id, user_id, chat_id FROM telegram_tokens")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TelegramToken
	for rows.Next() {
		var t TelegramToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.ChatID); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func GetLastCheckMonthForUser(userID int) string {
	var month string
	database.DB.QueryRow("SELECT last_check_month FROM scheduler_state WHERE id = ?", userID+100000).Scan(&month)
	return month
}

func SetLastCheckMonthForUser(userID int, month string) {
	database.DB.Exec("INSERT OR REPLACE INTO scheduler_state (id, last_check_month) VALUES (?, ?)", userID+100000, month)
}

func GetUserIDByChatID(chatID string) (int, error) {
	var userID int
	err := database.DB.QueryRow("SELECT user_id FROM telegram_tokens WHERE chat_id = ?", chatID).Scan(&userID)
	return userID, err
}
