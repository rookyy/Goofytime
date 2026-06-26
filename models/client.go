package models

import (
	"goofytime/database"
	"time"
)

type Client struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Address         string    `json:"address"`
	Recipients      string    `json:"recipients"`
	MailText        string    `json:"mail_text"`
	MailSubject     string    `json:"mail_subject"`
	HourlyRate      float64   `json:"hourly_rate"`
	ContactName     string    `json:"contact_name"`
	ContactPhone    string    `json:"contact_phone"`
	ContactEmail    string    `json:"contact_email"`
	AutoMailEnabled bool      `json:"auto_mail_enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

func GetClientsByUserID(userID int) ([]Client, error) {
	rows, err := database.DB.Query(
		"SELECT id, user_id, name, COALESCE(email,''), address, COALESCE(recipients,''), COALESCE(mail_text,''), COALESCE(hourly_rate,0), COALESCE(contact_name,''), COALESCE(contact_phone,''), COALESCE(contact_email,''), COALESCE(auto_mail_enabled,1), COALESCE(mail_subject,''), created_at FROM clients WHERE user_id = ? ORDER BY name ASC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Email, &c.Address, &c.Recipients, &c.MailText, &c.HourlyRate, &c.ContactName, &c.ContactPhone, &c.ContactEmail, &c.AutoMailEnabled, &c.MailSubject, &c.CreatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func GetClientByID(id int) (*Client, error) {
	c := &Client{}
	err := database.DB.QueryRow(
		"SELECT id, user_id, name, COALESCE(email,''), address, COALESCE(recipients,''), COALESCE(mail_text,''), COALESCE(hourly_rate,0), COALESCE(contact_name,''), COALESCE(contact_phone,''), COALESCE(contact_email,''), COALESCE(auto_mail_enabled,1), COALESCE(mail_subject,''), created_at FROM clients WHERE id = ?", id,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Email, &c.Address, &c.Recipients, &c.MailText, &c.HourlyRate, &c.ContactName, &c.ContactPhone, &c.ContactEmail, &c.AutoMailEnabled, &c.MailSubject, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func CreateClient(userID int, name, email, address string) (*Client, error) {
	result, err := database.DB.Exec(
		"INSERT INTO clients (user_id, name, email, address) VALUES (?, ?, ?, ?)",
		userID, name, email, address,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetClientByID(int(id))
}

func UpdateClient(id int, name, email, address, recipients, mailText string, hourlyRate float64, contactName, contactPhone, contactEmail string, autoMailEnabled bool, mailSubject string) error {
	_, err := database.DB.Exec(
		"UPDATE clients SET name = ?, email = ?, address = ?, recipients = ?, mail_text = ?, hourly_rate = ?, contact_name = ?, contact_phone = ?, contact_email = ?, auto_mail_enabled = ?, mail_subject = ? WHERE id = ?",
		name, email, address, recipients, mailText, hourlyRate, contactName, contactPhone, contactEmail, autoMailEnabled, mailSubject, id,
	)
	return err
}

func DeleteClient(id int) error {
	_, err := database.DB.Exec("DELETE FROM clients WHERE id = ?", id)
	return err
}
