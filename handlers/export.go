package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"golang.org/x/text/encoding/charmap"

	"goofytime/middleware"
	"goofytime/models"
)

var fallbackReplacer = strings.NewReplacer(
	"€", "EUR", "–", "-", "—", "-",
)

func cleanPDF(s string) string {
	encoded, err := charmap.ISO8859_1.NewEncoder().String(s)
	if err != nil {
		return s
	}
	return fallbackReplacer.Replace(encoded)
}

func cleanFilename(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, s)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	return s
}

func ExportPDF(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	rangeParam := r.URL.Query().Get("range")
	status := r.URL.Query().Get("status")
	clientFilter := r.URL.Query().Get("client")
	if status == "" {
		status = "unbilled"
	}

	now := time.Now()
	var fromDate, toDate time.Time

	switch rangeParam {
	case "1m":
		fromDate = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		toDate = now
	case "3m":
		fromDate = time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, now.Location())
		toDate = now
	case "6m":
		fromDate = time.Date(now.Year(), now.Month()-6, 1, 0, 0, 0, 0, now.Location())
		toDate = now
	case "1y":
		fromDate = time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location())
		toDate = now
	case "custom":
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		if fromStr != "" {
			fromDate, _ = time.Parse("2006-01-02", fromStr)
		}
		if toStr != "" {
			toDate, _ = time.Parse("2006-01-02", toStr)
			toDate = toDate.Add(24*time.Hour - time.Second)
		}
	default:
		// "all" or empty - no date filter
	}

	var allEntries []models.TimeEntry
	result, err := models.GetEntriesByUserID(user.ID, "", 1, 10000)
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}
	allEntries = result.Entries

	var entries []models.TimeEntry
	for _, e := range allEntries {
		entryDate, _ := time.Parse("2006-01-02", e.Date)

		if !fromDate.IsZero() && entryDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && entryDate.After(toDate) {
			continue
		}

		if status == "unbilled" && e.Billed {
			continue
		}
		if status == "billed" && !e.Billed {
			continue
		}

		if clientFilter != "" && clientFilter != "all" {
			cid, _ := strconv.Atoi(clientFilter)
			if cid > 0 && (e.ClientID == nil || *e.ClientID != cid) {
				continue
			}
		}

		entries = append(entries, e)
	}

	// Build title
	titleParts := []string{"Arbeitsstunden"}
	switch rangeParam {
	case "1m": titleParts = append(titleParts, "Letzter Monat")
	case "3m": titleParts = append(titleParts, "Letzte 3 Monate")
	case "6m": titleParts = append(titleParts, "Letzte 6 Monate")
	case "1y": titleParts = append(titleParts, "Letztes Jahr")
	case "custom": titleParts = append(titleParts, fmt.Sprintf("%s - %s", r.URL.Query().Get("from"), r.URL.Query().Get("to")))
	default: titleParts = append(titleParts, "Gesamt")
	}
	switch status {
	case "unbilled": titleParts = append(titleParts, "Offene")
	case "billed": titleParts = append(titleParts, "Abgerechnete")
	default: titleParts = append(titleParts, "Alle")
	}
	if clientFilter != "" && clientFilter != "all" {
		cid, _ := strconv.Atoi(clientFilter)
		client, err := models.GetClientByID(cid)
		if err == nil {
			titleParts = append(titleParts, client.Name)
		}
	}
	filterInfo := strings.Join(titleParts, " - ")
	title := fmt.Sprintf("Arbeitsstunden %d %02d", now.Year(), int(now.Month()))

	profile, _ := models.GetUserSettings(user.ID)

	var client *models.Client
	if clientFilter != "" && clientFilter != "all" {
		cid, _ := strconv.Atoi(clientFilter)
		client, _ = models.GetClientByID(cid)
	}

	models.LogActivity(user.ID, "Export", fmt.Sprintf("%d Einträge als PDF exportiert", len(entries)))

	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, cleanPDF(title), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, cleanPDF(filterInfo), "", 1, "C", false, 0, "")
	pdf.Ln(2)

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

	w.Header().Set("Content-Type", "application/pdf")
	var filename string
	if client != nil {
		filename = fmt.Sprintf("Arbeitsstunden_%s_%s.pdf", cleanFilename(client.Name), time.Now().Format("2006-01"))
	} else {
		filename = fmt.Sprintf("Arbeitsstunden_%s.pdf", time.Now().Format("2006-01"))
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	pdf.Output(w)
}
