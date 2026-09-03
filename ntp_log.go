package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type ntpRequestEntry struct {
	ID         uint64    `json:"id"`
	ReceivedAt time.Time `json:"receivedAt"`
	ClientIP   string    `json:"clientIp"`
	ClientPort int       `json:"clientPort"`
	Version    int       `json:"version"`
	Mode       int       `json:"mode"`
	Bytes      int       `json:"bytes"`
	Result     string    `json:"result"`
}

type ntpRequestLog struct {
	mutex    sync.RWMutex
	capacity int
	total    uint64
	entries  []ntpRequestEntry
}

type ntpRequestSnapshot struct {
	Total    uint64            `json:"total"`
	Retained int               `json:"retained"`
	Capacity int               `json:"capacity"`
	Stratum  uint8             `json:"stratum"`
	Synced   bool              `json:"synchronized"`
	Requests []ntpRequestEntry `json:"requests"`
}

func newNTPRequestLog(capacity int) *ntpRequestLog {
	if capacity < 1 {
		capacity = 1
	}
	return &ntpRequestLog{
		capacity: capacity,
		entries:  make([]ntpRequestEntry, 0, capacity),
	}
}

func (requestLog *ntpRequestLog) record(remoteAddress net.Addr, request []byte, receivedAt time.Time, result string) {
	clientIP := remoteAddress.String()
	clientPort := 0
	if udpAddress, ok := remoteAddress.(*net.UDPAddr); ok {
		clientIP = udpAddress.IP.String()
		clientPort = udpAddress.Port
	}

	version := 0
	mode := 0
	if len(request) > 0 {
		version = int(request[0]>>3) & 0x07
		mode = int(request[0] & 0x07)
	}

	requestLog.mutex.Lock()
	requestLog.total++
	entry := ntpRequestEntry{
		ID:         requestLog.total,
		ReceivedAt: receivedAt.UTC(),
		ClientIP:   clientIP,
		ClientPort: clientPort,
		Version:    version,
		Mode:       mode,
		Bytes:      len(request),
		Result:     result,
	}
	if len(requestLog.entries) == requestLog.capacity {
		copy(requestLog.entries, requestLog.entries[1:])
		requestLog.entries[len(requestLog.entries)-1] = entry
	} else {
		requestLog.entries = append(requestLog.entries, entry)
	}
	requestLog.mutex.Unlock()

	log.Printf("NTP request from %s:%d: version %d, mode %d, %d bytes, %s", entry.ClientIP, entry.ClientPort, entry.Version, entry.Mode, entry.Bytes, entry.Result)
}

func (requestLog *ntpRequestLog) snapshot(clock ntpClock) ntpRequestSnapshot {
	requestLog.mutex.RLock()
	defer requestLog.mutex.RUnlock()

	requests := make([]ntpRequestEntry, len(requestLog.entries))
	for index := range requestLog.entries {
		requests[index] = requestLog.entries[len(requestLog.entries)-1-index]
	}

	return ntpRequestSnapshot{
		Total:    requestLog.total,
		Retained: len(requests),
		Capacity: requestLog.capacity,
		Stratum:  clock.stratum,
		Synced:   clock.synchronized(),
		Requests: requests,
	}
}

func serveNTPRequests(requestLog *ntpRequestLog, clock ntpClock) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(requestLog.snapshot(clock))
	}
}
