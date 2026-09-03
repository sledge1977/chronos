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
		t.Fatal("valid client request was rejected")
	}
	if len(response) != ntpPacketSize {
		t.Fatalf("response length = %d, expected %d", len(response), ntpPacketSize)
	}
	if leap := response[0] >> 6; leap != 3 {
		t.Fatalf("leap indicator = %d, expected 3", leap)
	}
	if mode := response[0] & 0x07; mode != 4 {
		t.Fatalf("mode = %d, expected 4 (server)", mode)
	}
	if response[1] != 16 {
		t.Fatalf("stratum = %d, expected 16", response[1])
	}
	if got := string(response[12:16]); got != "INIT" {
		t.Fatalf("reference ID = %q, expected INIT", got)
	}
	if got := response[24:32]; string(got) != string(request[40:48]) {
		t.Fatal("originate timestamp was not copied from the request")
	}
	if binary.BigEndian.Uint32(response[40:44]) == 0 {
		t.Fatal("transmit timestamp is missing")
	}
}

func TestSynchronizedNTPResponse(t *testing.T) {
	request := make([]byte, ntpPacketSize)
	request[0] = 4<<3 | 3
	now := time.Now().UTC()

	response, ok := createNTPResponse(request, now, now, ntpClock{stratum: 3, referenceTime: now})
	if !ok {
		t.Fatal("valid client request was rejected")
	}
	if leap := response[0] >> 6; leap != 0 {
		t.Fatalf("leap indicator = %d, expected 0", leap)
	}
	if response[1] != 3 {
		t.Fatalf("stratum = %d, expected 3", response[1])
	}
}

func TestInvalidNTPRequestsAreIgnored(t *testing.T) {
	now := time.Now().UTC()
	clock := newNTPClock(16)

	if _, ok := createNTPResponse(make([]byte, 47), now, now, clock); ok {
		t.Fatal("short request was accepted")
	}

	notAClient := make([]byte, ntpPacketSize)
	notAClient[0] = 4<<3 | 4
	if _, ok := createNTPResponse(notAClient, now, now, clock); ok {
		t.Fatal("server-mode packet was accepted")
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
			t.Errorf("configuredStratum(%q) = %d, %v; expected %d", test.value, got, err, test.want)
		}
		if !test.valid && err == nil {
			t.Errorf("configuredStratum(%q) accepted an invalid value", test.value)
		}
	}
}

func TestNTPRequestLogKeepsNewestEntries(t *testing.T) {
	requestLog := newNTPRequestLog(2)
	client := &net.UDPAddr{IP: net.ParseIP("192.0.2.10"), Port: 49152}
	request := make([]byte, ntpPacketSize)
	request[0] = 4<<3 | 3
	start := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)

	requestLog.record(client, request, start, "answered")
	requestLog.record(client, request, start.Add(time.Second), "answered")
	requestLog.record(client, request, start.Add(2*time.Second), "answered")

	snapshot := requestLog.snapshot(newNTPClock(15))
	if snapshot.Total != 3 || snapshot.Retained != 2 {
		t.Fatalf("snapshot = total %d, retained %d; expected 3 and 2", snapshot.Total, snapshot.Retained)
	}
	if snapshot.Requests[0].ID != 3 || snapshot.Requests[1].ID != 2 {
		t.Fatalf("IDs = %d, %d; expected 3, 2", snapshot.Requests[0].ID, snapshot.Requests[1].ID)
	}
	if snapshot.Requests[0].ClientIP != "192.0.2.10" || snapshot.Requests[0].ClientPort != 49152 {
		t.Fatalf("client was not recorded correctly: %+v", snapshot.Requests[0])
	}
}
