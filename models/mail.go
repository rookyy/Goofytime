package models

import "stundenerfassung/database"

type MailSettings struct {
	UserID          int
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	DefaultMailText string
	EmailSubject    string
}

func GetMailSettings(userID int) (*MailSettings, error) {
	s := &MailSettings{UserID: userID, SMTPPort: 587}
	var encPass string
	err := database.DB.QueryRow(
		"SELECT user_id, smtp_host, smtp_port, smtp_user, smtp_pass, smtp_from, COALESCE(default_mail_text,''), COALESCE(email_subject,'') FROM mail_settings WHERE user_id = ?",
		userID,
	).Scan(&s.UserID, &s.SMTPHost, &s.SMTPPort, &s.SMTPUser, &encPass, &s.SMTPFrom, &s.DefaultMailText, &s.EmailSubject)
	if err != nil {
		return s, nil
	}
	s.SMTPPass = DecryptPass(encPass)
	return s, nil
}

func SaveMailSettings(userID int, host string, port int, user, pass, from, defaultMailText, emailSubject string) error {
	encPass := EncryptPass(pass)
	_, err := database.DB.Exec(
		`INSERT INTO mail_settings (user_id, smtp_host, smtp_port, smtp_user, smtp_pass, smtp_from, default_mail_text, email_subject)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		 smtp_host = excluded.smtp_host, smtp_port = excluded.smtp_port,
		 smtp_user = excluded.smtp_user, smtp_pass = excluded.smtp_pass,
		 smtp_from = excluded.smtp_from, default_mail_text = excluded.default_mail_text,
		 email_subject = excluded.email_subject`,
		userID, host, port, user, encPass, from, defaultMailText, emailSubject,
	)
	return err
}
