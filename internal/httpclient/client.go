// Package httpclient provides an HTTP client tuned for hotline.ua:
// realistic browser headers, a user-agent rotation pool, a global
// token-bucket rate limiter, an in-memory LRU response cache, and a
// browser-grade TLS fingerprint (via bogdanfinn/tls-client) to avoid
// Cloudflare bot detection.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
)

// ErrBotBlock is returned when hotline.ua responds with a Cloudflare challenge
// or other anti-bot block. Callers should surface this as an actionable message
// rather than a generic HTTP error.
var ErrBotBlock = errors.New("hotline.ua blocked the request (Cloudflare challenge); try again later or reduce request frequency")

const (
	BaseURL     = "https://hotline.ua"
	LocalePath  = "/ua"
	defaultRPS  = 1.0
	defaultTTL  = 10 * time.Minute
	cacheSize   = 256
	httpTimeout = 30 * time.Second
)

// Chrome 133 UAs to match the TLS profile used by the client.
var userAgents = []string{
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
}

type cachedResponse struct {
	body        []byte
	contentType string
	storedAt    time.Time
}

type Client struct {
	http    tls_client.HttpClient
	limiter *rate.Limiter
	cache   *lru.Cache[string, cachedResponse]
	ttl     time.Duration
	rng     *rand.Rand
	mu      sync.Mutex
}

// New builds a Client using env vars for overrides:
//
//	HOTLINE_RATE_LIMIT_RPS  (float, default 1.0)
//	HOTLINE_CACHE_TTL_SEC   (int, default 600)
func New() (*Client, error) {
	rps := envFloat("HOTLINE_RATE_LIMIT_RPS", defaultRPS)
	ttlSec := envInt("HOTLINE_CACHE_TTL_SEC", int(defaultTTL.Seconds()))

	cache, err := lru.New[string, cachedResponse](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("init lru cache: %w", err)
	}

	jar := tls_client.NewCookieJar()

	tlsClient, err := tls_client.NewHttpClient(
		tls_client.NewNoopLogger(),
		tls_client.WithTimeoutMilliseconds(int(httpTimeout.Milliseconds())),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(jar),
		tls_client.WithNotFollowRedirects(),
	)
	if err != nil {
		return nil, fmt.Errorf("init tls client: %w", err)
	}

	return &Client{
		http:    tlsClient,
		limiter: rate.NewLimiter(rate.Limit(rps), 1),
		cache:   cache,
		ttl:     time.Duration(ttlSec) * time.Second,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Get fetches url, returning the body. Responses are cached for the configured
// TTL keyed by URL. Requests are rate-limited globally.
// Returns ErrBotBlock if Cloudflare or another anti-bot system intercepts.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	if v, ok := c.cache.Get(url); ok && time.Since(v.storedAt) < c.ttl {
		return v.body, nil
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := fhttp.NewRequestWithContext(ctx, fhttp.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.applyHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if isBotBlock(resp.StatusCode, body) {
		return nil, ErrBotBlock
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("hotline returned %d for %s (first 200 bytes: %q)",
			resp.StatusCode, url, truncate(body, 200))
	}

	c.cache.Add(url, cachedResponse{
		body:        body,
		contentType: resp.Header.Get("Content-Type"),
		storedAt:    time.Now(),
	})
	return body, nil
}

func (c *Client) applyHeaders(req *fhttp.Request) {
	c.mu.Lock()
	ua := userAgents[c.rng.Intn(len(userAgents))]
	c.mu.Unlock()

	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,application/json;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "uk-UA,uk;q=0.9,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

// isBotBlock returns true when the response looks like a Cloudflare challenge
// or anti-bot intercept page (503/403 + known body markers).
func isBotBlock(status int, body []byte) bool {
	if status != 503 && status != 403 {
		return false
	}
	preview := strings.ToLower(string(truncate(body, 512)))
	return strings.Contains(preview, "cloudflare") ||
		strings.Contains(preview, "cf-ray") ||
		strings.Contains(preview, "just a moment") ||
		strings.Contains(preview, "enable javascript") ||
		strings.Contains(preview, "challenge")
}

func envFloat(key string, def float64) float64 {
	if s := os.Getenv(key); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return def
}

func envInt(key string, def int) int {
	if s := os.Getenv(key); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return def
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}
