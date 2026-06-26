package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"goofytime/database"
	"goofytime/handlers"
	"goofytime/middleware"
	"goofytime/models"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "goofytime.db"
	}

	database.Init(dbPath)
	handlers.InitTemplates()

	go func() {
		for {
			time.Sleep(1 * time.Hour)
			models.CleanupSessions()
		}
	}()

	go handlers.StartTelegramBot()
	go handlers.StartScheduler()

	ensureFirstRun()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /onboarding", handlers.OnboardingPage)
	mux.HandleFunc("POST /onboarding", handlers.OnboardingStep)

	mux.HandleFunc("GET /login", handlers.LoginPage)
	mux.HandleFunc("POST /login", handlers.Login)
	mux.HandleFunc("GET /logout", handlers.Logout)

	mux.HandleFunc("GET /dashboard", withAuth(handlers.Dashboard))
	mux.HandleFunc("GET /entries/new", withAuth(handlers.NewEntryForm))
	mux.HandleFunc("POST /entries", withAuth(handlers.CreateEntry))
	mux.HandleFunc("GET /entries/{id}/edit", withAuth(handlers.EditEntryForm))
	mux.HandleFunc("POST /entries/{id}/update", withAuth(handlers.UpdateEntry))
	mux.HandleFunc("DELETE /entries/{id}", withAuth(handlers.DeleteEntry))
	mux.HandleFunc("GET /entries/{id}/row", withAuth(handlers.HTMLEntryRow))
	mux.HandleFunc("POST /entries/{id}/toggle-billed", withAuth(handlers.ToggleBilled))

	mux.HandleFunc("GET /export", withAuth(handlers.ExportPDF))
	mux.HandleFunc("GET /send-mail", withAuth(handlers.SendMailForm))
	mux.HandleFunc("POST /send-mail", withAuth(handlers.SendMail))

	mux.HandleFunc("GET /mail-settings", withAuth(handlers.MailSettingsPage))
	mux.HandleFunc("POST /mail-settings/save", withAuth(handlers.SaveMailSettings))
	mux.HandleFunc("POST /mail-settings/test", withAuth(handlers.SendTestMail))
	mux.HandleFunc("POST /mail-settings/trigger-auto", withAuth(handlers.TriggerAutoMail))

	mux.HandleFunc("GET /profile", withAuth(handlers.ProfilePage))
	mux.HandleFunc("POST /profile", withAuth(handlers.SaveProfile))

	mux.HandleFunc("GET /settings", withAuth(handlers.SettingsPage))
	mux.HandleFunc("POST /settings", withAuth(handlers.SaveSettings))
	mux.HandleFunc("GET /settings/import", withAuth(handlers.ImportCSVForm))
	mux.HandleFunc("POST /settings/import", withAuth(handlers.ImportCSV))
	mux.HandleFunc("GET /settings/export-csv", withAuth(handlers.ExportCSV))

	mux.HandleFunc("GET /clients", withAuth(handlers.ClientsPage))
	mux.HandleFunc("POST /clients", withAuth(handlers.CreateClient))
	mux.HandleFunc("GET /clients/{id}/edit", withAuth(handlers.EditClientForm))
	mux.HandleFunc("POST /clients/{id}/update", withAuth(handlers.UpdateClient))
	mux.HandleFunc("DELETE /clients/{id}", withAuth(handlers.DeleteClient))
	mux.HandleFunc("POST /clients/quick", withAuth(handlers.QuickCreateClient))
	mux.HandleFunc("GET /clients/{id}/preview-mail", withAuth(handlers.PreviewClientMail))
	mux.HandleFunc("POST /clients/{id}/send-mail", withAuth(handlers.SendClientMail))

	mux.HandleFunc("GET /log", withAuth(handlers.LogPage))
	mux.HandleFunc("GET /stats", withAuth(handlers.StatsPage))

	mux.HandleFunc("GET /admin", withAuth(withAdmin(handlers.AdminPage)))
	mux.HandleFunc("GET /admin/users/new", withAuth(withAdmin(handlers.NewUserForm)))
	mux.HandleFunc("POST /admin/users", withAuth(withAdmin(handlers.CreateUser)))
	mux.HandleFunc("GET /admin/users/{id}/edit", withAuth(withAdmin(handlers.EditUserForm)))
	mux.HandleFunc("POST /admin/users/{id}/update", withAuth(withAdmin(handlers.UpdateUser)))
	mux.HandleFunc("DELETE /admin/users/{id}", withAuth(withAdmin(handlers.DeleteUser)))
	mux.HandleFunc("GET /admin/settings", withAuth(withAdmin(handlers.AdminSettingsPage)))
	mux.HandleFunc("POST /admin/settings", withAuth(withAdmin(handlers.SaveAdminSettings)))
	mux.HandleFunc("GET /admin/backup", withAuth(withAdmin(handlers.BackupDatabase)))
	mux.HandleFunc("POST /admin/restore", withAuth(withAdmin(handlers.RestoreDatabase)))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server startet auf http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := middleware.Auth(middleware.CSRF(http.HandlerFunc(next)))
		handler.ServeHTTP(w, r)
	}
}

func withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := middleware.Admin(http.HandlerFunc(next))
		handler.ServeHTTP(w, r)
	}
}

func ensureFirstRun() {
	count, _ := models.GetUserCount()
	if count == 0 {
		log.Println("Keine Benutzer gefunden – Onboarding unter /onboarding")
	}
}
