package models

import (
	"strings"

	"goofytime/database"
)

type UserSettings struct {
	UserID      int
	FirstName   string
	LastName    string
	Street      string
	ZipCity     string
	Phone       string
	Email       string
	AutoMailDay int
	PageSize    int
}

func (s *UserSettings) DisplayName() string {
	if s.FirstName != "" || s.LastName != "" {
		return strings.TrimSpace(s.FirstName + " " + s.LastName)
	}
	return ""
}

func GetUserSettings(userID int) (*UserSettings, error) {
	s := &UserSettings{UserID: userID, AutoMailDay: 1, PageSize: 30}
	row := database.DB.QueryRow(
		"SELECT user_id, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(street,''), COALESCE(zip_city,''), COALESCE(phone,''), COALESCE(email,''), auto_mail_day, page_size FROM user_settings WHERE user_id = ?", userID,
	)
	err := row.Scan(&s.UserID, &s.FirstName, &s.LastName, &s.Street, &s.ZipCity, &s.Phone, &s.Email, &s.AutoMailDay, &s.PageSize)
	if err != nil {
		return s, nil
	}
	return s, nil
}

func SaveUserSettings(userID, autoMailDay, pageSize int) error {
	if autoMailDay < 1 || autoMailDay > 28 {
		autoMailDay = 1
	}
	if pageSize < 5 || pageSize > 200 {
		pageSize = 30
	}
	_, err := database.DB.Exec(
		"INSERT INTO user_settings (user_id, auto_mail_day, page_size) VALUES (?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET auto_mail_day = excluded.auto_mail_day, page_size = excluded.page_size",
		userID, autoMailDay, pageSize,
	)
	return err
}

func SaveProfile(userID int, firstName, lastName, street, zipCity, phone, email string) error {
	_, err := database.DB.Exec(
		`INSERT INTO user_settings (user_id, first_name, last_name, street, zip_city, phone, email, auto_mail_day, page_size)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 1, 30)
		 ON CONFLICT(user_id) DO UPDATE SET
		 first_name = excluded.first_name, last_name = excluded.last_name,
		 street = excluded.street,
		 zip_city = excluded.zip_city, phone = excluded.phone, email = excluded.email`,
		userID, firstName, lastName, street, zipCity, phone, email,
	)
	return err
}

func GetAllUserSettings() ([]UserSettings, error) {
	rows, err := database.DB.Query("SELECT user_id, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(street,''), COALESCE(zip_city,''), COALESCE(phone,''), COALESCE(email,''), auto_mail_day, page_size FROM user_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []UserSettings
	for rows.Next() {
		var s UserSettings
		if err := rows.Scan(&s.UserID, &s.FirstName, &s.LastName, &s.Street, &s.ZipCity, &s.Phone, &s.Email, &s.AutoMailDay, &s.PageSize); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}
