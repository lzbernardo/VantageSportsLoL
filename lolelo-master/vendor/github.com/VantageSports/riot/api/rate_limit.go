package api

import "time"

// CallRate describes the number of calls the caller is allowed to make, and the
// duration over which that limit applies.
type CallRate struct {
	CallsPer int
	Dur      time.Duration
}

// DevRate is a "smoothed" rate limit for typical development keys, which allow
// 10 requests a second, but only 500 per 10 minutes, which is effectively 8.3
// per 10 seconds.
var DevRate = CallRate{CallsPer: 8, Dur: time.Second * 10}

// NOTE: we don't provide a "ProdRate" here because we want to discourage any
// one consumer from maximizing out the API key, as it's typically shared
// between many consumers.

type RateLimiter interface {
	Wait()     // Blocks until the request is allowed to be sent.
	Complete() // Signal that a request has completed.
}

// perDurationLimiter is a rate limiter that permits a pre-specified number
// of calls per a pre-specified time frame.
type perDurationLimiter struct {
	duration time.Duration
	tokens   chan bool
}

func NewDurationLimiter(limit int, duration time.Duration) *perDurationLimiter {
	tokens := make(chan bool, limit*2)
	for i := 0; i < limit; i++ {
		tokens <- true
	}
	return &perDurationLimiter{duration: duration, tokens: tokens}
}

func (p *perDurationLimiter) Wait() {
	<-p.tokens
}

func (p *perDurationLimiter) Complete() {
	go func(dur time.Duration) {
		time.Sleep(dur)
		p.tokens <- true
	}(p.duration)
}

type ComposedLimiter struct {
	limiters []RateLimiter
}

func (c *ComposedLimiter) Wait() {
	for _, limiter := range c.limiters {
		limiter.Wait()
	}
}

func (c *ComposedLimiter) Complete() {
	for _, limiter := range c.limiters {
		limiter.Complete()
	}
}
