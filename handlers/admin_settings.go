package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"goofytime/database"
	"goofytime/middleware"
	"goofytime/models"
)

func AdminSettingsPage(w http.ResponseWriter, r *http.Request) {
	settings := models.GetAdminSettings()
	msg := r.URL.Query().Get("message")

	users, _ := models.GetAllUsers()
	usersWithStats := buildUsersWithStats(users)

	RenderTemplate(w, "admin_settings.html", map[string]interface{}{
		"Title":    "Admin-Einstellungen",
		"User":     middleware.GetUser(r),
		"Settings": settings,
		"Users":    usersWithStats,
		"Message":  msg,
	})
}

func SaveAdminSettings(w http.ResponseWriter, r *http.Request) {
	headerTitle := r.FormValue("header_title")
	footerText := r.FormValue("footer_text")
	models.SaveAdminSettings(headerTitle, footerText, "")
	http.Redirect(w, r, "/admin/settings?message=Einstellungen+gespeichert", http.StatusSeeOther)
}

func BackupDatabase(w http.ResponseWriter, r *http.Request) {
	// Flush WAL to main database file
	database.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	time.Sleep(100 * time.Millisecond)

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" { dbPath = "goofytime.db" }

	data, err := os.ReadFile(dbPath)
	if err != nil {
		http.Error(w, "Backup fehlgeschlagen", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=goofytime_backup_%s.db", time.Now().Format("2006-01-02_1504")))
	w.Write(data)
}

func RestoreDatabase(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("db_file")
	if err != nil {
		http.Redirect(w, r, "/admin/settings?message=Keine+Datei+ausgewählt", http.StatusSeeOther)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".db") {
		http.Redirect(w, r, "/admin/settings?message=Nur+.db-Dateien+erlaubt", http.StatusSeeOther)
		return
	}
	if header.Size > 100*1024*1024 {
		http.Redirect(w, r, "/admin/settings?message=Datei+zu+groß+(max.+100+MB)", http.StatusSeeOther)
		return
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "goofytime.db"
	}

	if database.DB != nil {
		database.DB.Close()
	}

	dst, err := os.Create(dbPath)
	if err != nil {
		http.Redirect(w, r, "/admin/settings?message=Fehler+beim+Import", http.StatusSeeOther)
		return
	}
	io.Copy(dst, file)
	dst.Close()

	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")

	database.Init(dbPath)
	InitTemplates()

	models.LogActivity(middleware.GetUser(r).ID, "DB-Restore", "Datenbank aus Backup wiederhergestellt")
	http.Redirect(w, r, "/logout", http.StatusSeeOther)
}
