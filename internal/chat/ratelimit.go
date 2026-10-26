package chat

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter aplica un límite por usuario autenticado o por IP para invitados.
// Es en memoria; suficiente para una sola instancia de la API.
type RateLimiter struct {
	limit       rate.Limit
	burst       int
	mu          sync.Mutex
	clients     map[string]*rateClient
	lastCleanup time.Time
}

func NewRateLimiter(perMinute int) *RateLimiter {
	if perMinute <= 0 {
		perMinute = DefaultRateLimitPerMin
	}
	return &RateLimiter{
		limit:       rate.Limit(float64(perMinute) / 60.0),
		burst:       perMinute,
		clients:     make(map[string]*rateClient),
		lastCleanup: time.Now(),
	}
}

func (r *RateLimiter) Allow(key string) bool {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	if now.Sub(r.lastCleanup) > 5*time.Minute {
		for k, c := range r.clients {
			if now.Sub(c.lastSeen) > 10*time.Minute {
				delete(r.clients, k)
			}
		}
		r.lastCleanup = now
	}

	client, ok := r.clients[key]
	if !ok {
		client = &rateClient{
			limiter: rate.NewLimiter(r.limit, r.burst),
		}
		r.clients[key] = client
	}
	client.lastSeen = now
	return client.limiter.Allow()
}
