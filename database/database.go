package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) {
	dir := filepath.Dir(dbPath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	DB.SetMaxOpenConns(1)

	migrate()
}

func migrate() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			is_admin INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			address TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS time_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			client_id INTEGER,
			date TEXT NOT NULL,
			time_from TEXT NOT NULL,
			time_to TEXT NOT NULL,
			hours REAL NOT NULL,
			purpose TEXT NOT NULL DEFAULT '',
			location TEXT NOT NULL DEFAULT '',
			billed INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER UNIQUE NOT NULL,
			chat_id TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS user_settings (
			user_id INTEGER PRIMARY KEY,
			auto_mail_day INTEGER NOT NULL DEFAULT 1,
			page_size INTEGER NOT NULL DEFAULT 30,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS scheduler_state (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			last_check_month TEXT NOT NULL DEFAULT '',
			last_notification_sent INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS admin_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			header_title TEXT NOT NULL DEFAULT 'Stundenerfassung',
			footer_text TEXT NOT NULL DEFAULT 'Made by Cold-IT'
		)`,
		`INSERT OR IGNORE INTO admin_settings (id, header_title, footer_text) VALUES (1, 'Stundenerfassung', 'Made by Cold-IT')`,
		`CREATE INDEX IF NOT EXISTS idx_time_entries_user_id ON time_entries(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_time_entries_date ON time_entries(date)`,
		`CREATE INDEX IF NOT EXISTS idx_time_entries_billed ON time_entries(billed)`,
		`CREATE INDEX IF NOT EXISTS idx_time_entries_client_id ON time_entries(client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_clients_user_id ON clients(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token)`,
		`CREATE TABLE IF NOT EXISTS activity_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_log_user_id ON activity_log(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_log_created_at ON activity_log(created_at)`,
		`CREATE TABLE IF NOT EXISTS mail_settings (
			user_id INTEGER PRIMARY KEY,
			smtp_host TEXT NOT NULL DEFAULT '',
			smtp_port INTEGER NOT NULL DEFAULT 587,
			smtp_user TEXT NOT NULL DEFAULT '',
			smtp_pass TEXT NOT NULL DEFAULT '',
			smtp_from TEXT NOT NULL DEFAULT '',
			default_mail_text TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatal("Migration failed:", err)
		}
	}

	DB.Exec(`ALTER TABLE time_entries ADD COLUMN billed INTEGER NOT NULL DEFAULT 0`)
	DB.Exec(`ALTER TABLE time_entries ADD COLUMN client_id INTEGER REFERENCES clients(id) ON DELETE SET NULL`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN full_name TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN street TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN zip_city TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN phone TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN email TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN recipients TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN mail_text TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN hourly_rate REAL NOT NULL DEFAULT 0.0`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN contact_name TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN contact_phone TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN contact_email TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN auto_mail_enabled INTEGER NOT NULL DEFAULT 1`)
	DB.Exec(`ALTER TABLE mail_settings ADD COLUMN default_mail_text TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE mail_settings ADD COLUMN email_subject TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE clients ADD COLUMN mail_subject TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN first_name TEXT NOT NULL DEFAULT ''`)
	DB.Exec(`ALTER TABLE user_settings ADD COLUMN last_name TEXT NOT NULL DEFAULT ''`)
}
