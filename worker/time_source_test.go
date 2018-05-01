package worker

import (
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestMockTicker(t *testing.T) {
	assert := assert.New(t)

	now := time.Now().UTC()
	mt := &mockTicker{
		Current:  now,
		Interval: 250 * time.Millisecond,
		Tick:     make(chan time.Time),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	stop := make(chan struct{})

	var count int
	go func() {
		defer wg.Done()

		for {
			select {
			case <-mt.Tick:
				count = count + 1
			case <-stop:
				return
			}
		}
	}()

	mt.Advance(time.Second)
	close(stop)
	wg.Wait()
	assert.Equal(4, count)
}
