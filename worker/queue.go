package worker

const (
	// DefaultQueueWorkerMaxWork is the maximum number of work items before queueing blocks.
	DefaultQueueWorkerMaxWork = 1 << 10
)

// NewQueue returns a new queue worker.
func NewQueue(action func(interface{}) error) *Queue {
	return &Queue{
		action:  action,
		latch:   &Latch{},
		maxWork: DefaultQueueWorkerMaxWork,
	}
}

// Queue is a worker that is pushed work over a channel.
type Queue struct {
	action  func(interface{}) error
	latch   *Latch
	errors  chan error
	work    chan interface{}
	maxWork int
}

// WithMaxWork sets the worker max work.
func (qw *Queue) WithMaxWork(maxWork int) *Queue {
	qw.maxWork = maxWork
	return qw
}

// MaxWork returns the maximum work.
func (qw *Queue) MaxWork() int {
	return qw.maxWork
}

// Latch returns the worker latch.
func (qw *Queue) Latch() *Latch {
	return qw.latch
}

// WithErrorCollector returns the error channel.
func (qw *Queue) WithErrorCollector(errors chan error) *Queue {
	qw.errors = errors
	return qw
}

// ErrorCollector returns a channel to read action errors from.
func (qw *Queue) ErrorCollector() chan error {
	return qw.errors
}

// Enqueue adds an item to the work queue.
func (qw *Queue) Enqueue(obj interface{}) {
	if qw.work == nil {
		return
	}
	qw.work <- obj
}

// Start starts the worker.
func (qw *Queue) Start() {
	qw.latch.Starting()
	if qw.maxWork > 0 {
		qw.work = make(chan interface{}, qw.maxWork)
	} else {
		qw.work = make(chan interface{})
	}

	go func() {
		qw.latch.Started()
		var err error
		var workItem interface{}
		for {
			select {
			case workItem = <-qw.work:
				err = qw.action(workItem)
				if err != nil && qw.errors != nil {
					qw.errors <- err
				}
			case <-qw.latch.NotifyStop():
				qw.latch.Stopped()
				return
			}
		}
	}()
	<-qw.latch.NotifyStarted()
}

// Stop stops the worker.
func (qw *Queue) Stop() {
	qw.latch.Stop()
	<-qw.latch.NotifyStopped()
}
