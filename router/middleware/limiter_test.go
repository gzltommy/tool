package middleware

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestLimiter(t *testing.T) {
	limiter := rate.NewLimiter(1, 2)
	num := 0
	for i := 0; i < 20; i++ {
		if limiter.Allow() {
			t.Log("allow")
		} else {
			t.Log("---------")
		}
		num++
		if num%5 == 0 {
			time.Sleep(time.Second * 1)
		}
	}
}
