package handlers

import (
	"errors"
	"net/http"

	"goofytime/middleware"
	"goofytime/models"
	"golang.org/x/crypto/bcrypt"
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

func ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	oldPassword := r.FormValue("old_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if oldPassword == "" || newPassword == "" {
		http.Redirect(w, r, "/profile?message=Alle+Felder+sind+erforderlich", http.StatusSeeOther)
		return
	}
	if newPassword != confirmPassword {
		http.Redirect(w, r, "/profile?message=Passw%C3%B6rter+stimmen+nicht+%C3%BCberein", http.StatusSeeOther)
		return
	}
	if len(newPassword) < 4 {
		http.Redirect(w, r, "/profile?message=Passwort+muss+mindestens+4+Zeichen+lang+sein", http.StatusSeeOther)
		return
	}

	err := models.ChangePassword(user.ID, oldPassword, newPassword)
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) || err != nil {
		http.Redirect(w, r, "/profile?message=Altes+Passwort+ist+falsch", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/profile?message=Passwort+erfolgreich+ge%C3%A4ndert", http.StatusSeeOther)
}
