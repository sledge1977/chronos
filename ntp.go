package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sort"
	"time"
)

const (
	ntpPacketSize = 48
	ntpEpochDelta = 2_208_988_800
)

type ntpClock struct {
	stratum       uint8
	referenceTime time.Time
}

func newNTPClock(stratum uint8) ntpClock {
	return ntpClock{
		stratum:       stratum,
		referenceTime: time.Now().UTC(),
	}
}

func (c ntpClock) synchronized() bool {
	return c.stratum >= 1 && c.stratum <= 15
}

func serveNTP(connection net.PacketConn, clock ntpClock, requestLog *ntpRequestLog) error {
	buffer := make([]byte, 512)

	for {
		length, remoteAddress, err := connection.ReadFrom(buffer)
		if err != nil {
			return err
		}

		receivedAt := time.Now().UTC()
		request := append([]byte(nil), buffer[:length]...)
		response, ok := createNTPResponse(request, receivedAt, time.Now().UTC(), clock)
		if !ok {
			requestLog.record(remoteAddress, request, receivedAt, "ignoriert")
			continue
		}

		if _, err := connection.WriteTo(response, remoteAddress); err != nil {
			requestLog.record(remoteAddress, request, receivedAt, "fehler")
			log.Printf("NTP-Antwort an %s fehlgeschlagen: %v", remoteAddress, err)
			continue
		}
		requestLog.record(remoteAddress, request, receivedAt, "beantwortet")
	}
}

func createNTPResponse(request []byte, receivedAt, transmittedAt time.Time, clock ntpClock) ([]byte, bool) {
	if len(request) < ntpPacketSize || request[0]&0x07 != 3 {
		return nil, false
	}

	version := (request[0] >> 3) & 0x07
	if version < 3 || version > 4 {
		version = 4
	}

	leapIndicator := byte(0)
	referenceID := "LOCL"
	rootDispersion := uint32(1 << 16)
	if !clock.synchronized() {
		leapIndicator = 3
		referenceID = "INIT"
		rootDispersion = uint32(16 << 16)
	}

	response := make([]byte, ntpPacketSize)
	response[0] = leapIndicator<<6 | version<<3 | 4 // LI, Version, Server-Modus
	response[1] = clock.stratum
	response[2] = request[2]
	response[3] = 0xec // signierter NTP-Präzisionsexponent -20
	binary.BigEndian.PutUint32(response[8:12], rootDispersion)
	copy(response[12:16], referenceID)
	putNTPTimestamp(response[16:24], clock.referenceTime)
	copy(response[24:32], request[40:48])
	putNTPTimestamp(response[32:40], receivedAt)
	putNTPTimestamp(response[40:48], transmittedAt)

	return response, true
}

func putNTPTimestamp(destination []byte, timestamp time.Time) {
	seconds := uint64(timestamp.Unix() + ntpEpochDelta)
	fraction := uint64(timestamp.Nanosecond()) << 32 / 1_000_000_000
	binary.BigEndian.PutUint32(destination[0:4], uint32(seconds))
	binary.BigEndian.PutUint32(destination[4:8], uint32(fraction))
}

func logNTPReachability(address net.Addr, clock ntpClock) {
	udpAddress, ok := address.(*net.UDPAddr)
	if !ok {
		log.Printf("NTP-Server erreichbar unter %s", address)
		return
	}

	state := "synchronisiert"
	if !clock.synchronized() {
		state = "nicht synchronisiert; Antworten werden mit LI=3 gesendet"
	}
	log.Printf("NTP-Server aktiv auf UDP-Port %d (Stratum %d, %s)", udpAddress.Port, clock.stratum, state)

	for _, endpoint := range reachableNTPEndpoints(udpAddress) {
		log.Printf("NTP erreichbar: %s", endpoint)
	}
	log.Printf("NTP-Test unter Windows: w32tm /stripchart /computer:127.0.0.1 /dataonly /samples:5")
}

func reachableNTPEndpoints(boundAddress *net.UDPAddr) []string {
	port := boundAddress.Port
	if !boundAddress.IP.IsUnspecified() {
		return []string{net.JoinHostPort(boundAddress.IP.String(), fmt.Sprint(port)) + "/udp"}
	}

	endpoints := []string{net.JoinHostPort("127.0.0.1", fmt.Sprint(port)) + "/udp"}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return endpoints
	}

	seen := map[string]bool{"127.0.0.1": true}
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err != nil || ip.IsLoopback() || ip.IsUnspecified() || seen[ip.String()] {
			continue
		}
		seen[ip.String()] = true
		endpoints = append(endpoints, net.JoinHostPort(ip.String(), fmt.Sprint(port))+"/udp")
	}

	sort.Strings(endpoints[1:])
	return endpoints
}
