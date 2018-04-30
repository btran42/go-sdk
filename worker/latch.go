package worker

import (
	"sync"
	"sync/atomic"
)

const (
	latchStopped  int32 = 0
	latchStarting int32 = 1
	latchRunning  int32 = 2
	latchStopping int32 = 3
)

// Latch is a helper to coordinate killing goroutines.
// The lifecycle is generally as follows.
// 0 - stopped
// 1 - signal started
// 2 - running / started
// N-2 - signal stop
// N-1 - stopping
// goto 0
type Latch struct {
	sync.Mutex
	state int32

	started    chan struct{}
	shouldStop chan struct{}
	stopped    chan struct{}
}

// CanStart returns if the latch can start.
func (l *Latch) CanStart() bool {
	return atomic.LoadInt32(&l.state) == latchStopped
}

// CanStop returns if the latch can stop.
func (l *Latch) CanStop() bool {
	return atomic.LoadInt32(&l.state) == latchRunning
}

// IsStopped returns if the latch is stopped.
func (l *Latch) IsStopped() (isStopped bool) {
	return atomic.LoadInt32(&l.state) == latchStopped
}

// IsStarting indicates the latch is waiting to be scheduled.
func (l *Latch) IsStarting() bool {
	return atomic.LoadInt32(&l.state) == latchStarting
}

// IsRunning indicates we can signal to stop.
func (l *Latch) IsRunning() bool {
	return atomic.LoadInt32(&l.state) == latchRunning
}

// IsStopping returns if the latch is waiting to finish stopping.
func (l *Latch) IsStopping() bool {
	return atomic.LoadInt32(&l.state) == latchStopping
}

// NotifyStarted returns the started signal.
// It is used to coordinate the transition from starting -> started.
func (l *Latch) NotifyStarted() <-chan struct{} {
	return l.started
}

// NotifyStop returns the should stop signal.
// It is used to trigger the transition from running -> stopping -> stopped.
func (l *Latch) NotifyStop() <-chan struct{} {
	return l.shouldStop
}

// NotifyStopped returns the stopped signal.
// It is used to coordinate the transition from stopping -> stopped.
func (l *Latch) NotifyStopped() <-chan struct{} {
	return l.stopped
}

// Starting signals the latch is starting.
// This is typically done before you kick off a goroutine.
func (l *Latch) Starting() {
	l.Lock()
	atomic.StoreInt32(&l.state, latchStarting)
	l.started = make(chan struct{})
	l.Unlock()
}

// Started signals that the latch is started and has entered
// the `IsRunning` state.
func (l *Latch) Started() {
	if !l.IsStarting() {
		return
	}
	l.Lock()
	atomic.StoreInt32(&l.state, latchRunning)
	l.shouldStop = make(chan struct{})
	close(l.started)
	l.Unlock()
}

// Stop signals the latch to stop.
// It could also be thought of as `SignalStopping`.
func (l *Latch) Stop() {
	if !l.IsRunning() {
		return
	}
	l.Lock()
	atomic.StoreInt32(&l.state, latchStopping)
	l.stopped = make(chan struct{})
	close(l.shouldStop)
	l.Unlock()
}

// Stopped signals the latch has stopped.
func (l *Latch) Stopped() {
	if !l.IsStopping() {
		return
	}
	l.Lock()
	atomic.StoreInt32(&l.state, latchStopped)
	close(l.stopped)
	l.Unlock()
}
