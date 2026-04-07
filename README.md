HTTPCannon

# build
go build -o httpcannon httpcannon.go

# unlimited goroutines, run forever
./httpcannon -url "https://example.com/page?foo=bar"

# fixed 200 goroutines for 60 seconds
./httpcannon -url "https://example.com" -threads 200 -duration 60s

# cap at 500 req/s, run for 5 minutes
./httpcannon -url "https://example.com" -threads 100 -rps 500 -duration 5m

# supply your own UA and Referer lists
./httpcannon -url "https://example.com" -ua-file useragents.txt -ref-file referers.txt

# 500 goroutines hammering, but only 50 TCP connections open at once
./httpcannon -url "https://example.com" -threads 500 -conns 50

# fully unconstrained (default)
./httpcannon -url "https://example.com"

# tight connection budget with a rate cap
./httpcannon -url "https://example.com" -conns 10 -rps 100 -duration 30s
