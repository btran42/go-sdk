package worker

import "time"

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
	Current time.Time
	Tickers []*mockTicker
}

// Now returns the current mock time.
func (mts *MockTimeSource) Now() time.Time {
	return mts.Current
}

// Sleep advances the mock time by a given duration.
func (mts *MockTimeSource) Sleep(d time.Duration) {
	mts.Current = mts.Current.Add(d)
	for _, mt := range mts.Tickers {
		mt.Advance(d)
	}
}

// Tick implements a mock ticker.
func (mts *MockTimeSource) Tick(d time.Duration) <-chan time.Time {
	ticker := &mockTicker{Current: mts.Current, Interval: d, Tick: make(chan time.Time)}
	mts.Tickers = append(mts.Tickers, ticker)
	return ticker.Tick
}

type mockTicker struct {
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
}
