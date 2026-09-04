package auth

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterMaxFailures(t *testing.T) {
	limiter := NewLimiter(3, time.Minute)
	if limiter.Blocked("k") {
		t.Fatal("fresh key is already blocked")
	}

	limiter.Fail("k")
	limiter.Fail("k")
	if limiter.Blocked("k") {
		t.Fatal("blocked before reaching the limit")
	}

	limiter.Fail("k")
	if !limiter.Blocked("k") {
		t.Fatal("not blocked after max failures")
	}
}

func TestLimiterResetClearsFailures(t *testing.T) {
	limiter := NewLimiter(2, time.Minute)
	limiter.Fail("k")
	limiter.Fail("k")
	if !limiter.Blocked("k") {
		t.Fatal("expected blocked key")
	}

	limiter.Reset("k")
	if limiter.Blocked("k") {
		t.Fatal("still blocked after Reset")
	}
}

func TestLimiterWindowExpires(t *testing.T) {
	now := time.Now()
	limiter := NewLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	limiter.Fail("k")
	limiter.Fail("k")
	if !limiter.Blocked("k") {
		t.Fatal("expected blocked key inside the window")
	}

	now = now.Add(61 * time.Second)
	if limiter.Blocked("k") {
		t.Fatal("key remained blocked after the window elapsed")
	}
}
