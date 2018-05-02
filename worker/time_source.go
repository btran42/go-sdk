package worker

import (
	"sync"
	"time"
)

// TimeSource is a provider for time stuff.
type TimeSource interface {
	Now() time.Time
	Sleep(time.Duration)
	Tick(time.Duration) <-chan time.Time
}

// SystemTimeSource is the system time source.
type SystemTimeSource struct{}

// Now returns the current time.
func (sts SystemTimeSource) Now() time.Time {
	return time.Now().UTC()
}

// Sleep sleeps the current goroutine for the given duration.
func (sts SystemTimeSource) Sleep(d time.Duration) {
	time.Sleep(d)
}

// Tick returns a stdlib time.Ticker channel.
func (sts SystemTimeSource) Tick(d time.Duration) <-chan time.Time {
	return time.Tick(d)
}

// NewMockTimeSource returns a new mocked time source.
func NewMockTimeSource() *MockTimeSource {
	return &MockTimeSource{
		Current: time.Now().UTC(),
	}
}

// MockTimeSource is a mocked time source.
type MockTimeSource struct {
	sync.Mutex
	Current time.Time
	Tickers []*mockTicker
}

// Now returns the current mock time.
func (mts *MockTimeSource) Now() (current time.Time) {
	mts.Lock()
	current = mts.Current
	mts.Unlock()
	return
}

// Sleep advances the mock time by a given duration.
func (mts *MockTimeSource) Sleep(d time.Duration) {
	mts.Lock()
	mts.Current = mts.Current.Add(d)
	mts.Unlock()

	for _, mt := range mts.Tickers {
		mt.Advance(d)
	}
}

// Tick returns a mock ticker.
func (mts *MockTimeSource) Tick(d time.Duration) (tick <-chan time.Time) {
	mts.Lock()
	ticker := &mockTicker{Current: mts.Current, Interval: d, Tick: make(chan time.Time)}
	mts.Tickers = append(mts.Tickers, ticker)
	tick = ticker.Tick
	mts.Unlock()
	return
}

type mockTicker struct {
	sync.Mutex

	// Interval is the ticker interval.
	Interval time.Duration
	// Last is the last time the ticker fired.
	Last time.Time
	// Current is the current time as moved forward by advance.
	Current time.Time
	// Ticker is the channel to signal.
	Tick chan time.Time
}

// Advance fires the ticker for each `Interval` until the duration is satisfied.
func (mt *mockTicker) Advance(d time.Duration) {
	mt.Lock()

	if mt.Last.IsZero() {
		mt.Tick <- mt.Current
		mt.Last = mt.Current
	}

	next := mt.Last.Add(mt.Interval)
	end := mt.Current.Add(d)
	for next.Before(end) {
		mt.Last = next
		next = mt.Last.Add(mt.Interval)
		mt.Tick <- mt.Current
	}
	mt.Current = end
	mt.Unlock()
}
