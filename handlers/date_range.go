package handlers

import (
	"net/http"
	"time"
)

func parseDateRange(r *http.Request) (fromDate, toDate time.Time, rangeParam string) {
	rangeParam = r.URL.Query().Get("range")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	now := time.Now()

	switch rangeParam {
	case "1m": fromDate = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "3m": fromDate = time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "6m": fromDate = time.Date(now.Year(), now.Month()-6, 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "1y": fromDate = time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location()); toDate = now
	case "custom":
		if fromStr != "" { fromDate, _ = time.Parse("2006-01-02", fromStr) }
		if toStr != "" { toDate, _ = time.Parse("2006-01-02", toStr); toDate = toDate.Add(24*time.Hour - time.Second) }
	}
	return
}
