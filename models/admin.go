package models

import "goofytime/database"

type AdminSettings struct {
	HeaderTitle string
	FooterText  string
}

func GetAdminSettings() *AdminSettings {
	s := &AdminSettings{HeaderTitle: "Goofytime", FooterText: "Made by Cold-IT"}
	row := database.DB.QueryRow("SELECT header_title, footer_text FROM admin_settings WHERE id = 1")
	row.Scan(&s.HeaderTitle, &s.FooterText)
	return s
}

func SaveAdminSettings(headerTitle, footerText string) {
	if headerTitle == "" {
		headerTitle = "Goofytime"
	}
	if footerText == "" {
		footerText = "Made by Cold-IT"
	}
	database.DB.Exec("UPDATE admin_settings SET header_title = ?, footer_text = ? WHERE id = 1", headerTitle, footerText)
}
