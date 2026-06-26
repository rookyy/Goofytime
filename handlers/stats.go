package handlers

import (
	"net/http"

	"stundenerfassung/middleware"
	"stundenerfassung/models"
)

type ClientStats struct {
	Client        models.Client
	TotalHours    float64
	BilledHours   float64
	UnbilledHours float64
	BilledAmount  float64
	UnbilledAmount float64
}

func StatsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	clients, _ := models.GetClientsByUserID(user.ID)
	entries, _ := models.GetEntriesByUserID(user.ID, "", 1, 10000)

	var allStats []ClientStats
	totalBilled := 0.0
	totalUnbilled := 0.0
	totalBilledHours := 0.0
	totalUnbilledHours := 0.0

	for _, c := range clients {
		cs := ClientStats{Client: c}
		for _, e := range entries.Entries {
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
		}
	}

	// Also add entries without client
	var noClientHours, noClientBilled, noClientUnbilled float64
	for _, e := range entries.Entries {
		if e.ClientID == nil {
			noClientHours += e.Hours
			if e.Billed {
				noClientBilled += e.Hours
			} else {
				noClientUnbilled += e.Hours
			}
		}
	}

	RenderTemplate(w, "stats.html", map[string]interface{}{
		"Title":            "Statistiken",
		"User":             user,
		"Stats":            allStats,
		"TotalBilled":      totalBilled,
		"TotalUnbilled":    totalUnbilled,
		"TotalBilledHours": totalBilledHours,
		"TotalUnbilledHours": totalUnbilledHours,
		"NoClientHours":    noClientHours,
		"NoClientBilled":   noClientBilled,
		"NoClientUnbilled": noClientUnbilled,
	})
}
