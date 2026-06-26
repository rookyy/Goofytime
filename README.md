# Stundenerfassung

Webbasierte Arbeitszeiterfassung mit Multi-User-Support, Auftraggeber-Verwaltung, PDF-Export und Telegram-Integration.

## Funktionen

- **Zeiterfassung** mit Start/Stop-Uhr, Pause, Auftraggeber-Zuordnung
- **Dashboard** mit globalem Filter (Zeitraum, Auftraggeber, Einträge pro Seite)
- **Auftraggeber** mit Kontaktdaten, Empfängern, Mailtext, Stundenlohn, Auto-Mail
- **PDF-Export** mit flexiblem Zeitraum-, Status- und Auftraggeber-Filter
- **Mail-Versand** per SMTP mit PDF-Anhang, Testmail-Funktion
- **Telegram-Bot** – monatliche Zusammenfassung mit Bestätigungs-Buttons
- **Statistiken** – Stunden und Vergütung pro Auftraggeber
- **Aktivitätslog** – Chronik aller Aktionen mit Filter
- **Multi-User** – Admin verwaltet Benutzer, jeder hat eigene Daten
- **Admin-Einstellungen** – Header-Titel, Footer-Text, DB-Backup

## Schnellstart

```bash
go run .
# http://localhost:8080
# Login: justin / test
```

## Docker

```bash
docker compose up -d
```

Die Datenbank wird im Volume `./data` persistiert. Der Container läuft als non-root User `appuser`.

### Docker-Umgebungsvariablen

Nur in `docker-compose.yml` oder via `-e` setzen – nicht im Web-UI:

| Variable | Standard | Beschreibung |
|----------|----------|-------------|
| `PORT` | `8080` | HTTP-Port |
| `DB_PATH` | `/app/data/stunden.db` | Pfad zur SQLite-DB |
| `TZ` | `UTC` | Zeitzone (z.B. `Europe/Berlin`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token für Auto-Mail-Benachrichtigungen |
| `APP_ENCRYPTION_KEY` | — | 32-Byte-Hex-Key für SMTP-Passwort-Verschlüsselung (`openssl rand -hex 32`) |

### SMTP + Mail

SMTP wird **nicht** via Umgebungsvariablen, sondern pro Benutzer im Web-UI unter **Mail** konfiguriert. Der `APP_ENCRYPTION_KEY` verschlüsselt die SMTP-Passwörter in der Datenbank (AES-256-GCM). Ohne Key werden Passwörter im Klartext gespeichert.

## Umgebungsvariablen

| Variable | Standard | Beschreibung |
|----------|----------|-------------|
| `PORT` | `8080` | HTTP-Port |
| `DB_PATH` | `stunden.db` | Pfad zur SQLite-Datenbank |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token |
| `APP_ENCRYPTION_KEY` | — | 32-Byte-Hex für SMTP-Passwort-Verschlüsselung |

SMTP-Einstellungen werden pro Benutzer im Web-UI unter **Mail** konfiguriert.

## Projektstruktur

```
├── main.go                  # Entry Point + Routing
├── database/                # SQLite + Migrationen
├── models/                  # DB-Modelle (User, Entry, Client, Settings, …)
├── handlers/                # HTTP-Handler
├── middleware/               # Auth, Admin, CSRF
├── templates/               # Go-Templates (Tailwind + HTMX + Alpine)
├── Dockerfile
├── docker-compose.yml
└── stunden.db               # SQLite-DB (auto-erstellt)
```
# Goofytime
# Goofytime
