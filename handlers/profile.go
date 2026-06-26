package handlers

import (
	"net/http"

	"goofytime/middleware"
	"goofytime/models"
)

func ProfilePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	settings, _ := models.GetUserSettings(user.ID)
	message := r.URL.Query().Get("message")

	RenderTemplate(w, "profile.html", map[string]interface{}{
		"Title":    "Profil",
		"User":     user,
		"Settings": settings,
		"Message":  message,
	})
}

func SaveProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	street := r.FormValue("street")
	zipCity := r.FormValue("zip_city")
	phone := r.FormValue("phone")
	email := r.FormValue("email")

	models.SaveProfile(user.ID, firstName, lastName, street, zipCity, phone, email)

	http.Redirect(w, r, "/profile?message=Profil+gespeichert", http.StatusSeeOther)
}
