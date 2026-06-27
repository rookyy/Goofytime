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

func Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	rangeParam := r.URL.Query().Get("range")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	clientFilter, _ := strconv.Atoi(r.URL.Query().Get("client"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	now := time.Now()
	var fromDate, toDate time.Time

	switch rangeParam {
	case "1m": fromDate = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "3m": fromDate = time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "6m": fromDate = time.Date(now.Year(), now.Month()-6, 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "1y": fromDate = time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "custom":
		if fromStr != "" { fromDate, _ = time.Parse("2006-01-02", fromStr) }
		if toStr != "" { toDate, _ = time.Parse("2006-01-02", toStr); toDate = toDate.Add(24*time.Hour - time.Second) }
	}

	settings, _ := models.GetUserSettings(user.ID)
	pageSize := settings.PageSize
	if ps, _ := strconv.Atoi(r.URL.Query().Get("size")); ps >= 5 && ps <= 200 && ps != pageSize {
		pageSize = ps
		models.SaveUserSettings(user.ID, settings.AutoMailDay, ps)
	}

	result, err := models.GetEntriesByUserID(user.ID, "", 1, 10000)
	if err != nil {
		http.Error(w, "Fehler beim Laden der Einträge", http.StatusInternalServerError)
		return
	}

	var filtered []models.TimeEntry
	for _, e := range result.Entries {
		entryDate, _ := time.Parse("2006-01-02", e.Date)
		if !fromDate.IsZero() && entryDate.Before(fromDate) { continue }
		if !toDate.IsZero() && entryDate.After(toDate) { continue }
		if clientFilter > 0 && (e.ClientID == nil || *e.ClientID != clientFilter) { continue }
		filtered = append(filtered, e)
	}

	total := len(filtered)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 { totalPages = 1 }
	if page > totalPages { page = totalPages }

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total { end = total }
	var pageEntries []models.TimeEntry
	if start < total { pageEntries = filtered[start:end] }

	totalHours, _ := models.GetTotalHoursForUser(user.ID)
	entryCount, _ := models.GetEntryCountForUser(user.ID)
	monthlyHours, _ := models.GetMonthlyHoursForUser(user.ID, time.Now().Format("2006-01"))
	unbilledHours, _ := models.GetUnbilledHoursForUser(user.ID)
	clients, _ := models.GetClientsByUserID(user.ID)
	sent := r.URL.Query().Get("sent")
	partial := r.URL.Query().Get("partial") == "1"

	data := map[string]interface{}{
		"Title":            "Dashboard",
		"User":             user,
		"Entries":          pageEntries,
		"TotalHours":       totalHours,
		"EntryCount":       entryCount,
		"MonthlyHours":     monthlyHours,
		"UnbilledHours":    unbilledHours,
		"Sent":             sent,
		"Page":             page,
		"TotalPages":       totalPages,
		"PageSize":         pageSize,
		"Total":            total,
		"Pages":            pageNumbers(page, totalPages),
		"Clients":          clients,
		"SelectedClientID": 0,
		"SelectedClient":   clientFilter,
		"Range":            rangeParam,
		"From":             fromStr,
		"To":               toStr,
		"Today":            time.Now().Format("2006-01-02"),
	}

	if partial {
		RenderTemplate(w, "dashboard_partial", data)
	} else {
		RenderTemplate(w, "dashboard.html", data)
	}
}

func NewEntryForm(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	clients, _ := models.GetClientsByUserID(user.ID)

	RenderTemplate(w, "entry_form.html", map[string]interface{}{
		"Title":            "Neuer Eintrag",
		"Entry":            nil,
		"Clients":          clients,
		"SelectedClientID": 0,
		"Today":            time.Now().Format("2006-01-02"),
		"Errors":           map[string]string{},
	})
}

func CreateEntry(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	date := r.FormValue("date")
	timeFrom := r.FormValue("time_from")
	timeTo := r.FormValue("time_to")
	purpose := validateLength(r.FormValue("purpose"), 500)
	location := ""
	hoursStr := r.FormValue("hours")
	clientIDStr := r.FormValue("client_id")

	errors := map[string]string{}

	if date == "" {
		errors["date"] = "Datum ist erforderlich"
	}
	if timeFrom == "" {
		errors["time_from"] = "Von ist erforderlich"
	}
	if timeTo == "" {
		errors["time_to"] = "Bis ist erforderlich"
	}
	if purpose == "" {
		errors["purpose"] = "Beschreibung ist erforderlich"
	}

	hours := 0.0
	if hoursStr != "" {
		h, err := strconv.ParseFloat(strings.Replace(hoursStr, ",", ".", 1), 64)
		if err != nil {
			errors["hours"] = "Ungültige Stundenzahl"
		} else {
			hours = h
		}
	}

	var clientID *int
	if clientIDStr != "" {
		id, err := strconv.Atoi(clientIDStr)
		if err == nil && id > 0 {
			clientID = &id
		}
	}

	if len(errors) > 0 {
		clients, _ := models.GetClientsByUserID(user.ID)
		sel := 0
		if clientID != nil {
			sel = *clientID
		}
		entry := map[string]interface{}{
			"date":      date,
			"time_from": timeFrom,
			"time_to":   timeTo,
			"hours":     hoursStr,
			"purpose":   purpose,
			"location":  location,
		}
		RenderTemplate(w, "entry_form.html", map[string]interface{}{
			"Title":            "Neuer Eintrag",
			"Entry":            entry,
			"Clients":          clients,
			"SelectedClientID": sel,
			"Errors":           errors,
		})
		return
	}

	if hours == 0 {
		hours = calculateHours(timeFrom, timeTo)
	}

	_, err := models.CreateEntry(user.ID, clientID, date, timeFrom, timeTo, hours, purpose, location)
	if err != nil {
		http.Error(w, "Fehler beim Speichern", http.StatusInternalServerError)
		return
	}

	models.LogActivity(user.ID, "Eintrag erstellt", date+" "+purpose)

	if r.FormValue("from_timer") == "1" {
		models.DeleteActiveTimer(user.ID)
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func EditEntryForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	entry, err := models.GetEntryByID(id)
	if err != nil || entry.UserID != user.ID {
		http.Error(w, "Eintrag nicht gefunden", http.StatusNotFound)
		return
	}

	clients, _ := models.GetClientsByUserID(user.ID)
	sel := 0
	if entry.ClientID != nil {
		sel = *entry.ClientID
	}

	if r.URL.Query().Get("partial") == "1" {
		RenderTemplate(w, "entry_form_inner", map[string]interface{}{
			"Title":            "Eintrag bearbeiten",
			"Entry":            entry,
			"Clients":          clients,
			"SelectedClientID": sel,
			"Errors":           map[string]string{},
		})
		return
	}

	RenderTemplate(w, "entry_form.html", map[string]interface{}{
		"Title":            "Eintrag bearbeiten",
		"Entry":            entry,
		"Clients":          clients,
		"SelectedClientID": sel,
		"Errors":           map[string]string{},
	})
}

func UpdateEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	entry, err := models.GetEntryByID(id)
	if err != nil || entry.UserID != user.ID {
		http.Error(w, "Eintrag nicht gefunden", http.StatusNotFound)
		return
	}

	date := r.FormValue("date")
	timeFrom := r.FormValue("time_from")
	timeTo := r.FormValue("time_to")
	purpose := validateLength(r.FormValue("purpose"), 500)
	location := ""
	hoursStr := r.FormValue("hours")
	clientIDStr := r.FormValue("client_id")

	errors := map[string]string{}

	if date == "" {
		errors["date"] = "Datum ist erforderlich"
	}
	if timeFrom == "" {
		errors["time_from"] = "Von ist erforderlich"
	}
	if timeTo == "" {
		errors["time_to"] = "Bis ist erforderlich"
	}

	hours := 0.0
	if hoursStr != "" {
		h, err := strconv.ParseFloat(strings.Replace(hoursStr, ",", ".", 1), 64)
		if err != nil {
			errors["hours"] = "Ungültige Stundenzahl"
		} else {
			hours = h
		}
	} else {
		hours = entry.Hours
	}

	var clientID *int
	if clientIDStr != "" {
		cid, err := strconv.Atoi(clientIDStr)
		if err == nil && cid > 0 {
			clientID = &cid
		}
	} else if clientIDStr == "" {
		clientID = nil
	}

	if len(errors) > 0 {
		clients, _ := models.GetClientsByUserID(user.ID)
		sel := 0
		if clientID != nil {
			sel = *clientID
		}
		RenderTemplate(w, "entry_form.html", map[string]interface{}{
			"Title":            "Eintrag bearbeiten",
			"Entry":            entry,
			"Clients":          clients,
			"SelectedClientID": sel,
			"Errors":           errors,
		})
		return
	}

	if hoursStr == "" {
		hours = calculateHours(timeFrom, timeTo)
	}

	err = models.UpdateEntry(id, clientID, date, timeFrom, timeTo, hours, purpose, location)
	if err != nil {
		http.Error(w, "Fehler beim Aktualisieren", http.StatusInternalServerError)
		return
	}

	models.LogActivity(user.ID, "Eintrag bearbeitet", date+" "+purpose)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func DeleteEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	entry, err := models.GetEntryByID(id)
	if err != nil || entry.UserID != user.ID {
		http.Error(w, "Eintrag nicht gefunden", http.StatusNotFound)
		return
	}

	models.DeleteEntry(id)
	models.LogActivity(user.ID, "Eintrag gelöscht", entry.Date+" "+entry.Purpose)

	if r.Header.Get("HX-Request") != "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func calculateHours(timeFrom, timeTo string) float64 {
	parseMinutes := func(t string) int {
		parts := strings.Split(t, ":")
		if len(parts) != 2 {
			return 0
		}
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		return h*60 + m
	}

	from := parseMinutes(timeFrom)
	to := parseMinutes(timeTo)

	diff := to - from
	if diff < 0 {
		diff += 24 * 60
	}

	return float64(diff) / 60.0
}

func HTMLEntryRow(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	entry, err := models.GetEntryByID(id)
	if err != nil || entry.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	RenderTemplate(w, "entry_row", map[string]interface{}{
		"Entry": entry,
	})
}

func ToggleBilled(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	user := middleware.GetUser(r)

	entry, err := models.GetEntryByID(id)
	if err != nil || entry.UserID != user.ID {
		http.Error(w, "Nicht gefunden", http.StatusNotFound)
		return
	}

	billed, err := models.ToggleBilled(id)
	if err != nil {
		http.Error(w, "Fehler", http.StatusInternalServerError)
		return
	}
	entry.Billed = billed

	if billed {
		models.LogActivity(user.ID, "Eintrag abgerechnet", entry.Date+" "+entry.Purpose)
	} else {
		models.LogActivity(user.ID, "Eintrag offen", entry.Date+" "+entry.Purpose)
	}

	RenderTemplate(w, "entry_row", map[string]interface{}{
		"Entry": entry,
	})
}

func pageNumbers(current, total int) []int {
	pages := []int{}
	start := current - 2
	if start < 1 {
		start = 1
	}
	end := current + 2
	if end > total {
		end = total
	}
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}
	return pages
}

func BulkToggleBilled(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	ids := parseIDs(r.FormValue("ids"))
	if len(ids) == 0 {
		http.Error(w, "Keine Einträge ausgewählt", http.StatusBadRequest)
		return
	}
	for _, id := range ids {
		entry, err := models.GetEntryByID(id)
		if err != nil || entry.UserID != user.ID {
			continue
		}
		models.ToggleBilled(id)
	}
	models.LogActivity(user.ID, "Bulk-Aktion", fmt.Sprintf("%d Einträge abgerechnet/offen", len(ids)))
	w.WriteHeader(http.StatusOK)
}

func BulkDeleteEntries(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	ids := parseIDs(r.FormValue("ids"))
	if len(ids) == 0 {
		http.Error(w, "Keine Einträge ausgewählt", http.StatusBadRequest)
		return
	}
	count := 0
	for _, id := range ids {
		entry, err := models.GetEntryByID(id)
		if err != nil || entry.UserID != user.ID {
			continue
		}
		models.DeleteEntry(id)
		count++
	}
	models.LogActivity(user.ID, "Bulk-Löschung", fmt.Sprintf("%d Einträge gelöscht", count))
	w.WriteHeader(http.StatusOK)
}

func parseIDs(raw string) []int {
	var ids []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if id, err := strconv.Atoi(part); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}
