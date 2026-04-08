# HTTPCannon

A simple high-performance HTTP load testing tool written in Go.

## Build

```bash
go build -o httpcannon httpcannon.go
```

## Usage

### Run with unlimited goroutines (runs indefinitely)
```bash
./httpcannon -url "https://example.com/page?foo=bar"
```

### Run with fixed 200 goroutines for 60 seconds
```bash
./httpcannon -url "https://example.com" -threads 200 -duration 60s
```

### Cap at 500 requests per second for 5 minutes
```bash
./httpcannon -url "https://example.com" -threads 100 -rps 500 -duration 5m
```

### Supply custom User-Agent and Referer lists
```bash
./httpcannon -url "https://example.com" -ua-file useragents.txt -ref-file referers.txt
```

### 500 goroutines with only 50 concurrent TCP connections
```bash
./httpcannon -url "https://example.com" -threads 500 -conns 50
```

### Fully unconstrained (default settings)
```bash
./httpcannon -url "https://example.com"
```

### Tight connection limit with rate cap
```bash
./httpcannon -url "https://example.com" -conns 10 -rps 100 -duration 30s
```

### mTLS: send client cert, but don't verify server cert
```bash
./httpcannon -url https://example.com -mtls-cert client.crt -mtls-key client.key
```
### mTLS: send client cert + verify server against your CA
```bash
./httpcannon -url https://example.com -mtls-cert client.crt -mtls-key client.key -mtls-ca ca.crt
```
### CA-only: no client cert, but verify server (standard TLS with custom CA)
```bash
./httpcannon -url https://internal-service:8443 -mtls-ca internal-ca.crt
```

## Flags

| Flag        | Description |
|------------|------------|
| `-url`      | Target URL |
| `-threads`  | Number of goroutines |
| `-duration` | Test duration (e.g., `30s`, `5m`) |
| `-rps`      | Requests per second limit |
| `-conns`    | Maximum concurrent TCP connections |
| `-ua-file`  | File containing User-Agent strings |
| `-ref-file` | File containing Referer values |
| `-mtls-cert` | Client certificate PEM file |
| `-mtls-key` | Client private key PEM file |
| `-mtls-ca` | CA certificate PEM to verify the server |


## Notes

- If no limits are specified, the tool runs with maximum throughput.
- Use `-rps` to avoid overwhelming the target server.
- Combine flags for fine-grained control over load behavior.
