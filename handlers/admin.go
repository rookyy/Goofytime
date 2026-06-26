package handlers

import (
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"goofytime/middleware"
	"goofytime/models"
)

type UserWithStats struct {
	*models.User
	EntryCount int
	TotalHours float64
}

func buildUsersWithStats(users []models.User) []UserWithStats {
	var result []UserWithStats
	for _, u := range users {
		count, _ := models.GetEntryCountForUser(u.ID)
		hours, _ := models.GetTotalHoursForUser(u.ID)
		result = append(result, UserWithStats{User: &u, EntryCount: count, TotalHours: hours})
	}
	return result
}

func AdminPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	users, err := models.GetAllUsers()
	if err != nil {
		http.Error(w, "Fehler beim Laden der Benutzer", http.StatusInternalServerError)
		return
	}

	usersWithStats := buildUsersWithStats(users)

	RenderTemplate(w, "admin.html", map[string]interface{}{
		"Title": "Benutzerverwaltung",
		"User":  user,
		"Users": usersWithStats,
	})
}

func NewUserForm(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "user_form.html", map[string]interface{}{
		"Title":  "Neuer Benutzer",
		"Edit":   false,
		"Target": nil,
		"Errors": map[string]string{},
	})
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	isAdmin := r.FormValue("is_admin") == "on"

	errors := map[string]string{}

	if username == "" {
		errors["username"] = "Benutzername ist erforderlich"
	} else {
		exists, _ := models.UserExists(username)
		if exists {
			errors["username"] = "Benutzername existiert bereits"
		}
	}

	if password == "" {
		errors["password"] = "Passwort ist erforderlich"
	}

	if len(errors) > 0 {
		RenderTemplate(w, "user_form.html", map[string]interface{}{
			"Title": "Neuer Benutzer",
			"Edit":  false,
			"Target": map[string]interface{}{
				"username": username,
				"is_admin": isAdmin,
			},
			"Errors": errors,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server-Fehler", http.StatusInternalServerError)
		return
	}

	_, err = models.CreateUser(username, string(hash), isAdmin)
	if err != nil {
		http.Error(w, "Fehler beim Erstellen", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func EditUserForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))

	target, err := models.GetUserByID(id)
	if err != nil {
		http.Error(w, "Benutzer nicht gefunden", http.StatusNotFound)
		return
	}

	if r.URL.Query().Get("partial") == "1" {
		RenderTemplate(w, "user_form_inner", map[string]interface{}{
			"Title":  "Benutzer bearbeiten",
			"Edit":   true,
			"Target": target,
			"Errors": map[string]string{},
		})
		return
	}

	RenderTemplate(w, "user_form.html", map[string]interface{}{
		"Title":  "Benutzer bearbeiten",
		"Edit":   true,
		"Target": target,
		"Errors": map[string]string{},
	})
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))

	target, err := models.GetUserByID(id)
	if err != nil {
		http.Error(w, "Benutzer nicht gefunden", http.StatusNotFound)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	isAdmin := r.FormValue("is_admin") == "on"

	errors := map[string]string{}

	if username == "" {
		errors["username"] = "Benutzername ist erforderlich"
	} else if username != target.Username {
		exists, _ := models.UserExists(username)
		if exists {
			errors["username"] = "Benutzername existiert bereits"
		}
	}

	if len(errors) > 0 {
		RenderTemplate(w, "user_form.html", map[string]interface{}{
			"Title":  "Benutzer bearbeiten",
			"Edit":   true,
			"Target": target,
			"Errors": errors,
		})
		return
	}

	var passwordHash string
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Server-Fehler", http.StatusInternalServerError)
			return
		}
		passwordHash = string(hash)
	}

	err = models.UpdateUser(id, username, passwordHash, isAdmin)
	if err != nil {
		http.Error(w, "Fehler beim Aktualisieren", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	currentUser := middleware.GetUser(r)

	if id == currentUser.ID {
		w.Header().Set("HX-Trigger", `{"showToast": "Du kannst dich nicht selbst löschen"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	models.DeleteUser(id)
	w.WriteHeader(http.StatusOK)
}
