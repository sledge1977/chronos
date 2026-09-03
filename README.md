# Chronos – Go-Zeitserver

Ein kleiner Go-Webserver, der eine responsive Webseite mit lokaler Uhrzeit, UTC und Unix-Zeit ausliefert. Zusätzlich beantwortet das Programm NTPv3-/NTPv4-Anfragen über UDP. Die Webseite zeigt die letzten 200 NTP-Anfragen live an; der Gesamtzähler läuft bis zum nächsten Programmstart weiter. Alle Anfragen werden außerdem in das normale Service-Log geschrieben. Alle Webdateien werden beim Kompilieren in die Programmdatei eingebettet.

## Starten

Voraussetzung: Go 1.22 oder neuer.

```powershell
go run .
```

Danach im Browser öffnen: <http://localhost:8080>

Beim Start gibt das Programm alle lokalen Adressen aus, unter denen der NTP-Server erreichbar ist. Standardmäßig verwendet er UDP-Port 123, beispielsweise `127.0.0.1:123/udp`.

Test unter Windows:

```powershell
w32tm /stripchart /computer:127.0.0.1 /dataonly /samples:5
```

Optional kann ein anderer Port gesetzt werden:

```powershell
$env:PORT = "9000"
go run .
```

Der NTP-Port kann ebenfalls geändert werden, zum Beispiel für einen Start ohne Administratorrechte oder wenn Port 123 bereits belegt ist:

```powershell
$env:NTP_ADDR = ":8123"
go run .
```

## Stratum und Synchronisationsstatus

Ohne weitere Konfiguration arbeitet der NTP-Server bewusst als **nicht synchronisierte Uhr**: Er antwortet weiterhin, setzt aber gemäß NTP-Protokoll Stratum 16 und den Leap Indicator auf 3. Dadurch erkennen Clients, dass die Zeitquelle aktuell nicht als synchronisiert bestätigt ist.

Wenn die Uhr des Rechners nachweislich durch eine externe Zeitquelle synchronisiert wird, kann das zugehörige Stratum zwischen 1 und 15 angegeben werden:

```powershell
$env:NTP_STRATUM = "3"
go run .
```

Das Programm verändert die Systemzeit nicht und behauptet deshalb ohne diese explizite Konfiguration keine Synchronisation.

## Endpunkte

- `/` – Webseite
- `/api/time` – aktuelle Serverzeit als JSON
- `/api/ntp/requests` – NTP-Anfragen und Serverstatus als JSON
- `/healthz` – einfache Statusprüfung

Der NTP-Dienst läuft unabhängig davon über UDP und nicht über einen HTTP-Endpunkt.

## Als Windows-Dienst installieren

Die Dateien im Verzeichnis `service` richten Chronos mithilfe von WinSW als automatisch startenden Windows-Dienst ein. Die Ausgaben werden größenbasiert in `C:\Program Files\Chronos\logs` rotiert. Das Installationsskript lädt einen fehlenden WinSW-Wrapper aus dem offiziellen GitHub-Release, prüft dessen SHA-256-Prüfsumme und richtet außerdem die eingehende Firewallfreigabe für UDP-Port 123 ein.

Installation in einer PowerShell mit Administratorrechten:

```powershell
.\service\install-service.ps1
```

Deinstallation:

```powershell
.\service\uninstall-service.ps1
```

Laufendes Protokoll anzeigen:

```powershell
Get-Content "C:\Program Files\Chronos\logs\chronos-service.err.log" -Encoding UTF8 -Wait
```
