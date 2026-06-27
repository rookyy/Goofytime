package handlers

import (
	"net/http"
	"strconv"

	"goofytime/middleware"
	"goofytime/models"
)

func SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	chatID, _ := models.GetTelegramChatID(user.ID)
	message := r.URL.Query().Get("message")
	adminSettings := models.GetAdminSettings()

	RenderTemplate(w, "settings.html", map[string]interface{}{
		"Title":            "Einstellungen",
		"User":             user,
		"ChatID":           chatID,
		"TelegramBotToken": adminSettings.TelegramBotToken,
		"Message":          message,
	})
}

func SaveSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	chatID := r.FormValue("chat_id")
	telegramBotToken := r.FormValue("telegram_bot_token")

	if chatID != "" {
		if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
			http.Redirect(w, r, "/settings?message=Ungültige+Chat-ID", http.StatusSeeOther)
			return
		}
		models.SetTelegramChatID(user.ID, chatID)
	}

	if telegramBotToken != "" {
		models.SaveTelegramBotToken(telegramBotToken)
	}

	http.Redirect(w, r, "/settings?message=Einstellungen+gespeichert", http.StatusSeeOther)
}
