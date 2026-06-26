<div align="center">
  <h1>⏱️ Goofytime</h1>
  <p>Webbasierte Arbeitszeiterfassung</p>

  <img src="https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/database-SQLite-003B57?logo=sqlite" alt="SQLite">
  <img src="https://img.shields.io/badge/frontend-Tailwind%20%2B%20HTMX-06B6D4" alt="Tailwind">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="License">
</div>

<br>

> **⚠️ KI-Hinweis** — Der Code dieses Projekts wurde in großen Teilen mithilfe von KI (Large Language Models) generiert.

---

## 🚀 Schnellstart

```bash
go run .
# → http://localhost:8080
# Beim ersten Start öffnet sich das Onboarding
```

---

## 🧭 Onboarding

Beim ersten Start führt ein **5-stufiger Assistent** durch die Einrichtung:

| Schritt | |
|---|---|
| **1. Admin-Konto** | Benutzername & Passwort festlegen. Alternativ DB-Backup importieren. |
| **2. Persönliche Daten** | Vor- und Nachname für E-Mails und PDF-Exporte. |
| **3. Erster Auftraggeber** | Optional anlegen oder überspringen. |
| **4. Telegram** | Bot-Token für Benachrichtigungen (optional). |
| **5. Abschluss** | App-Titel & Footer festlegen. Danach automatischer Login. |

---

## ✨ Funktionen

| Bereich | Details |
|---|---|
| **Zeiterfassung** | Start/Stop-Uhr mit Pause, Auftraggeber-Zuordnung, Beschreibung |
| **Dashboard** | Globaler Filter (Zeitraum, Auftraggeber), Paginierung, Einträge pro Seite |
| **Auftraggeber** | Name, Adresse, Kontaktdaten, Empfänger, Mailtext & Betreff, Stundenlohn, Auto-Mail |
| **PDF-Export** | Flexibler Zeitraum-, Status- und Auftraggeber-Filter |
| **Mail-Versand** | SMTP, PDF-Anhang, Testmail, pro Auftraggeber mit eigenen Texten/Betreff |
| **Telegram-Bot** | Commands, monatliche Abrechnung mit Bestätigung, Scheduler |
| **Statistiken** | Stunden & Vergütung pro Auftraggeber |
| **Aktivitätslog** | Chronik aller Aktionen mit Filter |
| **Multi-User** | Admin verwaltet Benutzer, jeder hat eigene Daten |
| **Betreff-Platzhalter** | `%M` = Monat, `%J` = Jahr, `%N` = Nachname aus Profil |

---

## 🤖 Telegram-Bot

Mit `TELEGRAM_BOT_TOKEN` wird ein Telegram-Bot gestartet:

| Befehl | Beschreibung |
|---|---|
| `/start` | Chat-ID anzeigen & Befehlsübersicht |
| `/status` | Offene Stunden gruppiert nach Auftraggeber |
| `/monat` | Monatsübersicht aller Einträge |
| `/export` | PDF des Monats als Dokument |
| `✅ Senden` | E-Mail-Versand bestätigen (Inline-Button) |
| `❌ Abbrechen` | Versand abbrechen (Inline-Button) |

**Scheduler:** Täglich 9:00 Uhr Prüfung des Vormonats auf unbezahlte Einträge → Telegram-Nachricht mit Bestätigungs-Buttons. Auto-Mail-Tag (1-28) in den Mail-Einstellungen konfigurierbar.

---

## 🐳 Docker

```bash
docker compose up -d
```

Die Datenbank wird in `./data` persistiert, Container läuft als `appuser`.

| Variable | Standard | Beschreibung |
|---|---|---|
| `PORT` | `8080` | HTTP-Port |
| `DB_PATH` | `/app/data/goofytime.db` | Pfad zur SQLite-DB |
| `TZ` | `UTC` | Zeitzone (z.B. `Europe/Berlin`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token |
| `APP_ENCRYPTION_KEY` | — | 32-Byte-Hex für SMTP-Passwort-Verschlüsselung |

---

## ⚙️ Konfiguration

### Umgebungsvariablen

| Variable | Standard | Beschreibung |
|---|---|---|
| `PORT` | `8080` | HTTP-Port |
| `DB_PATH` | `goofytime.db` | Pfad zur SQLite-Datenbank |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token |
| `APP_ENCRYPTION_KEY` | — | 32-Byte-Hex für SMTP-Passwort-Verschlüsselung |

SMTP wird pro Benutzer im Web-UI unter **Mail** eingerichtet (nicht via Umgebungsvariablen).  
`APP_ENCRYPTION_KEY` verschlüsselt SMTP-Passwörter in der DB (AES-256-GCM). Ohne Key → Klartext.

---

## 📁 Struktur

```
├── main.go                  # Entry Point & Routing
├── database/                # SQLite & Migrationen
├── models/                  # DB-Modelle
├── handlers/                # HTTP-Handler
├── middleware/               # Auth, Admin, CSRF
├── templates/               # Go-Templates (Tailwind + HTMX)
├── Dockerfile
├── docker-compose.yml
└── goofytime.db             # SQLite-DB (auto-erstellt)
```
