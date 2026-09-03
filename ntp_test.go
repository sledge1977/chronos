package main

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestUnsynchronizedNTPResponse(t *testing.T) {
	request := make([]byte, ntpPacketSize)
	request[0] = 4<<3 | 3
	request[2] = 6
	for index := 40; index < 48; index++ {
		request[index] = byte(index)
	}

	now := time.Date(2026, time.September, 3, 12, 34, 56, 123_000_000, time.UTC)
	response, ok := createNTPResponse(request, now, now, ntpClock{stratum: 16, referenceTime: now})
	if !ok {
		t.Fatal("gültige Client-Anfrage wurde abgelehnt")
	}
	if len(response) != ntpPacketSize {
		t.Fatalf("Antwortlänge = %d, erwartet %d", len(response), ntpPacketSize)
	}
	if leap := response[0] >> 6; leap != 3 {
		t.Fatalf("Leap Indicator = %d, erwartet 3", leap)
	}
	if mode := response[0] & 0x07; mode != 4 {
		t.Fatalf("Modus = %d, erwartet 4 (Server)", mode)
	}
	if response[1] != 16 {
		t.Fatalf("Stratum = %d, erwartet 16", response[1])
	}
	if got := string(response[12:16]); got != "INIT" {
		t.Fatalf("Reference ID = %q, erwartet INIT", got)
	}
	if got := response[24:32]; string(got) != string(request[40:48]) {
		t.Fatal("Originate Timestamp wurde nicht aus der Anfrage übernommen")
	}
	if binary.BigEndian.Uint32(response[40:44]) == 0 {
		t.Fatal("Transmit Timestamp fehlt")
	}
}

func TestSynchronizedNTPResponse(t *testing.T) {
	request := make([]byte, ntpPacketSize)
	request[0] = 4<<3 | 3
	now := time.Now().UTC()

	response, ok := createNTPResponse(request, now, now, ntpClock{stratum: 3, referenceTime: now})
	if !ok {
		t.Fatal("gültige Client-Anfrage wurde abgelehnt")
	}
	if leap := response[0] >> 6; leap != 0 {
		t.Fatalf("Leap Indicator = %d, erwartet 0", leap)
	}
	if response[1] != 3 {
		t.Fatalf("Stratum = %d, erwartet 3", response[1])
	}
}

func TestInvalidNTPRequestsAreIgnored(t *testing.T) {
	now := time.Now().UTC()
	clock := newNTPClock(16)

	if _, ok := createNTPResponse(make([]byte, 47), now, now, clock); ok {
		t.Fatal("zu kurze Anfrage wurde akzeptiert")
	}

	notAClient := make([]byte, ntpPacketSize)
	notAClient[0] = 4<<3 | 4
	if _, ok := createNTPResponse(notAClient, now, now, clock); ok {
		t.Fatal("Paket im Server-Modus wurde akzeptiert")
	}
}

func TestConfiguredStratum(t *testing.T) {
	for _, test := range []struct {
		value string
		want  uint8
		valid bool
	}{
		{"", 16, true},
		{"1", 1, true},
		{"15", 15, true},
		{"16", 16, true},
		{"0", 0, false},
		{"17", 0, false},
		{"abc", 0, false},
	} {
		got, err := configuredStratum(test.value)
		if test.valid && (err != nil || got != test.want) {
			t.Errorf("configuredStratum(%q) = %d, %v; erwartet %d", test.value, got, err, test.want)
		}
		if !test.valid && err == nil {
			t.Errorf("configuredStratum(%q) akzeptierte ungültigen Wert", test.value)
		}
	}
}

func TestNTPRequestLogKeepsNewestEntries(t *testing.T) {
	requestLog := newNTPRequestLog(2)
	client := &net.UDPAddr{IP: net.ParseIP("192.0.2.10"), Port: 49152}
	request := make([]byte, ntpPacketSize)
	request[0] = 4<<3 | 3
	start := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)

	requestLog.record(client, request, start, "beantwortet")
	requestLog.record(client, request, start.Add(time.Second), "beantwortet")
	requestLog.record(client, request, start.Add(2*time.Second), "beantwortet")

	snapshot := requestLog.snapshot(newNTPClock(15))
	if snapshot.Total != 3 || snapshot.Retained != 2 {
		t.Fatalf("Snapshot = total %d, retained %d; erwartet 3 und 2", snapshot.Total, snapshot.Retained)
	}
	if snapshot.Requests[0].ID != 3 || snapshot.Requests[1].ID != 2 {
		t.Fatalf("IDs = %d, %d; erwartet 3, 2", snapshot.Requests[0].ID, snapshot.Requests[1].ID)
	}
	if snapshot.Requests[0].ClientIP != "192.0.2.10" || snapshot.Requests[0].ClientPort != 49152 {
		t.Fatalf("Client nicht korrekt erfasst: %+v", snapshot.Requests[0])
	}
}
