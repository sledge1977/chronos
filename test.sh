$server = "127.0.0.1"
$port = 8123

$client = [System.Net.Sockets.UdpClient]::new()
$client.Client.ReceiveTimeout = 3000

$request = [byte[]]::new(48)
$request[0] = 0x23  # NTPv4, client mode

[void]$client.Send($request, $request.Length, $server, $port)

$remote = [System.Net.IPEndPoint]::new(
    [System.Net.IPAddress]::Any,
    0
)

try {
    $response = $client.Receive([ref]$remote)

    $seconds =
        ([uint64]$response[40] -shl 24) -bor
        ([uint64]$response[41] -shl 16) -bor
        ([uint64]$response[42] -shl 8)  -bor
        [uint64]$response[43]

    $fraction =
        ([uint64]$response[44] -shl 24) -bor
        ([uint64]$response[45] -shl 16) -bor
        ([uint64]$response[46] -shl 8)  -bor
        [uint64]$response[47]

    $unixTime = [double]$seconds - 2208988800 +
                ([double]$fraction / 4294967296)

    $utc = [DateTimeOffset]::FromUnixTimeMilliseconds(
        [int64]($unixTime * 1000)
    )

    [pscustomobject]@{
        ResponseFrom  = $remote
        Bytes         = $response.Length
        NTPVersion    = ($response[0] -shr 3) -band 7
        Mode          = $response[0] -band 7
        LeapIndicator = $response[0] -shr 6
        Stratum       = $response[1]
        UTC           = $utc.UtcDateTime
        LocalTime     = $utc.LocalDateTime
    } | Format-List
}
catch {
    Write-Error "No NTP response: $($_.Exception.Message)"
}
finally {
    $client.Dispose()
}
