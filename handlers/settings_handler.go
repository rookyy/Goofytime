package handlers

import (
	"net/http"
	"strconv"

	"stundenerfassung/middleware"
	"stundenerfassung/models"
)

func SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	chatID, _ := models.GetTelegramChatID(user.ID)
	message := r.URL.Query().Get("message")

	RenderTemplate(w, "settings.html", map[string]interface{}{
		"Title":   "Einstellungen",
		"User":    user,
		"ChatID":  chatID,
		"Message": message,
	})
}

func SaveSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	chatID := r.FormValue("chat_id")

	if chatID != "" {
		if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
			http.Redirect(w, r, "/settings?message=Ungültige+Chat-ID", http.StatusSeeOther)
			return
		}
		models.SetTelegramChatID(user.ID, chatID)
	}

	http.Redirect(w, r, "/settings?message=Einstellungen+gespeichert", http.StatusSeeOther)
}
