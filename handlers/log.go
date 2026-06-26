package handlers

import (
	"net/http"

	"stundenerfassung/middleware"
	"stundenerfassung/models"
)

func LogPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	actionFilter := r.URL.Query().Get("action")

	logs, _ := models.GetActivityLogs(user.ID, actionFilter, 200)
	actions, _ := models.GetDistinctActions(user.ID)

	RenderTemplate(w, "log.html", map[string]interface{}{
		"Title":        "Aktivitätslog",
		"User":         user,
		"Logs":         logs,
		"Actions":      actions,
		"ActionFilter": actionFilter,
	})
}
