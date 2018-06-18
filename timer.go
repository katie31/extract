package walg

import (
	"math"
	"math/rand"
	"time"
)

// ExponentialTicker is used for exponential backoff
// for uploading to S3. If the max wait time is reached,
// retries will occur after max wait time intervals up to
// max retries.
type ExponentialTicker struct {
	MaxRetries int
	retries    int
	MaxWait    float64
	wait       float64
}

// NewExpTicker creates a new ExponentialTicker with
// configurable max number of retries and max wait time.
func NewExpTicker(retries int, wait float64) *ExponentialTicker {
	return &ExponentialTicker{
		MaxRetries: retries,
		MaxWait:    wait,
	}
}

// Update increases running count of retries by 1 and
// exponentially increases the wait time until the
// max wait time is reached.
func (ticker *ExponentialTicker) Update() {
	if ticker.wait < ticker.MaxWait {
		rand.Seed(time.Now().UTC().UnixNano())
		ticker.wait = math.Exp2(float64(ticker.retries)) + rand.Float64()
	}
	ticker.retries++
}

// Sleep will wait in seconds.
func (ticker *ExponentialTicker) Sleep() {
	time.Sleep(time.Duration(ticker.wait) * time.Second)
}
