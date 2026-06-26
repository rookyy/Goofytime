package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stundenerfassung/middleware"
	"stundenerfassung/models"
)

func ExportCSV(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	entries, err := models.GetEntriesByUserID(user.ID, "", 1, 100000)
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=Stundenerfassung_%s.csv", time.Now().Format("2006-01-02")))
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
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 5 {
			continue
		}

		date := strings.TrimSpace(row[0])
		timeFrom := strings.TrimSpace(row[1])
		timeTo := strings.TrimSpace(row[2])
		hoursStr := strings.TrimSpace(row[3])
		purpose := strings.TrimSpace(row[4])

		if date == "" || timeFrom == "" || timeTo == "" || purpose == "" {
			continue
		}

		hours, _ := strconv.ParseFloat(strings.Replace(hoursStr, ",", ".", 1), 64)
		if hours == 0 {
			hours = calculateHours(timeFrom, timeTo)
		}

		var clientID *int
		if len(row) > 5 && strings.TrimSpace(row[5]) != "" {
			clientName := strings.TrimSpace(row[5])
			clients, _ := models.GetClientsByUserID(user.ID)
			for _, c := range clients {
				if strings.EqualFold(c.Name, clientName) {
					cid := c.ID
					clientID = &cid
					break
				}
			}
		}

		_, err := models.CreateEntry(user.ID, clientID, date, timeFrom, timeTo, hours, purpose, "")
		if err == nil {
			count++
		}
	}

	models.LogActivity(user.ID, "CSV-Import", fmt.Sprintf("%d Einträge importiert", count))
	http.Redirect(w, r, "/settings/import?message=Import+abgeschlossen&count="+strconv.Itoa(count), http.StatusSeeOther)
}
