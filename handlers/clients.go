package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goofytime/middleware"
	"goofytime/models"
)

func ClientsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	clients, _ := models.GetClientsByUserID(user.ID)

	RenderTemplate(w, "clients.html", map[string]interface{}{
		"Title":   "Auftraggeber",
		"User":    user,
		"Clients": clients,
		"Error":   r.URL.Query().Get("error"),
	})
}

func CreateClient(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	name := validateLength(r.FormValue("name"), 200)
	email := validateLength(r.FormValue("email"), 200)
	address := validateLength(r.FormValue("address"), 500)
	recipients := validateLength(r.FormValue("recipients"), 500)
	mailText := validateLength(r.FormValue("mail_text"), 2000)
	hourlyRateStr := r.FormValue("hourly_rate")
	contactName := validateLength(r.FormValue("contact_name"), 200)
	contactPhone := validateLength(r.FormValue("contact_phone"), 200)
	contactEmail := validateLength(r.FormValue("contact_email"), 200)
	autoMailEnabled := r.FormValue("auto_mail_enabled") == "on"
	mailSubject := r.FormValue("mail_subject")

	if name == "" {
		http.Redirect(w, r, "/clients?error=Name+ist+erforderlich", http.StatusSeeOther)
		return
	}

	hourlyRate := 0.0
	if hourlyRateStr != "" {
		if r, err := strconv.ParseFloat(strings.Replace(hourlyRateStr, ",", ".", 1), 64); err == nil {
			hourlyRate = r
		}
	}

	_, err := models.CreateClient(user.ID, name, email, address, recipients, mailText, hourlyRate, contactName, contactPhone, contactEmail, autoMailEnabled, mailSubject)
	if err != nil {
		http.Redirect(w, r, "/clients?error=Fehler+beim+Erstellen", http.StatusSeeOther)
		return
	}

	models.LogActivity(user.ID, "Auftraggeber erstellt", name)

	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

func EditClientForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	client, err := models.GetClientByID(id)
	if err != nil || client.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	RenderTemplate(w, "client_form_inner", map[string]interface{}{
		"Client": client,
		"Errors": map[string]string{},
	})
}

func UpdateClient(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	client, err := models.GetClientByID(id)
	if err != nil || client.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	name := validateLength(r.FormValue("name"), 200)
	email := validateLength(r.FormValue("email"), 200)
	address := validateLength(r.FormValue("address"), 500)
	recipients := validateLength(r.FormValue("recipients"), 500)
	mailText := validateLength(r.FormValue("mail_text"), 2000)
	hourlyRateStr := r.FormValue("hourly_rate")
	contactName := validateLength(r.FormValue("contact_name"), 200)
	contactPhone := validateLength(r.FormValue("contact_phone"), 200)
	contactEmail := validateLength(r.FormValue("contact_email"), 200)
	autoMailEnabled := r.FormValue("auto_mail_enabled") == "on"
	mailSubject := r.FormValue("mail_subject")

	if name == "" {
		RenderTemplate(w, "client_form_inner", map[string]interface{}{
			"Client": client,
			"Errors": map[string]string{"name": "Name ist erforderlich"},
		})
		return
	}

	if email != "" && !validateEmail(email) {
		RenderTemplate(w, "client_form_inner", map[string]interface{}{
			"Client": client,
			"Errors": map[string]string{"email": "Ungültiges E-Mail-Format"},
		})
		return
	}

	if !validateEmails(recipients) {
		RenderTemplate(w, "client_form_inner", map[string]interface{}{
			"Client": client,
			"Errors": map[string]string{"recipients": "Ungültiges E-Mail-Format bei den Empfängern"},
		})
		return
	}

	hourlyRate, _ := strconv.ParseFloat(strings.Replace(hourlyRateStr, ",", ".", 1), 64)

	err = models.UpdateClient(id, name, email, address, recipients, mailText, hourlyRate, contactName, contactPhone, contactEmail, autoMailEnabled, mailSubject)
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}

	models.LogActivity(user.ID, "Auftraggeber bearbeitet", name)

	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

func DeleteClient(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	client, err := models.GetClientByID(id)
	if err != nil || client.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	models.DeleteClient(id)
	models.LogActivity(user.ID, "Auftraggeber gelöscht", client.Name)
	w.WriteHeader(http.StatusOK)
}

func PreviewClientMail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	client, err := models.GetClientByID(id)
	if err != nil || client.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	unbilledHours := 0.0
	var unbilledCount int
	entries, _ := models.GetUnbilledEntriesForUser(user.ID)
	for _, e := range entries {
		if e.ClientID != nil && *e.ClientID == id {
			unbilledHours += e.Hours
			unbilledCount++
		}
	}

	recipients := ""
	if client.Recipients != "" {
		recipients = client.Recipients
	}

	mailSettings, _ := models.GetMailSettings(user.ID)
	mailText := client.MailText
	if mailText == "" && mailSettings != nil {
		mailText = mailSettings.DefaultMailText
	}
	if mailText == "" {
		mailText = "Anbei die geleisteten Arbeitsstunden."
	}

	RenderTemplate(w, "client_mail_preview", map[string]interface{}{
		"Client":        client,
		"UnbilledHours": unbilledHours,
		"UnbilledCount": unbilledCount,
		"Recipients":    recipients,
		"MailText":      mailText,
	})
}

func SendClientMail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	client, err := models.GetClientByID(id)
	if err != nil || client.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	markBilled := r.FormValue("mark_billed") == "1"

	cfg := getSMTPConfigForUser(user.ID)
	if cfg.Host == "" || cfg.User == "" {
		http.Error(w, "SMTP nicht konfiguriert.", http.StatusInternalServerError)
		return
	}

	if client.Recipients == "" {
		http.Error(w, "Keine Empfänger hinterlegt.", http.StatusBadRequest)
		return
	}

	recipients := []string{}
	for _, r := range strings.Split(client.Recipients, ",") {
		r = strings.TrimSpace(r)
		if r != "" { recipients = append(recipients, r) }
	}

	entries, _ := models.GetUnbilledEntriesForUser(user.ID)
	var clientEntries []models.TimeEntry
	totalHours := 0.0
	for _, e := range entries {
		if e.ClientID != nil && *e.ClientID == id {
			clientEntries = append(clientEntries, e)
			totalHours += e.Hours
		}
	}
	if len(clientEntries) == 0 {
		http.Error(w, "Keine unbezahlten Einträge.", http.StatusBadRequest)
		return
	}

	mailSettings, _ := models.GetMailSettings(user.ID)
	mailText := client.MailText
	if mailText == "" && mailSettings != nil { mailText = mailSettings.DefaultMailText }
	if mailText == "" { mailText = "Anbei die geleisteten Arbeitsstunden." }

	profile, _ := models.GetUserSettings(user.ID)

	now := time.Now()
	subject := ""
	if client.MailSubject != "" {
		subject = formatEmailSubject(client.MailSubject, now, profile.LastName)
	} else if mailSettings != nil && mailSettings.EmailSubject != "" {
		subject = formatEmailSubject(mailSettings.EmailSubject, now, profile.LastName)
	}
	if subject == "" {
		subject = fmt.Sprintf("Arbeitsstunden %s - %.1fh", client.Name, totalHours)
	}

	pdfData, filename, _ := generatePDF(clientEntries, profile, client, subject)

	body := mailText + fmt.Sprintf("\n\nGesamt: %.1f Stunden (%d Einträge)\n", totalHours, len(clientEntries))
	for _, e := range clientEntries {
		body += fmt.Sprintf("- %s: %s-%s (%.1fh) %s\n", e.Date, e.TimeFrom, e.TimeTo, e.Hours, e.Purpose)
	}

	for _, recipient := range recipients {
		sendEmailWithAttachment(cfg, recipient, subject, body, pdfData, filename)
	}

	if markBilled {
		ids := make([]int, len(clientEntries))
		for i, e := range clientEntries { ids[i] = e.ID }
		models.MarkEntriesAsBilled(ids)
	}

	models.LogActivity(user.ID, "Mail versendet", fmt.Sprintf("%s: %d Einträge (%.1fh)", client.Name, len(clientEntries), totalHours))
	http.Redirect(w, r, "/clients?sent=1", http.StatusSeeOther)
}
