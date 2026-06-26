package models

import (
	"fmt"
	"time"

	"goofytime/database"
)

type TimeEntry struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ClientID   *int      `json:"client_id"`
	ClientName string    `json:"client_name"`
	Date       string    `json:"date"`
	TimeFrom   string    `json:"time_from"`
	TimeTo     string    `json:"time_to"`
	Hours      float64   `json:"hours"`
	Purpose    string    `json:"purpose"`
	Location   string    `json:"location"`
	Billed     bool      `json:"billed"`
	CreatedAt  time.Time `json:"created_at"`
}

type PaginationResult struct {
	Entries    []TimeEntry
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

func GetEntriesByUserID(userID int, yearMonth string, page, pageSize int) (*PaginationResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}

	var countQuery string
	var dataQuery string
	args := []interface{}{}

	baseWhere := " WHERE e.user_id = ?"
	args = append(args, userID)

	if yearMonth != "" {
		baseWhere += " AND e.date LIKE ?"
		args = append(args, yearMonth+"%")
	}

	countQuery = "SELECT COUNT(*) FROM time_entries e" + baseWhere

	var total int
	if err := database.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize

	cols := "e.id, e.user_id, e.client_id, COALESCE(c.name, '') as client_name, e.date, e.time_from, e.time_to, e.hours, e.purpose, e.location, e.billed, e.created_at"
	dataQuery = fmt.Sprintf("SELECT %s FROM time_entries e LEFT JOIN clients c ON e.client_id = c.id%s ORDER BY e.date DESC, e.time_from DESC LIMIT ? OFFSET ?", cols, baseWhere)

	dataArgs := append([]interface{}{}, args...)
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := database.DB.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TimeEntry
	for rows.Next() {
		var e TimeEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.ClientID, &e.ClientName, &e.Date, &e.TimeFrom, &e.TimeTo, &e.Hours, &e.Purpose, &e.Location, &e.Billed, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return &PaginationResult{
		Entries:    entries,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func GetEntryByID(id int) (*TimeEntry, error) {
	e := &TimeEntry{}
	err := database.DB.QueryRow(
		"SELECT e.id, e.user_id, e.client_id, COALESCE(c.name, ''), e.date, e.time_from, e.time_to, e.hours, e.purpose, e.location, e.billed, e.created_at FROM time_entries e LEFT JOIN clients c ON e.client_id = c.id WHERE e.id = ?", id,
	).Scan(&e.ID, &e.UserID, &e.ClientID, &e.ClientName, &e.Date, &e.TimeFrom, &e.TimeTo, &e.Hours, &e.Purpose, &e.Location, &e.Billed, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func CreateEntry(userID int, clientID *int, date, timeFrom, timeTo string, hours float64, purpose, location string) (*TimeEntry, error) {
	result, err := database.DB.Exec(
		"INSERT INTO time_entries (user_id, client_id, date, time_from, time_to, hours, purpose, location) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		userID, clientID, date, timeFrom, timeTo, hours, purpose, location,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetEntryByID(int(id))
}

func UpdateEntry(id int, clientID *int, date, timeFrom, timeTo string, hours float64, purpose, location string) error {
	_, err := database.DB.Exec(
		"UPDATE time_entries SET client_id = ?, date = ?, time_from = ?, time_to = ?, hours = ?, purpose = ?, location = ? WHERE id = ?",
		clientID, date, timeFrom, timeTo, hours, purpose, location, id,
	)
	return err
}

func ToggleBilled(id int) (bool, error) {
	_, err := database.DB.Exec(
		"UPDATE time_entries SET billed = CASE WHEN billed = 0 THEN 1 ELSE 0 END WHERE id = ?", id,
	)
	if err != nil {
		return false, err
	}
	var billed bool
	err = database.DB.QueryRow("SELECT billed FROM time_entries WHERE id = ?", id).Scan(&billed)
	return billed, err
}

func DeleteEntry(id int) error {
	_, err := database.DB.Exec("DELETE FROM time_entries WHERE id = ?", id)
	return err
}

func GetUnbilledEntriesForUser(userID int) ([]TimeEntry, error) {
	rows, err := database.DB.Query(
		"SELECT e.id, e.user_id, e.client_id, COALESCE(c.name, ''), e.date, e.time_from, e.time_to, e.hours, e.purpose, e.location, e.billed, e.created_at FROM time_entries e LEFT JOIN clients c ON e.client_id = c.id WHERE e.user_id = ? AND e.billed = 0 ORDER BY e.date ASC, e.time_from ASC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TimeEntry
	for rows.Next() {
		var e TimeEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.ClientID, &e.ClientName, &e.Date, &e.TimeFrom, &e.TimeTo, &e.Hours, &e.Purpose, &e.Location, &e.Billed, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func GetUnbilledHoursForUser(userID int) (float64, error) {
	var total float64
	err := database.DB.QueryRow(
		"SELECT COALESCE(SUM(hours), 0) FROM time_entries WHERE user_id = ? AND billed = 0",
		userID,
	).Scan(&total)
	return total, err
}

func MarkEntriesAsBilled(ids []int) error {
	for _, id := range ids {
		if _, err := database.DB.Exec("UPDATE time_entries SET billed = 1 WHERE id = ?", id); err != nil {
			return err
		}
	}
	return nil
}

func GetUnbilledEntriesByMonth(userID int, yearMonth string) ([]TimeEntry, error) {
	rows, err := database.DB.Query(
		"SELECT e.id, e.user_id, e.client_id, COALESCE(c.name, ''), e.date, e.time_from, e.time_to, e.hours, e.purpose, e.location, e.billed, e.created_at FROM time_entries e LEFT JOIN clients c ON e.client_id = c.id WHERE e.user_id = ? AND e.billed = 0 AND e.date LIKE ? ORDER BY e.date ASC, e.time_from ASC",
		userID, yearMonth+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TimeEntry
	for rows.Next() {
		var e TimeEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.ClientID, &e.ClientName, &e.Date, &e.TimeFrom, &e.TimeTo, &e.Hours, &e.Purpose, &e.Location, &e.Billed, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func GetMonthlyHoursForUser(userID int, yearMonth string) (float64, error) {
	var total float64
	err := database.DB.QueryRow(
		"SELECT COALESCE(SUM(hours), 0) FROM time_entries WHERE user_id = ? AND date LIKE ?",
		userID, yearMonth+"%",
	).Scan(&total)
	return total, err
}

func GetAvailableMonths(userID int) ([]string, error) {
	rows, err := database.DB.Query(
		"SELECT DISTINCT substr(date, 1, 7) AS month FROM time_entries WHERE user_id = ? ORDER BY month DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var months []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		months = append(months, m)
	}
	return months, nil
}
