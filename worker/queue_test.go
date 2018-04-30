package worker

import (
	"sync"
	"testing"

	"github.com/blend/go-sdk/assert"
)

func TestQueueWorker(t *testing.T) {
	assert := assert.New(t)

	var didWork bool
	wg := sync.WaitGroup{}
	wg.Add(1)
	w := NewQueue(func(obj interface{}) error {
		defer wg.Done()
		didWork = true
		assert.Equal("hello", obj)
		return nil
	})

	w.Start()
	assert.True(w.Latch().IsRunning())
	w.Enqueue("hello")
	wg.Wait()
	w.Stop()
	assert.False(w.Latch().IsRunning())

	assert.True(didWork)
}
