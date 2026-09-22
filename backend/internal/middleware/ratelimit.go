package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/isasumer/go-react-calculator/backend/internal/httpapi"
)

// rateLimitedDetail points at the header rather than repeating a number the
// client would have to parse out of prose.
const rateLimitedDetail = "too many requests; wait for the time in the Retry-After header and try again"

// DefaultIdleTTL is how long a client's bucket survives its last request.
// Buckets are tiny, but one per source address with nothing removing them is
// unbounded memory paid for by whoever sends the most addresses.
const DefaultIdleTTL = 10 * time.Minute

// sweepEvery ties the cost of eviction to the growth that makes it necessary:
// every sweepEvery-th new client also walks the map and drops what has gone
// idle. A ticker would mean a goroutine to start, own and stop — lifecycle
// for a map that only grows when someone new arrives.
const sweepEvery = 64

// RateLimitConfig configures [RateLimit]. The first three fields come
// straight from the environment (RATE_LIMIT_RPS, RATE_LIMIT_BURST,
// TRUST_PROXY_HEADERS).
type RateLimitConfig struct {
	// RPS is the sustained rate a single client is allowed, and Burst how
	// many requests it may make back to back before that rate applies. A
	// keypad user is bursty and then idle, which is exactly the shape a
	// token bucket is for. Either below 1 disables the middleware.
	RPS   int
	Burst int

	// TrustProxyHeaders makes the limiter key on the first X-Forwarded-For
	// entry instead of the peer address. It is off by default and it is a
	// deployment decision: see [clientKey].
	TrustProxyHeaders bool

	// IdleTTL overrides [DefaultIdleTTL]. Zero means the default.
	IdleTTL time.Duration

	// Now is the clock. Zero means time.Now; tests inject one so a refill
	// can be observed without sleeping through it.
	Now func() time.Time
}

// RateLimit gives every client its own token bucket and answers a client over
// its budget with a problem+json 429 and a Retry-After, without ever reaching
// the router.
//
// It is the last gate before the router, so a rejected request costs a map
// lookup. It skips the operational endpoints: throttling a probe because an
// API client is hammering the service would take a healthy instance out of
// rotation at the worst possible moment.
//
// The limit is per process. Two replicas mean twice the rate; a limit that
// has to hold across replicas belongs in the edge proxy, not here. What this
// buys is that one client cannot occupy a whole instance.
func RateLimit(cfg RateLimitConfig) Middleware {
	return func(next http.Handler) http.Handler {
		if cfg.RPS < 1 || cfg.Burst < 1 {
			return next
		}
		buckets := newBucketStore(cfg)
		now := cfg.Now
		if now == nil {
			now = time.Now
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if operationalPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			wait, ok := buckets.allow(clientKey(r, cfg.TrustProxyHeaders), now())
			if ok {
				next.ServeHTTP(w, r)
				return
			}
			// Retry-After is whole seconds (RFC 9110), and 0 would invite an
			// immediate retry, so the shortest answer is 1.
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(math.Ceil(wait.Seconds())))))
			httpapi.Write(w, r, httpapi.NewProblem(httpapi.CodeRateLimited, rateLimitedDetail))
		})
	}
}

// clientKey identifies the client a bucket belongs to.
//
// RemoteAddr is the only address the process can vouch for: it is the peer
// the kernel accepted the connection from. X-Forwarded-For is whatever the
// caller typed, unless a proxy we control overwrites it — so trusting it is a
// deployment decision, off by default. Trusting it with nothing rewriting it
// hands every client an unlimited supply of fresh buckets; not trusting it
// behind a proxy puts every client in one bucket. Both are wrong in the other
// deployment, which is why this is a switch and not a default.
//
// The first entry is the original client; the rest are the proxies it came
// through.
func clientKey(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if first, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ","); strings.TrimSpace(first) != "" {
			return strings.TrimSpace(first)
		}
	}
	return remoteIP(r.RemoteAddr)
}

// bucket is one client's allowance and when it was last seen, which is what
// makes it evictable.
type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// bucketStore is the map of clients to buckets, with the mutex that makes it
// safe and the eviction that keeps it bounded.
type bucketStore struct {
	rps   rate.Limit
	burst int
	ttl   time.Duration

	mu      sync.Mutex
	buckets map[string]*bucket
	inserts int
}

func newBucketStore(cfg RateLimitConfig) *bucketStore {
	ttl := cfg.IdleTTL
	if ttl <= 0 {
		ttl = DefaultIdleTTL
	}
	return &bucketStore{
		rps:     rate.Limit(cfg.RPS),
		burst:   cfg.Burst,
		ttl:     ttl,
		buckets: make(map[string]*bucket),
	}
}

// allow takes one token for key. When there is none it reports how long until
// the next one, which is what Retry-After has to say.
func (s *bucketStore) allow(key string, now time.Time) (wait time.Duration, ok bool) {
	s.mu.Lock()
	b := s.buckets[key]
	if b == nil {
		b = &bucket{limiter: rate.NewLimiter(s.rps, s.burst), lastSeen: now}
		s.buckets[key] = b
		s.inserts++
		if s.inserts%sweepEvery == 0 {
			s.sweepLocked(now)
		}
	}
	b.lastSeen = now
	s.mu.Unlock()

	// Reserve rather than Allow: a reservation that cannot be met still says
	// when it could be, and canceling it puts the token back so a rejected
	// request does not delay the one that would have been allowed.
	// rate.Limiter has a lock of its own, so this is outside ours.
	res := b.limiter.ReserveN(now, 1)
	if !res.OK() {
		// Unreachable with burst >= 1; an unfulfillable reservation would
		// otherwise become an infinite Retry-After.
		return time.Second, false
	}
	if d := res.DelayFrom(now); d > 0 {
		res.CancelAt(now)
		return d, false
	}
	return 0, true
}

// sweepLocked drops the buckets nobody has used for a TTL. A client that
// comes back simply gets a full one, which is the same answer it would have
// got from a bucket that had been refilling all along.
func (s *bucketStore) sweepLocked(now time.Time) {
	for key, b := range s.buckets {
		if now.Sub(b.lastSeen) > s.ttl {
			delete(s.buckets, key)
		}
	}
}

// size is the number of live buckets; the eviction test reads it.
func (s *bucketStore) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.buckets)
}
