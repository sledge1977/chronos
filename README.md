# Chronos — Go Time Server

Chronos is a small Go server that provides a responsive website displaying local time, UTC, and Unix time. It also answers NTPv3 and NTPv4 requests over UDP. The website shows the latest 200 NTP requests in real time, while a total counter tracks all requests since startup. Every request is also written to the regular service log. All web assets are embedded in the compiled executable.

## Run

Requires Go 1.22 or newer.

```powershell
go run .
```

Then open <http://localhost:8080> in a browser.

At startup, Chronos prints every local address at which the NTP server is available. It uses UDP port 123 by default, for example `127.0.0.1:123/udp`.

Test it on Windows:

```powershell
w32tm /stripchart /computer:127.0.0.1 /dataonly /samples:5
```

Set a different web port if required:

```powershell
$env:PORT = "9000"
go run .
```

The NTP port can also be changed, for example when port 123 is already in use:

```powershell
$env:NTP_ADDR = ":8123"
go run .
```

Note that many NTP clients only support the standard destination port 123.

## Stratum and synchronization state

Without additional configuration, the NTP server deliberately operates as an **unsynchronized clock**. It continues to respond, but identifies itself with stratum 16 and sets the leap indicator to 3. This allows clients to recognize that the time source is not confirmed as synchronized.

If the host clock is known to be synchronized by an external time source, set the corresponding stratum from 1 through 15:

```powershell
$env:NTP_STRATUM = "3"
go run .
```

Chronos does not modify the system clock and therefore does not claim synchronization unless explicitly configured.

## HTTP endpoints

- `/` — website
- `/api/time` — current server time as JSON
- `/api/ntp/requests` — NTP request log and server state as JSON
- `/healthz` — basic health check

The NTP service runs independently over UDP and is not exposed as an HTTP endpoint.

## Install as a Windows service

The files in `service` install Chronos as an automatically starting Windows service using WinSW. Output logs are rotated by size in `C:\Program Files\Chronos\logs`. When required, the installation script downloads the WinSW wrapper from its official GitHub release, verifies its SHA-256 checksum, and creates an inbound firewall rule for UDP port 123.

First build the executable:

```powershell
go build -o chronos.exe .
```

Then run the installation script from an elevated PowerShell session:

```powershell
.\service\install-service.ps1
```

Remove the service:

```powershell
.\service\uninstall-service.ps1
```

Follow the live log:

```powershell
Get-Content "C:\Program Files\Chronos\logs\chronos-service.err.log" -Encoding UTF8 -Wait
```
