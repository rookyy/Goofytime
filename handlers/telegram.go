package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"stundenerfassung/models"
)

var (
	pendingConfirmationsMu sync.Mutex
	pendingConfirmations   = make(map[string]PendingConfirmation)
)

type PendingConfirmation struct {
	UserID    int
	ChatID    string
	YearMonth string
	Entries   []TimeEntryExport
	TotalHours float64
	CreatedAt time.Time
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
		Data string `json:"data"`
	} `json:"callback_query"`
}

type TelegramResponse struct {
	OK     bool   `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

func getTelegramToken() string {
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}

func sendTelegramMessage(chatID, text string) error {
	token := getTelegramToken()
	if token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN nicht gesetzt")
	}

	msg := TelegramMessage{ChatID: chatID, Text: text, ParseMode: "Markdown"}
	body, _ := json.Marshal(msg)

	resp, err := http.Post(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API error: %d", resp.StatusCode)
	}
	return nil
}

func sendTelegramDocument(chatID string, pdfData []byte, filename string) error {
	token := getTelegramToken()
	if token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN nicht gesetzt")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	w.WriteField("chat_id", chatID)

	fw, err := w.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write(pdfData); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %d - %s", resp.StatusCode, string(body))
	}
	return nil
}

func answerCallbackQuery(callbackID string) {
	token := getTelegramToken()
	if token == "" {
		return
	}
	http.Post(
		fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token),
		"application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{"callback_query_id":"%s"}`, callbackID))),
	)
}

func getUserIDFromChat(chatID string) (int, error) {
	return models.GetUserIDByChatID(chatID)
}

func StartTelegramBot() {
	token := getTelegramToken()
	if token == "" {
		log.Println("Telegram: TELEGRAM_BOT_TOKEN nicht gesetzt, Bot deaktiviert")
		return
	}

	log.Println("Telegram Bot gestartet")
	var lastUpdateID int

	for {
		resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", token, lastUpdateID+1))
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var result TelegramResponse
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if result.OK {
			for _, update := range result.Result {
				lastUpdateID = update.UpdateID
				if update.Message != nil {
					handleTelegramMessage(fmt.Sprintf("%d", update.Message.Chat.ID), update.Message.Text)
				}
				if update.CallbackQuery != nil {
					chatID := fmt.Sprintf("%d", update.CallbackQuery.Message.Chat.ID)
					// Answer callback to remove loading state
					go answerCallbackQuery(update.CallbackQuery.ID)
					handleTelegramMessage(chatID, update.CallbackQuery.Data)
				}
			}
		}
	}
}

func handleTelegramMessage(chatID, text string) {
	text = strings.TrimSpace(text)
	// Strip bot username suffix (e.g. /start@mybot)
	if idx := strings.Index(text, "@"); idx > 0 && strings.HasPrefix(text, "/") {
		text = text[:idx]
	}

	if strings.HasPrefix(text, "/start") {
		sendTelegramMessage(chatID, "Willkommen! Sende deine Chat-ID, um Benachrichtigungen zu aktivieren.\nDeine Chat-ID ist: `"+chatID+"`\n\nTrage diese in den Einstellungen der Stundenerfassung ein.\n\n*Befehle:*\n/status - Offene Stunden & Einträge\n/monat - Monatsübersicht\n/export - PDF des Monats senden")
		return
	}

	if strings.EqualFold(text, "/confirm") || strings.EqualFold(text, "senden") || strings.EqualFold(text, "ja") {
		if !isRegisteredChat(chatID) {
			sendTelegramMessage(chatID, "Diese Chat-ID ist nicht registriert.")
			return
		}
		confirmMailSend(chatID)
		return
	}

	if strings.EqualFold(text, "/cancel") || strings.EqualFold(text, "nein") || strings.EqualFold(text, "abbrechen") {
		if !isRegisteredChat(chatID) {
			sendTelegramMessage(chatID, "Diese Chat-ID ist nicht registriert.")
			return
		}
		cancelMailSend(chatID)
		return
	}

	if strings.HasPrefix(text, "/status") {
		if !isRegisteredChat(chatID) {
			sendTelegramMessage(chatID, "Diese Chat-ID ist nicht registriert.")
			return
		}
		handleStatus(chatID)
		return
	}

	if strings.HasPrefix(text, "/monat") {
		if !isRegisteredChat(chatID) {
			sendTelegramMessage(chatID, "Diese Chat-ID ist nicht registriert.")
			return
		}
		handleMonat(chatID)
		return
	}

	if strings.HasPrefix(text, "/export") {
		if !isRegisteredChat(chatID) {
			sendTelegramMessage(chatID, "Diese Chat-ID ist nicht registriert.")
			return
		}
		handleExport(chatID)
		return
	}
}

func isRegisteredChat(chatID string) bool {
	tokens, _ := models.GetAllTelegramTokens()
	for _, t := range tokens {
		if t.ChatID == chatID {
			return true
		}
	}
	return false
}

func NotifyMonthlyUnbilled(userID int, chatID, yearMonth string, entries []TimeEntryExport, totalHours float64) {
	text := fmt.Sprintf("*Monatsabrechnung %s*\n\n", yearMonth)

	// Group by client
	type clientGroup struct {
		name      string
		hours     float64
		count     int
		entries   []TimeEntryExport
	}
	groups := map[string]*clientGroup{}
	var noClient *clientGroup
	var orderedKeys []string

	for _, e := range entries {
		name := e.ClientName
		if name == "" {
			if noClient == nil {
				noClient = &clientGroup{name: "Kein Auftraggeber"}
			}
			noClient.hours += e.Hours
			noClient.count++
			noClient.entries = append(noClient.entries, e)
		} else {
			if groups[name] == nil {
				groups[name] = &clientGroup{name: name}
				orderedKeys = append(orderedKeys, name)
			}
			groups[name].hours += e.Hours
			groups[name].count++
			groups[name].entries = append(groups[name].entries, e)
		}
	}

	// Get all clients for recipient check
	allClients, _ := models.GetClientsByUserID(userID)
	clientMap := map[string]*models.Client{}
	for i := range allClients {
		clientMap[allClients[i].Name] = &allClients[i]
	}

	warnings := []string{}

	for _, key := range orderedKeys {
		g := groups[key]
		text += fmt.Sprintf("*%s*: %.1fh (%d Einträge)\n", g.name, g.hours, g.count)
		for _, e := range g.entries {
			text += fmt.Sprintf("  - %s: %s-%s %s\n", e.Date[8:10]+"."+e.Date[5:7], e.TimeFrom, e.TimeTo, e.Purpose)
		}
		text += "\n"

		// Check recipients
		if client, ok := clientMap[g.name]; ok {
			if client.Recipients == "" {
				warnings = append(warnings, fmt.Sprintf("⚠️ *%s* hat keine Empfänger-Mail hinterlegt", g.name))
			}
		}
	}
	if noClient != nil {
		text += fmt.Sprintf("*Kein Auftraggeber*: %.1fh (%d Einträge)\n", noClient.hours, noClient.count)
		for _, e := range noClient.entries {
			text += fmt.Sprintf("  - %s: %s-%s %s\n", e.Date[8:10]+"."+e.Date[5:7], e.TimeFrom, e.TimeTo, e.Purpose)
		}
		text += "\n"
	}

	if len(warnings) > 0 {
		text += "\n"
		for _, w := range warnings {
			text += w + "\n"
		}
	}

	text += fmt.Sprintf("*Gesamt: %.1f h (%d Einträge)*\n", totalHours, len(entries))
	text += "\nSoll die Abrechnung per E-Mail versendet werden?"

	replyMarkup := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "✅ Senden", "callback_data": "/confirm"},
				{"text": "❌ Abbrechen", "callback_data": "/cancel"},
			},
		},
	}

	token := getTelegramToken()
	if token == "" {
		return
	}

	msg := map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "Markdown",
		"reply_markup": replyMarkup,
	}
	body, _ := json.Marshal(msg)

	go func() {
		http.Post(
			fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
			"application/json",
			bytes.NewReader(body),
		)
	}()

	pendingConfirmationsMu.Lock()
	pendingConfirmations[chatID] = PendingConfirmation{
		UserID:     userID,
		ChatID:     chatID,
		YearMonth:  yearMonth,
		Entries:    entries,
		TotalHours: totalHours,
		CreatedAt:  time.Now(),
	}
	pendingConfirmationsMu.Unlock()
}

func confirmMailSend(chatID string) {
	pendingConfirmationsMu.Lock()
	pc, ok := pendingConfirmations[chatID]
	delete(pendingConfirmations, chatID)
	pendingConfirmationsMu.Unlock()

	if !ok {
		sendTelegramMessage(chatID, "Keine ausstehende Bestätigung gefunden.")
		return
	}

	cfg := getSMTPConfigForUser(pc.UserID)
	if cfg.Host == "" || cfg.User == "" {
		sendTelegramMessage(chatID, "❌ SMTP nicht konfiguriert. Bitte in den Mail-Einstellungen einrichten.")
		return
	}

	mailSettings, _ := models.GetMailSettings(pc.UserID)
	profile, _ := models.GetUserSettings(pc.UserID)
	now := time.Now()
	allClients, _ := models.GetClientsByUserID(pc.UserID)

	// Group entries by client
	clientByID := map[int]models.Client{}
	for _, c := range allClients {
		clientByID[c.ID] = c
	}

	entriesByClient := map[int][]TimeEntryExport{}
	for _, e := range pc.Entries {
		for _, c := range allClients {
			if c.Name == e.ClientName {
				entriesByClient[c.ID] = append(entriesByClient[c.ID], e)
				break
			}
		}
	}

	sent := 0
	sentHours := 0.0

	for clientID, clientEntries := range entriesByClient {
		c := clientByID[clientID]
		if c.Recipients == "" {
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
			subj = formatEmailSubject(c.MailSubject, now, profile.LastName)
	} else if mailSettings != nil && mailSettings.EmailSubject != "" {
		subj = formatEmailSubject(mailSettings.EmailSubject, now, profile.LastName)
		}
		if subj == "" {
			subj = fmt.Sprintf("Arbeitsstunden %s - %.1fh", c.Name, clientTotal)
		}

		body += fmt.Sprintf("\n\nGesamt: %.1f Stunden (%d Einträge)\n", clientTotal, len(clientEntries))
		for _, e := range clientEntries {
			body += fmt.Sprintf("- %s: %s-%s (%.1fh) %s\n", e.Date, e.TimeFrom, e.TimeTo, e.Hours, e.Purpose)
		}

		pdfData, filename, err := generatePDFForExport(clientEntries, profile, subj, &c)
		if err != nil {
			sendTelegramMessage(chatID, fmt.Sprintf("❌ Fehler bei PDF für %s: %s", c.Name, err.Error()))
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

		sent += len(clientEntries)
		sentHours += clientTotal
	}

	sendTelegramMessage(chatID, fmt.Sprintf("✅ Mails versendet! %d Einträge (%.1fh) wurden als abgerechnet markiert.", sent, sentHours))
}

func cancelMailSend(chatID string) {
	pendingConfirmationsMu.Lock()
	delete(pendingConfirmations, chatID)
	pendingConfirmationsMu.Unlock()

	sendTelegramMessage(chatID, "❌ Versand wurde abgebrochen.")
}

func handleStatus(chatID string) {
	userID, err := getUserIDFromChat(chatID)
	if err != nil {
		sendTelegramMessage(chatID, "Chat-ID nicht gefunden.")
		return
	}

	entries, err := models.GetUnbilledEntriesForUser(userID)
	if err != nil {
		sendTelegramMessage(chatID, "Fehler beim Laden der Daten.")
		return
	}

	if len(entries) == 0 {
		sendTelegramMessage(chatID, "✅ Alle Einträge sind abgerechnet. *Keine offenen Stunden.*")
		return
	}

	// Group by client
	type clientGroup struct {
		name  string
		hours float64
		count int
	}
	groups := map[string]*clientGroup{}
	var noClient *clientGroup
	totalHours := 0.0

	for _, e := range entries {
		totalHours += e.Hours
		name := e.ClientName
		if name == "" {
			if noClient == nil {
				noClient = &clientGroup{name: "Kein Auftraggeber"}
			}
			noClient.hours += e.Hours
			noClient.count++
		} else {
			if groups[name] == nil {
				groups[name] = &clientGroup{name: name}
			}
			groups[name].hours += e.Hours
			groups[name].count++
		}
	}

	text := "*Offene Stunden*\n\n"
	for _, g := range groups {
		text += fmt.Sprintf("• *%s*: %.1fh (%d)\n", g.name, g.hours, g.count)
	}
	if noClient != nil {
		text += fmt.Sprintf("• _Kein Auftraggeber_: %.1fh (%d)\n", noClient.hours, noClient.count)
	}
	text += fmt.Sprintf("\n📝 Gesamt: *%d Einträge, %.1fh*", len(entries), totalHours)

	sendTelegramMessage(chatID, text)
}

func handleMonat(chatID string) {
	userID, err := getUserIDFromChat(chatID)
	if err != nil {
		sendTelegramMessage(chatID, "Chat-ID nicht gefunden.")
		return
	}

	now := time.Now()
	monthStr := now.Format("2006-01")

	result, err := models.GetEntriesByUserID(userID, monthStr, 1, 10000)
	if err != nil {
		sendTelegramMessage(chatID, "Fehler beim Laden der Daten.")
		return
	}

	if len(result.Entries) == 0 {
		sendTelegramMessage(chatID, fmt.Sprintf("*Keine Einträge im %s %d*", germanMonths[now.Month()], now.Year()))
		return
	}

	totalHours := 0.0
	text := fmt.Sprintf("*%s %d*\n\n", germanMonths[now.Month()], now.Year())
	for _, e := range result.Entries {
		totalHours += e.Hours
		billed := ""
		if !e.Billed {
			billed = " 🔴"
		}
		text += fmt.Sprintf("- %s: %s-%s (%.1fh)%s %s\n", e.Date[8:10]+"."+e.Date[5:7], e.TimeFrom, e.TimeTo, e.Hours, billed, e.Purpose)
	}
	text += fmt.Sprintf("\n*Gesamt: %.1f h (%d Einträge)*", totalHours, len(result.Entries))

	sendTelegramMessage(chatID, text)
}

func handleExport(chatID string) {
	userID, err := getUserIDFromChat(chatID)
	if err != nil {
		sendTelegramMessage(chatID, "Chat-ID nicht gefunden.")
		return
	}

	now := time.Now()
	monthStr := now.Format("2006-01")

	entries, err := GetUnbilledEntriesForExport(userID, monthStr)
	if err != nil || len(entries) == 0 {
		sendTelegramMessage(chatID, fmt.Sprintf("Keine offenen Einträge für %s %d.", germanMonths[now.Month()], now.Year()))
		return
	}

	sendTelegramMessage(chatID, fmt.Sprintf("📄 Erstelle PDF für %s %d...", germanMonths[now.Month()], now.Year()))

	profile, _ := models.GetUserSettings(userID)
	pdfData, filename, err := generatePDFForExport(entries, profile, fmt.Sprintf("Arbeitsstunden %d %02d", now.Year(), int(now.Month())), nil)
	if err != nil {
		sendTelegramMessage(chatID, "❌ Fehler beim Erstellen des PDFs: "+err.Error())
		return
	}

	if err := sendTelegramDocument(chatID, pdfData, filename); err != nil {
		sendTelegramMessage(chatID, "❌ Fehler beim Senden des PDFs: "+err.Error())
		return
	}
}

func CheckMonthlyBilling() {
	now := time.Now()
	prevMonth := now.AddDate(0, -1, 0)
	prevMonthStr := prevMonth.Format("2006-01")
	currentDay := now.Day()

	tokens, err := models.GetAllTelegramTokens()
	if err != nil || len(tokens) == 0 {
		return
	}

	for _, t := range tokens {
		settings, _ := models.GetUserSettings(t.UserID)
		autoMailDay := 1
		if settings != nil {
			autoMailDay = settings.AutoMailDay
		}

		if currentDay < autoMailDay {
			continue
		}

		lastCheck := models.GetLastCheckMonthForUser(t.UserID)
		if lastCheck == prevMonthStr {
			continue
		}

		entries, err := GetUnbilledEntriesForExport(t.UserID, prevMonthStr)
		if err != nil || len(entries) == 0 {
			continue
		}

		// Only notify if at least one entry belongs to a client with auto_mail enabled, or has no client
		hasAutoMail := false
		for _, e := range entries {
			if e.ClientName != "" {
				clients, _ := models.GetClientsByUserID(t.UserID)
				for _, c := range clients {
					if c.Name == e.ClientName && c.AutoMailEnabled {
						hasAutoMail = true
						break
					}
				}
				if hasAutoMail { break }
			} else {
				hasAutoMail = true // entries without client are always auto-mailed
				break
			}
		}
		if !hasAutoMail {
			continue
		}

		totalHours := 0.0
		for _, e := range entries {
			totalHours += e.Hours
		}

		if totalHours == 0 {
			continue
		}

		NotifyMonthlyUnbilled(t.UserID, t.ChatID, prevMonthStr, entries, totalHours)
		models.SetLastCheckMonthForUser(t.UserID, prevMonthStr)
		time.Sleep(500 * time.Millisecond)
	}
}

func CheckAllUnbilled() {
	tokens, err := models.GetAllTelegramTokens()
	if err != nil || len(tokens) == 0 {
		return
	}

	for _, t := range tokens {
		entries, err := GetUnbilledEntriesForExport(t.UserID, "")
		if err != nil || len(entries) == 0 {
			continue
		}

		totalHours := 0.0
		for _, e := range entries {
			totalHours += e.Hours
		}

		if totalHours == 0 {
			continue
		}

		now := time.Now()
		monthLabel := fmt.Sprintf("alle offenen (%s)", now.Format("02.01.2006"))

		NotifyMonthlyUnbilled(t.UserID, t.ChatID, monthLabel, entries, totalHours)
		time.Sleep(500 * time.Millisecond)
	}
}

func StartScheduler() {
	log.Println("Scheduler gestartet (Prüfung täglich um 9:00)")
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location())
			time.Sleep(next.Sub(now))

			CheckMonthlyBilling()
		}
	}()
}

