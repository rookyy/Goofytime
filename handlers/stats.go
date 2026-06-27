package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"goofytime/middleware"
	"goofytime/models"
)

type ClientStats struct {
	Client          models.Client
	TotalHours      float64
	BilledHours     float64
	UnbilledHours   float64
	BilledAmount    float64
	UnbilledAmount  float64
	PctOfTotal      float64
}

type MonthlyStats struct {
	Month          string
	TotalHours     float64
	BilledHours    float64
	UnbilledHours  float64
	BilledAmount   float64
	UnbilledAmount float64
}

func StatsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	var fromDate, toDate time.Time
	if fromStr != "" {
		fromDate, _ = time.Parse("2006-01-02", fromStr)
	}
	if toStr != "" {
		toDate, _ = time.Parse("2006-01-02", toStr)
		toDate = toDate.Add(24*time.Hour - time.Second)
	}

	clients, _ := models.GetClientsByUserID(user.ID)
	entries, _ := models.GetEntriesByUserID(user.ID, "", 1, 10000)

	var filtered []models.TimeEntry
	for _, e := range entries.Entries {
		entryDate, _ := time.Parse("2006-01-02", e.Date)
		if !fromDate.IsZero() && entryDate.Before(fromDate) {
			continue
		}
		if !toDate.IsZero() && entryDate.After(toDate) {
			continue
		}
		filtered = append(filtered, e)
	}

	var allStats []ClientStats
	totalBilled := 0.0
	totalUnbilled := 0.0
	totalBilledHours := 0.0
	totalUnbilledHours := 0.0
	totalHours := 0.0

	for _, c := range clients {
		cs := ClientStats{Client: c}
		for _, e := range filtered {
			if e.ClientID != nil && *e.ClientID == c.ID {
				cs.TotalHours += e.Hours
				if e.Billed {
					cs.BilledHours += e.Hours
				} else {
					cs.UnbilledHours += e.Hours
				}
			}
		}
		cs.BilledAmount = cs.BilledHours * c.HourlyRate
		cs.UnbilledAmount = cs.UnbilledHours * c.HourlyRate
		if cs.TotalHours > 0 {
			allStats = append(allStats, cs)
			totalBilled += cs.BilledAmount
			totalUnbilled += cs.UnbilledAmount
			totalBilledHours += cs.BilledHours
			totalUnbilledHours += cs.UnbilledHours
			totalHours += cs.TotalHours
		}
	}

	sort.Slice(allStats, func(i, j int) bool {
		return allStats[i].TotalHours > allStats[j].TotalHours
	})

	for i := range allStats {
		if totalHours > 0 {
			allStats[i].PctOfTotal = allStats[i].TotalHours / totalHours * 100
		}
	}

	var noClientHours, noClientBilled, noClientUnbilled float64
	for _, e := range filtered {
		if e.ClientID == nil {
			noClientHours += e.Hours
			if e.Billed {
				noClientBilled += e.Hours
			} else {
				noClientUnbilled += e.Hours
			}
		}
	}

	monthlyStats := buildMonthlyStats(filtered, clients)

	filterDesc := "Gesamt"
	if fromStr != "" || toStr != "" {
		filterDesc = fmt.Sprintf("%s – %s", fromStr, toStr)
	}

	RenderTemplate(w, "stats.html", map[string]interface{}{
		"Title":               "Statistiken",
		"User":                user,
		"Stats":               allStats,
		"TotalBilled":         totalBilled,
		"TotalUnbilled":       totalUnbilled,
		"TotalBilledHours":    totalBilledHours,
		"TotalUnbilledHours":  totalUnbilledHours,
		"TotalHours":          totalHours,
		"NoClientHours":       noClientHours,
		"NoClientBilled":      noClientBilled,
		"NoClientUnbilled":    noClientUnbilled,
		"From":                fromStr,
		"To":                  toStr,
		"FilterDesc":          filterDesc,
		"MonthlyStats":        monthlyStats,
	})
}

func buildMonthlyStats(entries []models.TimeEntry, clients []models.Client) []MonthlyStats {
	clientByID := map[int]models.Client{}
	for _, c := range clients {
		clientByID[c.ID] = c
	}

	monthMap := map[string]*MonthlyStats{}
	for _, e := range entries {
		if len(e.Date) >= 7 {
			key := e.Date[:7]
			if _, ok := monthMap[key]; !ok {
				monthMap[key] = &MonthlyStats{Month: key}
			}
			ms := monthMap[key]
			ms.TotalHours += e.Hours
			if e.Billed {
				ms.BilledHours += e.Hours
			} else {
				ms.UnbilledHours += e.Hours
			}
		}
	}

	var months []string
	for k := range monthMap {
		months = append(months, k)
	}
	sort.Strings(months)

	var result []MonthlyStats
	for i := len(months) - 1; i >= 0; i-- {
		m := months[i]
		ms := monthMap[m]
		for _, c := range clients {
			for _, e := range entries {
				if e.ClientID != nil && *e.ClientID == c.ID && len(e.Date) >= 7 && e.Date[:7] == m {
					if e.Billed {
						ms.BilledAmount += e.Hours * c.HourlyRate
					} else {
						ms.UnbilledAmount += e.Hours * c.HourlyRate
					}
				}
			}
		}
		result = append(result, *ms)
	}

	if len(result) > 24 {
		result = result[len(result)-24:]
	}

	for i := range result {
		t, _ := time.Parse("2006-01", result[i].Month)
		gerMonth := germanMonths[int(t.Month())]
		result[i].Month = fmt.Sprintf("%s %d", gerMonth, t.Year())
	}

	return result
}
