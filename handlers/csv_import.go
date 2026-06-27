package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goofytime/middleware"
	"goofytime/models"
)

var csvColumnAliases = map[string][]string{
	"date":     {"datum", "date", "tag"},
	"from":     {"von", "from", "start", "beginn", "time_from", "zeit_von"},
	"to":       {"bis", "to", "ende", "end", "time_to", "zeit_bis"},
	"hours":    {"stunden", "hours", "std", "dauer", "h"},
	"purpose":  {"beschreibung", "purpose", "zweck", "tätigkeit", "bemerkung", "kommentar", "description", "comment"},
	"client":   {"auftraggeber", "client", "kunde", "firma", "customer", "company"},
	"location": {"ort", "location", "standort", "place", "einsatzort"},
}

func mapCSVColumns(headers []string) map[string]int {
	mapping := make(map[string]int)
	for i, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		for key, aliases := range csvColumnAliases {
			for _, alias := range aliases {
				if lower == alias {
					if _, exists := mapping[key]; !exists {
						mapping[key] = i
					}
					break
				}
			}
		}
	}
	return mapping
}

func getCSVField(row []string, mapping map[string]int, key string) string {
	if idx, ok := mapping[key]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func ExportCSV(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	entries, err := models.GetEntriesByUserID(user.ID, "", 1, 100000)
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=Goofytime_%s.csv", time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Transfer-Encoding", "binary")

	writer := csv.NewWriter(w)
	writer.Comma = ';'
	writer.Write([]string{"Datum", "Von", "Bis", "Stunden", "Beschreibung", "Auftraggeber", "Abgerechnet"})

	for _, e := range entries.Entries {
		billed := "Nein"
		if e.Billed {
			billed = "Ja"
		}
		writer.Write([]string{
			e.Date,
			e.TimeFrom,
			e.TimeTo,
			strings.Replace(fmt.Sprintf("%.2f", e.Hours), ".", ",", 1),
			e.Purpose,
			e.ClientName,
			billed,
		})
	}
	writer.Flush()
}

func ImportCSVForm(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	RenderTemplate(w, "import_csv.html", map[string]interface{}{
		"Title":   "CSV-Import",
		"User":    user,
		"Message": r.URL.Query().Get("message"),
		"Count":   r.URL.Query().Get("count"),
	})
}

func ImportCSV(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	file, header, err := r.FormFile("csv_file")
	if err != nil {
		http.Redirect(w, r, "/settings/import?message=Keine+Datei+ausgewählt", http.StatusSeeOther)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		http.Redirect(w, r, "/settings/import?message=Nur+CSV-Dateien+erlaubt", http.StatusSeeOther)
		return
	}

	reader := csv.NewReader(file)
	reader.Comma = ';'

	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 {
		http.Redirect(w, r, "/settings/import?message=Leere+oder+ungültige+CSV-Datei", http.StatusSeeOther)
		return
	}

	count := 0
	colMap := mapCSVColumns(records[0])
	hasHeader := colMap["date"] >= 0 || colMap["from"] >= 0 || colMap["purpose"] >= 0

	for i, row := range records {
		if i == 0 && hasHeader {
			continue
		}
		if i == 0 && !hasHeader {
			// No recognized header — position-based fallback
		}

		var date, timeFrom, timeTo, hoursStr, purpose, clientName, location string
		if hasHeader {
			date = getCSVField(row, colMap, "date")
			timeFrom = getCSVField(row, colMap, "from")
			timeTo = getCSVField(row, colMap, "to")
			hoursStr = getCSVField(row, colMap, "hours")
			purpose = getCSVField(row, colMap, "purpose")
			clientName = getCSVField(row, colMap, "client")
			location = getCSVField(row, colMap, "location")
		} else {
			if len(row) < 5 {
				continue
			}
			date = strings.TrimSpace(row[0])
			timeFrom = strings.TrimSpace(row[1])
			timeTo = strings.TrimSpace(row[2])
			hoursStr = strings.TrimSpace(row[3])
			purpose = strings.TrimSpace(row[4])
			if len(row) > 5 {
				clientName = strings.TrimSpace(row[5])
			}
			if len(row) > 6 {
				location = strings.TrimSpace(row[6])
			}
		}

		if date == "" || purpose == "" {
			continue
		}
		if timeFrom == "" && timeTo == "" && hoursStr == "" {
			continue
		}

		hours, _ := strconv.ParseFloat(strings.Replace(hoursStr, ",", ".", 1), 64)
		if hours == 0 && timeFrom != "" && timeTo != "" {
			hours = calculateHours(timeFrom, timeTo)
		}
		if hours == 0 && timeFrom == "" && timeTo == "" && hoursStr == "" {
			continue
		}

		if timeFrom == "" {
			timeFrom = "08:00"
		}
		if timeTo == "" {
			toH := int(hours)
			toM := int((hours - float64(toH)) * 60)
			timeTo = fmt.Sprintf("%02d:%02d", 8+toH, toM)
		}

		var clientID *int
		if clientName != "" {
			clients, _ := models.GetClientsByUserID(user.ID)
			for _, c := range clients {
				if strings.EqualFold(c.Name, clientName) {
					cid := c.ID
					clientID = &cid
					break
				}
			}
			// Auto-create client if not found
			if clientID == nil {
				created, err := models.CreateClient(user.ID, clientName, "", "", "", "", 0, "", "", "", false, "")
				if err == nil {
					clientID = &created.ID
				}
			}
		}

		_, err := models.CreateEntry(user.ID, clientID, date, timeFrom, timeTo, hours, purpose, location)
		if err == nil {
			count++
		}
	}

	models.LogActivity(user.ID, "CSV-Import", fmt.Sprintf("%d Einträge importiert", count))
	http.Redirect(w, r, "/settings/import?message=Import+abgeschlossen&count="+strconv.Itoa(count), http.StatusSeeOther)
}
