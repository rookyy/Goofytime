package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"goofytime/middleware"
	"goofytime/models"
)

var (
	loginAttempts   = make(map[string]int)
	loginBlocked    = make(map[string]time.Time)
	loginMu         sync.Mutex
)

func LoginPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	RenderTemplate(w, "login.html", map[string]interface{}{
		"Title": "Anmeldung",
		"Error": r.URL.Query().Get("error"),
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if idx := len(ip) - 1; idx > 0 {
		for i := len(ip) - 1; i >= 0; i-- {
			if ip[i] == ':' {
				ip = ip[:i]
				break
			}
		}
	}
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimPrefix(ip, "]")

	loginMu.Lock()
	if blockedUntil, ok := loginBlocked[ip]; ok && time.Now().Before(blockedUntil) {
		loginMu.Unlock()
		http.Redirect(w, r, "/login?error=Zu+viele+Versuche.+Bitte+warten.", http.StatusSeeOther)
		return
	}

	attempts := loginAttempts[ip] + 1
	loginAttempts[ip] = attempts

	if attempts > 5 {
		loginBlocked[ip] = time.Now().Add(60 * time.Second)
		delete(loginAttempts, ip)
		loginMu.Unlock()
		http.Redirect(w, r, "/login?error=Zu+viele+Versuche.+60+Sekunden+sperre.", http.StatusSeeOther)
		return
	}
	loginMu.Unlock()

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := models.GetUserByUsername(username)
	if err != nil {
		time.Sleep(time.Duration(attempts) * 500 * time.Millisecond)
		http.Redirect(w, r, "/login?error=Benutzername+oder+Passwort+falsch", http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		time.Sleep(time.Duration(attempts) * 500 * time.Millisecond)
		http.Redirect(w, r, "/login?error=Benutzername+oder+Passwort+falsch", http.StatusSeeOther)
		return
	}

	loginMu.Lock()
	delete(loginAttempts, ip)
	delete(loginBlocked, ip)
	loginMu.Unlock()

	session, err := models.CreateSession(user.ID)
	if err != nil {
		http.Redirect(w, r, "/login?error=Server-Fehler", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    session.Token,
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		models.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
