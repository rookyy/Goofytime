package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"goofytime/database"
	"goofytime/models"
)

func OnboardingPage(w http.ResponseWriter, r *http.Request) {
	step, _ := strconv.Atoi(r.URL.Query().Get("step"))
	if step < 1 { step = 1 }
	msg := r.URL.Query().Get("message")

	RenderTemplate(w, "onboarding.html", map[string]interface{}{
		"Title":   "Einrichtung",
		"Step":    step,
		"Message": msg,
	})
}

func OnboardingStep(w http.ResponseWriter, r *http.Request) {
	step, _ := strconv.Atoi(r.URL.Query().Get("step"))
	if step < 1 { step = 1 }

	switch step {
	case 1:
		// DB import as alternative to admin creation
		file, header, err := r.FormFile("db_file")
		if err == nil {
			defer file.Close()
			if header.Size > 100*1024*1024 {
				http.Redirect(w, r, "/onboarding?step=1&message=Datei+zu+groß+(max.+100+MB)", http.StatusSeeOther)
				return
			}
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" { dbPath = "goofytime.db" }
		if database.DB != nil {
			if err := database.DB.Close(); err != nil {
				log.Printf("DB close error: %v", err)
			}
			database.DB = nil
		}
		dst, err := os.Create(dbPath)
		if err != nil {
			http.Redirect(w, r, "/onboarding?step=1&message=Fehler+beim+Import", http.StatusSeeOther)
			return
		}
		if _, err := io.Copy(dst, file); err != nil {
			dst.Close()
			http.Redirect(w, r, "/onboarding?step=1&message=Fehler+beim+Kopieren", http.StatusSeeOther)
			return
		}
		dst.Close()
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
		database.Init(dbPath)
		InitTemplates()
		count, _ := models.GetUserCount()
		log.Printf("DB import: %d users found after import", count)
		if count == 0 {
			http.Redirect(w, r, "/onboarding?step=1&message=Keine+Benutzer+in+der+Datei", http.StatusSeeOther)
			return
		}
			log.Println("Datenbank aus Backup wiederhergestellt")
			http.Redirect(w, r, "/login?error=Datenbank+wiederhergestellt.+Bitte+mit+den+Zugangsdaten+aus+dem+Backup+anmelden.", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || len(username) > 50 {
			http.Redirect(w, r, "/onboarding?step=1&message=Benutzername+erforderlich+(max.+50+Zeichen)", http.StatusSeeOther)
			return
		}
		if len(password) < 4 {
			http.Redirect(w, r, "/onboarding?step=1&message=Passwort+mindestens+4+Zeichen", http.StatusSeeOther)
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		models.CreateUser(username, string(hash), true)
		http.Redirect(w, r, "/onboarding?step=2", http.StatusSeeOther)

	case 2:
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		models.SaveProfile(1, firstName, lastName, "", "", "", "")
		http.Redirect(w, r, "/onboarding?step=3", http.StatusSeeOther)

	case 3:
		if r.FormValue("skip") == "1" {
			http.Redirect(w, r, "/onboarding?step=4", http.StatusSeeOther)
			return
		}
		name := r.FormValue("name")
		email := r.FormValue("email")
		if name != "" {
			models.CreateClient(1, name, email, "")
		}
		http.Redirect(w, r, "/onboarding?step=4", http.StatusSeeOther)

	case 4:
		if r.FormValue("skip") == "1" {
			http.Redirect(w, r, "/onboarding?step=5", http.StatusSeeOther)
			return
		}
		token := r.FormValue("telegram_token")
		if token != "" {
			models.SaveTelegramBotToken(token)
			models.LogActivity(1, "Onboarding", "Telegram-Token gesetzt")
		}
		http.Redirect(w, r, "/onboarding?step=5", http.StatusSeeOther)

	case 5:
		title := r.FormValue("header_title")
		footer := r.FormValue("footer_text")
		if title == "" { title = "Cold-IT Zeiterfassung" }
		if footer == "" { footer = "Made by Cold-IT" }
		models.SaveAdminSettings(title, footer, "")

		session, _ := models.CreateSession(1)
		http.SetCookie(w, &http.Cookie{Name: "session", Value: session.Token, Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
		http.SetCookie(w, &http.Cookie{Name: "csrf_token", Value: session.Token, Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: false, Secure: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}
