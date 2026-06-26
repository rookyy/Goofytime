# Goofytime

> **Hinweis:** Der Code dieses Projekts wurde in großen Teilen mithilfe von KI (Large Language Models) generiert.

Webbasierte Arbeitszeiterfassung mit Multi-User-Support, Auftraggeber-Verwaltung, PDF-Export und Telegram-Integration.

## Onboarding

Beim ersten Start (keine Benutzer in der Datenbank) öffnet sich automatisch ein 5-stufiger Einrichtungsassistent unter `/onboarding`:

| Schritt | Inhalt |
|---|---|
| **1. Admin-Konto** | Benutzername und Passwort für den Administrator festlegen. Alternativ: Datenbank-Backup importieren. |
| **2. Persönliche Daten** | Vor- und Nachname für E-Mails und PDF-Exporte. |
| **3. Erster Auftraggeber** | Optional einen ersten Auftraggeber anlegen oder überspringen. |
| **4. Telegram** | Optional einen Telegram-Bot-Token für monatliche Benachrichtigungen hinterlegen oder überspringen. |
| **5. Abschluss** | App-Titel und Footer-Text festlegen, dann startet die Anwendung. |

Nach Abschluss wird automatisch eingeloggt (`justin`/`test` als Fallback wenn kein Name vergeben wurde).

## Funktionen

- **Zeiterfassung** mit Start/Stop-Uhr, Pause, Auftraggeber-Zuordnung
- **Dashboard** mit globalem Filter (Zeitraum, Auftraggeber, Einträge pro Seite)
- **Auftraggeber** mit Kontaktdaten, Empfängern, Mailtext/Betreff, Stundenlohn, Auto-Mail
- **PDF-Export** mit flexiblem Zeitraum-, Status- und Auftraggeber-Filter
- **Mail-Versand** per SMTP mit PDF-Anhang, Testmail-Funktion, pro Auftraggeber
- **Telegram-Bot** – monatliche Zusammenfassung mit Bestätigungs-Buttons, `/status`, `/monat`, `/export`
- **Statistiken** – Stunden und Vergütung pro Auftraggeber
- **Aktivitätslog** – Chronik aller Aktionen mit Filter
- **Multi-User** – Admin verwaltet Benutzer, jeder hat eigene Daten
- **Admin-Einstellungen** – Header-Titel, Footer-Text, DB-Backup

## Schnellstart

```bash
go run .
# http://localhost:8080
# Beim ersten Start: Onboarding unter /onboarding
# Bei bestehender DB: Login justin / test
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
| `DB_PATH` | `/app/data/goofytime.db` | Pfad zur SQLite-DB |
| `TZ` | `UTC` | Zeitzone (z.B. `Europe/Berlin`) |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token für Auto-Mail-Benachrichtigungen |
| `APP_ENCRYPTION_KEY` | — | 32-Byte-Hex-Key für SMTP-Passwort-Verschlüsselung (`openssl rand -hex 32`) |

### SMTP + Mail

SMTP wird **nicht** via Umgebungsvariablen, sondern pro Benutzer im Web-UI unter **Mail** konfiguriert. Der `APP_ENCRYPTION_KEY` verschlüsselt die SMTP-Passwörter in der Datenbank (AES-256-GCM). Ohne Key werden Passwörter im Klartext gespeichert.

## Umgebungsvariablen

| Variable | Standard | Beschreibung |
|----------|----------|-------------|
| `PORT` | `8080` | HTTP-Port |
| `DB_PATH` | `goofytime.db` | Pfad zur SQLite-Datenbank |
| `TELEGRAM_BOT_TOKEN` | — | Telegram Bot Token |
| `APP_ENCRYPTION_KEY` | — | 32-Byte-Hex für SMTP-Passwort-Verschlüsselung |

SMTP-Einstellungen werden pro Benutzer im Web-UI unter **Mail** konfiguriert.

## Telegram-Bot

Mit `TELEGRAM_BOT_TOKEN` läuft ein Telegram-Bot, der folgende Befehle unterstützt:

| Befehl | Funktion |
|---|---|
| `/start` | Chat-ID anzeigen und Befehlsübersicht |
| `/status` | Offene Stunden nach Auftraggeber gruppiert |
| `/monat` | Monatsübersicht mit allen Einträgen |
| `/export` | PDF des aktuellen Monats als Dokument senden |

Zusätzlich läuft täglich um 9:00 Uhr ein Scheduler, der für jeden Nutzer prüft, ob für den Vormonat unbezahlte Einträge existieren. Falls ja, wird eine Telegram-Nachricht mit Bestätigungs-Buttons gesendet (`✅ Senden` / `❌ Abbrechen`). Bei Bestätigung werden die Mails pro Auftraggeber mit PDF-Anhang versendet.

Der Auto-Mail-Tag (1-28) ist in den Mail-Einstellungen konfigurierbar.

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
└── goofytime.db             # SQLite-DB (auto-erstellt)
```
