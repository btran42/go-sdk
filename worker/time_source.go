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

// MockTimeSource is a mocked time source.
type MockTimeSource struct {
	Current time.Time
	Ticker  <-chan time.Time
}

// Now returns the current mock time.
func (mts MockTimeSource) Now() time.Time {
	return mts.Current
}

// Sleep advances the mock time by a given duration.
func (mts MockTimeSource) Sleep(d time.Duration) {
	mts.Current = mts.Current.Add(d)
}

// Tick implements a mock ticker.
func (mts MockTimeSource) Tick(d time.Duration) <-chan time.Time {
	if mts.Ticker == nil {
		mts.Ticker = make(<-chan time.Time)
	}
	return mts.Ticker
}
