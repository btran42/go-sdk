package worker

import (
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestIntervalWorker(t *testing.T) {
	assert := assert.New(t)

	var didWork bool
	wg := sync.WaitGroup{}
	wg.Add(1)
	w := NewInterval(func() error {
		defer wg.Done()
		didWork = true
		return nil
	}, time.Millisecond)

	w.Start()
	assert.True(w.Latch().IsRunning())
	wg.Wait()
	w.Stop()
	assert.True(w.Latch().IsStopped())

	assert.True(didWork)
}

func TestIntervalWorkerWithMockedTimeSource(t *testing.T) {
	assert := assert.New(t)
	assert.StartTimeout(time.Millisecond)
	defer assert.EndTimeout()

	mts := NewMockTimeSource()

	var count int
	iw := NewInterval(func() error {
		count = count + 1
		return nil
	}, time.Millisecond).WithTimeSource(mts)

	iw.Start()
	mts.Sleep(50 * time.Millisecond)
	iw.Stop()
	assert.Equal(50, count)
}
