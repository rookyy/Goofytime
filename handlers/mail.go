package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"goofytime/middleware"
	"goofytime/models"
)

type MailPreview struct {
	ClientID     int
	ClientName   string
	Recipients   string
	Subject      string
	Body         string
	EntryCount   int
	TotalHours   float64
	PDFFilename  string
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

var germanMonths = []string{"", "Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}

func formatEmailSubject(template string, t time.Time, lastName string) string {
	s := strings.ReplaceAll(template, "%M", germanMonths[t.Month()])
	s = strings.ReplaceAll(s, "%J", strconv.Itoa(t.Year()))
	s = strings.ReplaceAll(s, "%N", lastName)
	return s
}

func getSMTPConfigForUser(userID int) SMTPConfig {
	settings, err := models.GetMailSettings(userID)
	if err != nil || settings.SMTPHost == "" {
		return SMTPConfig{}
	}
	return SMTPConfig{
		Host:     settings.SMTPHost,
		Port:     fmt.Sprintf("%d", settings.SMTPPort),
		User:     settings.SMTPUser,
		Password: settings.SMTPPass,
		From:     settings.SMTPFrom,
	}
}

func SendMailForm(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	hours, _ := models.GetUnbilledHoursForUser(user.ID)
	entries, _ := models.GetUnbilledEntriesForUser(user.ID)
	cfg, _ := models.GetMailSettings(user.ID)

	var previews []MailPreview
	if len(entries) > 0 && cfg != nil && cfg.SMTPHost != "" {
		previews = buildMailPreviews(user, entries, cfg)
	}

	RenderTemplate(w, "send_mail.html", map[string]interface{}{
		"Title":         "E-Mail versenden",
		"User":          user,
		"UnbilledHours": hours,
		"EntryCount":    len(entries),
		"Settings":      cfg,
		"Previews":      previews,
	})
}

func buildMailPreviews(user *models.User, entries []models.TimeEntry, mailSettings *models.MailSettings) []MailPreview {
	profile, _ := models.GetUserSettings(user.ID)
	allClients, _ := models.GetClientsByUserID(user.ID)
	now := time.Now()

	clientByID := map[int]models.Client{}
	for _, c := range allClients {
		clientByID[c.ID] = c
	}

	entriesByClient := map[int][]models.TimeEntry{}
	for _, e := range entries {
		if e.ClientID != nil {
			entriesByClient[*e.ClientID] = append(entriesByClient[*e.ClientID], e)
		}
	}

	sig := buildSignature(profile)

	var previews []MailPreview
	for clientID, clientEntries := range entriesByClient {
		c, ok := clientByID[clientID]
		if !ok || c.Recipients == "" {
			continue
		}

		clientTotal := 0.0
		for _, e := range clientEntries {
			clientTotal += e.Hours
		}

		body := c.MailText
		if body == "" {
			body = mailSettings.DefaultMailText
		}
		if body == "" {
			body = "Anbei die geleisteten Arbeitsstunden."
		}

		subj := ""
		if c.MailSubject != "" {
			subj = formatEmailSubject(c.MailSubject, now, profile.LastName)
		} else if mailSettings.EmailSubject != "" {
			subj = formatEmailSubject(mailSettings.EmailSubject, now, profile.LastName)
		}
		if subj == "" {
			subj = fmt.Sprintf("Arbeitsstunden %s - %.1fh", c.Name, clientTotal)
		}

		bodyWithDetails := body + fmt.Sprintf("\n\nGesamt: %.1f Stunden (%d Einträge)\n", clientTotal, len(clientEntries))
		for _, e := range clientEntries {
			bodyWithDetails += fmt.Sprintf("- %s: %s-%s (%.1fh) %s", e.Date, e.TimeFrom, e.TimeTo, e.Hours, e.Purpose)
			if e.ClientName != "" {
				bodyWithDetails += fmt.Sprintf(" [%s]", e.ClientName)
			}
			bodyWithDetails += "\n"
		}
		bodyWithDetails += sig

		_, filename, _ := generatePDF(clientEntries, profile, &c, subj)

		previews = append(previews, MailPreview{
			ClientID:    clientID,
			ClientName:  c.Name,
			Recipients:  c.Recipients,
			Subject:     subj,
			Body:        bodyWithDetails,
			EntryCount:  len(clientEntries),
			TotalHours:  clientTotal,
			PDFFilename: filename,
		})
	}
	return previews
}

func buildSignature(profile *models.UserSettings) string {
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
	return sig
}

func SendMail(w http.ResponseWriter, r *http.Request) {
	sendMailAndMarkBilled(w, r)
}

func sendMailAndMarkBilled(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	cfg := getSMTPConfigForUser(user.ID)

	if cfg.Host == "" || cfg.User == "" {
		http.Error(w, "SMTP nicht konfiguriert. Bitte in den Mail-Einstellungen konfigurieren.", http.StatusInternalServerError)
		return
	}

	entries, err := models.GetUnbilledEntriesForUser(user.ID)
	if err != nil || len(entries) == 0 {
		http.Error(w, "Keine unbezahlten Einträge", http.StatusBadRequest)
		return
	}

	mailSettings, _ := models.GetMailSettings(user.ID)
	profile, _ := models.GetUserSettings(user.ID)
	allClients, _ := models.GetClientsByUserID(user.ID)

	clientByID := map[int]models.Client{}
	for _, c := range allClients {
		clientByID[c.ID] = c
	}

	entriesByClient := map[int][]models.TimeEntry{}
	for _, e := range entries {
		if e.ClientID != nil {
			entriesByClient[*e.ClientID] = append(entriesByClient[*e.ClientID], e)
		}
	}

	sig := buildSignature(profile)
	sent := 0
	sentHours := 0.0

	for clientID, clientEntries := range entriesByClient {
		c, ok := clientByID[clientID]
		if !ok || c.Recipients == "" {
			continue
		}

		clientTotal := 0.0
		for _, e := range clientEntries {
			clientTotal += e.Hours
		}

		body := c.MailText
		if body == "" && mailSettings != nil {
			body = mailSettings.DefaultMailText
		}
		if body == "" {
			body = "Anbei die geleisteten Arbeitsstunden."
		}

		subj := ""
		if c.MailSubject != "" {
			subj = formatEmailSubject(c.MailSubject, time.Now(), profile.LastName)
		} else if mailSettings != nil && mailSettings.EmailSubject != "" {
			subj = formatEmailSubject(mailSettings.EmailSubject, time.Now(), profile.LastName)
		}
		if subj == "" {
			subj = fmt.Sprintf("Arbeitsstunden %s - %.1fh", c.Name, clientTotal)
		}

		body += fmt.Sprintf("\n\nGesamt: %.1f Stunden (%d Einträge)\n", clientTotal, len(clientEntries))
		for _, e := range clientEntries {
			body += fmt.Sprintf("- %s: %s-%s (%.1fh) %s", e.Date, e.TimeFrom, e.TimeTo, e.Hours, e.Purpose)
			if e.ClientName != "" {
				body += fmt.Sprintf(" [%s]", e.ClientName)
			}
			body += "\n"
		}
		body += sig

		pdfData, filename, err := generatePDF(clientEntries, profile, &c, subj)
		if err != nil {
			log.Printf("PDF error for %s: %v", c.Name, err)
			continue
		}

		for _, r := range strings.Split(c.Recipients, ",") {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			if err := sendEmailWithAttachment(cfg, r, subj, body, pdfData, filename); err != nil {
				log.Printf("Email error (to %s): %v", r, err)
			}
		}

		ids := make([]int, len(clientEntries))
		for i, e := range clientEntries {
			ids[i] = e.ID
		}
		models.MarkEntriesAsBilled(ids)
		sent++
		sentHours += clientTotal
	}

	if sent == 0 {
		http.Redirect(w, r, "/send-mail?info=Keine+Empfänger+konfiguriert", http.StatusSeeOther)
	} else {
		models.LogActivity(user.ID, "Mail versendet", fmt.Sprintf("%d E-Mails (%s) versendet", sent, formatHoursStr(sentHours)))
		http.Redirect(w, r, "/dashboard?message="+fmt.Sprintf("%d+E-Mails+versendet", sent), http.StatusSeeOther)
	}
}

func formatHoursStr(hours float64) string {
	return fmt.Sprintf("%.1fh", hours)
}

func GetUnbilledEntriesForExport(userID int, yearMonth string) ([]TimeEntryExport, error) {
	entries, err := models.GetUnbilledEntriesByMonth(userID, yearMonth)
	if err != nil {
		return nil, err
	}
	exports := make([]TimeEntryExport, len(entries))
	for i, e := range entries {
		exports[i] = TimeEntryExport{
			ID: e.ID, Date: e.Date, TimeFrom: e.TimeFrom,
			TimeTo: e.TimeTo, Hours: e.Hours, Purpose: e.Purpose,
			Location: e.Location, ClientName: e.ClientName, Billed: e.Billed,
		}
	}
	return exports, nil
}

type TimeEntryExport struct {
	ID         int
	Date       string
	TimeFrom   string
	TimeTo     string
	Hours      float64
	Purpose    string
	Location   string
	ClientName string
	Billed     bool
}

func generatePDF(entries []models.TimeEntry, profile *models.UserSettings, client *models.Client, title string) ([]byte, string, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, cleanPDF(title), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Personal info left, client info right
	pdf.SetFont("Helvetica", "", 8)
	leftX := 10.0
	rightX := 160.0
	startY := pdf.GetY()

	if profile.DisplayName() != "" {
		pdf.SetXY(leftX, startY)
		pdf.CellFormat(70, 4, cleanPDF(profile.DisplayName()), "", 1, "L", false, 0, "")
	}
	if profile.Street != "" {
		pdf.SetX(leftX)
		pdf.CellFormat(70, 4, cleanPDF(profile.Street), "", 1, "L", false, 0, "")
	}
	if profile.ZipCity != "" {
		pdf.SetX(leftX)
		pdf.CellFormat(70, 4, cleanPDF(profile.ZipCity), "", 1, "L", false, 0, "")
	}
	if profile.Phone != "" {
		pdf.SetX(leftX)
		pdf.CellFormat(70, 4, "Tel: "+cleanPDF(profile.Phone), "", 1, "L", false, 0, "")
	}
	if profile.Email != "" {
		pdf.SetX(leftX)
		pdf.CellFormat(70, 4, cleanPDF(profile.Email), "", 1, "L", false, 0, "")
	}

	leftEndY := pdf.GetY()

	if client != nil {
		pdf.SetXY(rightX, startY)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(80, 5, cleanPDF(client.Name), "", 1, "R", false, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		if client.ContactName != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(80, 4, cleanPDF(client.ContactName), "", 1, "R", false, 0, "")
		}
		if client.ContactPhone != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(80, 4, "Tel: "+cleanPDF(client.ContactPhone), "", 1, "R", false, 0, "")
		}
		if client.ContactEmail != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(80, 4, cleanPDF(client.ContactEmail), "", 1, "R", false, 0, "")
		}
		if client.Address != "" {
			pdf.SetX(rightX)
			pdf.CellFormat(80, 4, cleanPDF(client.Address), "", 1, "R", false, 0, "")
		}
	}

	// Ensure Y is below both columns (right column may be shorter than left)
	if pdf.GetY() < leftEndY {
		pdf.SetY(leftEndY)
	}
	pdf.Ln(8)

	headers := []string{"Datum", "Von", "Bis", "Std", "Beschreibung"}
	widths := []float64{25, 18, 18, 14, 140}
	if client == nil {
		headers = append(headers, "Auftraggeber")
		widths = []float64{25, 18, 18, 14, 100, 40}
	}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 9)
	total := 0.0
	for _, e := range entries {
		if pdf.GetY() > 180 {
			pdf.AddPage()
			pdf.SetFont("Helvetica", "B", 9)
			pdf.SetFillColor(230, 230, 230)
			for i, h := range headers {
				pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont("Helvetica", "", 9)
		}

		date, _ := time.Parse("2006-01-02", e.Date)
		dateStr := date.Format("02.01.2006")
		pdf.CellFormat(widths[0], 6, dateStr, "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[1], 6, e.TimeFrom, "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[2], 6, e.TimeTo, "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[3], 6, fmt.Sprintf("%.1f", e.Hours), "1", 0, "C", false, 0, "")

		clipped := cleanPDF(e.Purpose)
		if client != nil {
			if len(clipped) > 80 { clipped = clipped[:77] + "..." }
		} else {
			if len(clipped) > 55 { clipped = clipped[:52] + "..." }
		}

		if client != nil {
			pdf.CellFormat(widths[4], 6, clipped, "1", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(widths[4], 6, clipped, "1", 0, "L", false, 0, "")
			clName := cleanPDF(e.ClientName)
			if clName == "" { clName = cleanPDF(e.Location) }
			pdf.CellFormat(widths[5], 6, clName, "1", 1, "L", false, 0, "")
		}
		total += e.Hours
	}

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(widths[0]+widths[1]+widths[2], 7, cleanPDF(fmt.Sprintf("Gesamt: %.1f h (%d Einträge)", total, len(entries))), "1", 0, "R", false, 0, "")
	pdf.CellFormat(widths[3], 7, fmt.Sprintf("%.1f", total), "1", 0, "C", false, 0, "")
	if client != nil {
		pdf.CellFormat(widths[4], 7, "", "1", 1, "C", false, 0, "")
	} else {
		pdf.CellFormat(widths[4]+widths[5], 7, "", "1", 1, "C", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", err
	}

	var filename string
	if client != nil {
		filename = fmt.Sprintf("Arbeitsstunden_%s_%s.pdf", cleanFilename(client.Name), time.Now().Format("2006-01"))
	} else {
		filename = fmt.Sprintf("Arbeitsstunden_%s.pdf", time.Now().Format("2006-01"))
	}
	return buf.Bytes(), filename, nil
}

func generatePDFForExport(entries []TimeEntryExport, profile *models.UserSettings, title string, client *models.Client) ([]byte, string, error) {
	modelEntries := make([]models.TimeEntry, len(entries))
	for i, e := range entries {
		modelEntries[i] = models.TimeEntry{
			Date: e.Date, TimeFrom: e.TimeFrom, TimeTo: e.TimeTo,
			Hours: e.Hours, Purpose: e.Purpose, Location: e.Location,
			ClientName: e.ClientName,
		}
	}
	return generatePDF(modelEntries, profile, client, title)
}

func sendEmailWithAttachment(cfg SMTPConfig, to, subject, body string, attachment []byte, filename string) error {
	boundary := "BOUNDARY_1234567890"
	header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", cfg.From, to, subject, boundary)

	msg := bytes.Buffer{}
	msg.WriteString(header)
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	msg.WriteString(body)
	msg.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	msg.WriteString(fmt.Sprintf("Content-Type: application/pdf\r\n"))
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename))
	msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	msg.WriteString(base64.StdEncoding.EncodeToString(attachment))
	msg.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg.Bytes())
}
