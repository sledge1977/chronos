package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

//go:embed web/*
var webFiles embed.FS

func main() {
	ntpAddress := envOrDefault("NTP_ADDR", ":123")
	ntpStratum, err := configuredStratum(os.Getenv("NTP_STRATUM"))
	if err != nil {
		log.Fatal(err)
	}
	webEnabled, err := configuredWebEnabled(os.Getenv("WEB_ENABLED"))
	if err != nil {
		log.Fatal(err)
	}

	ntpConnection, err := net.ListenPacket("udp", ntpAddress)
	if err != nil {
		log.Fatalf("NTP server cannot use %s/udp: %v", ntpAddress, err)
	}
	defer ntpConnection.Close()

	clock := newNTPClock(ntpStratum)
	requestLog := newNTPRequestLog(200)
	logNTPReachability(ntpConnection.LocalAddr(), clock)

	if !webEnabled {
		log.Print("Web server is disabled")
		if err := serveNTP(ntpConnection, clock, requestLog); err != nil {
			log.Fatalf("NTP server stopped: %v", err)
		}
		return
	}

	go func() {
		if err := serveNTP(ntpConnection, clock, requestLog); err != nil {
			log.Fatalf("NTP server stopped: %v", err)
		}
	}()

	publicFiles, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/time", serveTime)
	mux.HandleFunc("GET /api/ntp/requests", serveNTPRequests(requestLog, clock))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("GET /", http.FileServer(http.FS(publicFiles)))

	port := envOrDefault("PORT", "8080")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Time server is running at http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func configuredStratum(value string) (uint8, error) {
	if value == "" {
		return 16, nil
	}

	stratum, err := strconv.Atoi(value)
	if err != nil || stratum < 1 || stratum > 16 {
		return 0, errors.New("NTP_STRATUM must be an integer between 1 and 16")
	}
	return uint8(stratum), nil
}

func configuredWebEnabled(value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.New("WEB_ENABLED must be a boolean")
	}
	return enabled, nil
}

func serveTime(w http.ResponseWriter, _ *http.Request) {
	now := time.Now()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(struct {
		UnixMilliseconds int64  `json:"unixMilliseconds"`
		UTC              string `json:"utc"`
	}{
		UnixMilliseconds: now.UnixMilli(),
		UTC:              now.UTC().Format(time.RFC3339Nano),
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
