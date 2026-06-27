package models

import "goofytime/database"

type AdminSettings struct {
	HeaderTitle      string
	FooterText       string
	TelegramBotToken string
}

func GetAdminSettings() *AdminSettings {
	s := &AdminSettings{HeaderTitle: "Goofytime", FooterText: "Made by Cold-IT"}
	row := database.DB.QueryRow("SELECT header_title, footer_text, telegram_bot_token FROM admin_settings WHERE id = 1")
	row.Scan(&s.HeaderTitle, &s.FooterText, &s.TelegramBotToken)
	s.TelegramBotToken = DecryptPass(s.TelegramBotToken)
	return s
}

func SaveAdminSettings(headerTitle, footerText, telegramBotToken string) {
	if headerTitle == "" {
		headerTitle = "Goofytime"
	}
	if footerText == "" {
		footerText = "Made by Cold-IT"
	}
	encToken := EncryptPass(telegramBotToken)
	database.DB.Exec("UPDATE admin_settings SET header_title = ?, footer_text = ?, telegram_bot_token = ? WHERE id = 1", headerTitle, footerText, encToken)
}

func SaveTelegramBotToken(token string) {
	encToken := EncryptPass(token)
	database.DB.Exec("UPDATE admin_settings SET telegram_bot_token = ? WHERE id = 1", encToken)
}
