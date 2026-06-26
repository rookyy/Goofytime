package main

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"stundenerfassung/models"
)

func importFromExcel() (int, error) {
	f, err := excelize.OpenFile("Stunden.xlsx")
	if err != nil {
		return 0, fmt.Errorf("Excel-Datei nicht lesbar: %w", err)
	}
	defer f.Close()

	count := 0
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		for _, row := range rows {
			if len(row) < 5 {
				continue
			}

			dateStr := row[0]
			timeFrom := row[1]
			timeTo := row[2]
			hoursStr := row[3]
			purpose := row[4]
			location := ""
			if len(row) >= 6 {
				location = row[5]
			}

			if dateStr == "" || dateStr == "Datum" || timeFrom == "" || timeTo == "" || purpose == "" || purpose == "Zweck" {
				continue
			}

			date, err := parseDate(dateStr)
			if err != nil {
				continue
			}

			hours := parseHours(hoursStr)

			_, err = models.CreateEntry(1, nil, date.Format("2006-01-02"), timeFrom, timeTo, hours, purpose, location)
			if err == nil {
				count++
			}
		}
	}

	return count, nil
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"01-02-06",
		"1/2/06",
		"02/01/2006",
		"2006-01-02",
	}

	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func parseHours(s string) float64 {
	if s == "" {
		return 0
	}

	s = trimSpace(s)
	if s == "0" || s == "0:00" {
		return 0
	}

	t, err := time.Parse("15:04", s)
	if err == nil {
		hours := float64(t.Hour()) + float64(t.Minute())/60.0
		if hours > 0 {
			return hours
		}
	}

	t2, err := time.Parse("15:04:05", s)
	if err == nil {
		hours := float64(t2.Hour()) + float64(t2.Minute())/60.0 + float64(t2.Second())/3600.0
		if hours > 0 {
			return hours
		}
	}

	hours := 0.0
	totalMinutes := 0
	fmt.Sscanf(s, "%f", &hours)
	if hours > 0 {
		return hours
	}

	totalMinutes = 0
	fmt.Sscanf(s, "%d:%d", &hours, &totalMinutes)
	h := float64(int(hours)) + float64(totalMinutes)/60.0
	return h
}

func trimSpace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
