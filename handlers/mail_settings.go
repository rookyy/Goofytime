package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"

	"goofytime/middleware"
	"goofytime/models"
)

func MailSettingsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	settings, _ := models.GetMailSettings(user.ID)
	message := r.URL.Query().Get("message")

	unbilledHours, _ := models.GetUnbilledHoursForUser(user.ID)
	entries, _ := models.GetUnbilledEntriesForUser(user.ID)
	userSettings, _ := models.GetUserSettings(user.ID)

	RenderTemplate(w, "mail_settings.html", map[string]interface{}{
		"Title":         "Mail",
		"User":          user,
		"Settings":      settings,
		"Message":       message,
		"UnbilledHours": unbilledHours,
		"EntryCount":    len(entries),
		"AutoMailDay":   userSettings.AutoMailDay,
	})
}

func SaveMailSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	host := r.FormValue("smtp_host")
	port, _ := strconv.Atoi(r.FormValue("smtp_port"))
	if port == 0 {
		port = 587
	}
	smtpUser := r.FormValue("smtp_user")
	smtpPass := r.FormValue("smtp_pass")
	from := r.FormValue("smtp_from")
	defaultMailText := r.FormValue("default_mail_text")
	emailSubject := r.FormValue("email_subject")
	autoMailDay, _ := strconv.Atoi(r.FormValue("auto_mail_day"))

	models.SaveMailSettings(user.ID, host, port, smtpUser, smtpPass, from, defaultMailText, emailSubject)
	models.SaveUserSettings(user.ID, autoMailDay, 30)
	http.Redirect(w, r, "/mail-settings?message=Einstellungen+gespeichert", http.StatusSeeOther)
}

func SendTestMail(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	settings, _ := models.GetMailSettings(user.ID)
	profile, _ := models.GetUserSettings(user.ID)

	if settings.SMTPHost == "" || settings.SMTPUser == "" {
		http.Redirect(w, r, "/mail-settings?message=SMTP+nicht+vollständig+konfiguriert", http.StatusSeeOther)
		return
	}

	recipient := r.FormValue("test_recipient")
	if recipient == "" {
		http.Redirect(w, r, "/mail-settings?message=Kein+Empfänger+angegeben", http.StatusSeeOther)
		return
	}

	sig := ""
	if profile.DisplayName() != "" {
		sig += fmt.Sprintf("\n\n-- \n%s", profile.DisplayName())
		if profile.Street != "" {
			sig += fmt.Sprintf("\n%s", profile.Street)
		}
		if profile.ZipCity != "" {
			sig += fmt.Sprintf("\n%s", profile.ZipCity)
		}
		if profile.Phone != "" {
			sig += fmt.Sprintf("\nTel: %s", profile.Phone)
		}
		if profile.Email != "" {
			sig += fmt.Sprintf("\n%s", profile.Email)
		}
	}

	body := fmt.Sprintf("Dies ist eine Test-Mail der Goofytime.%s", sig)
	cfg := SMTPConfig{
		Host:     settings.SMTPHost,
		Port:     fmt.Sprintf("%d", settings.SMTPPort),
		User:     settings.SMTPUser,
		Password: settings.SMTPPass,
		From:     settings.SMTPFrom,
	}

	err := sendSimpleEmail(cfg, recipient, "Goofytime - Testmail", body)
	if err != nil {
		log.Printf("Testmail error: %v", err)
		http.Redirect(w, r, "/mail-settings?message=Fehler+beim+Senden", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/mail-settings?message=Testmail+erfolgreich+gesendet", http.StatusSeeOther)
}

func sendSimpleEmail(cfg SMTPConfig, to, subject, body string) error {
	header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n", cfg.From, to, subject)
	msg := header + body

	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(msg))
}

func TriggerAutoMail(w http.ResponseWriter, r *http.Request) {
	go CheckAllUnbilled()
	http.Redirect(w, r, "/mail-settings?message=Auto-Mail+Prüfung+gestartet", http.StatusSeeOther)
}
