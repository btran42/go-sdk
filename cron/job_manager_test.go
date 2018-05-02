package cron

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"strings"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/logger"
	"github.com/blend/go-sdk/worker"
)

func TestRunTask(t *testing.T) {
	a := assert.New(t)

	wg := sync.WaitGroup{}
	wg.Add(1)
	var runCount int32
	New().RunTask(NewTask(func(ctx context.Context) error {
		defer wg.Done()
		atomic.AddInt32(&runCount, 1)
		return nil
	}))

	wg.Wait()
	a.Equal(1, runCount)
}

func TestRunTaskAndCancel(t *testing.T) {
	a := assert.New(t)

	ts := worker.NewMockTimeSource()
	start := ts.Now()
	jm := New().WithTimeSource(ts)
	didRun := make(chan struct{})
	didFinish := make(chan struct{})

	jm.RunTask(NewTaskWithName("taskToCancel", func(ctx context.Context) error {
		close(didRun)
		for {
			select {
			case <-ctx.Done():
				close(didFinish)
				return nil
			default:
				jm.TimeSource().Sleep(10 * time.Millisecond)
			}
		}
	}))
	<-didRun
	jm.CancelTask("taskToCancel")
	<-didFinish
	a.NotZero(ts.Now().Sub(start))
}

func TestRunJobBySchedule(t *testing.T) {
	a := assert.New(t)

	didRun := make(chan bool)

	jm := New().WithTimeSource(worker.NewMockTimeSource())
	runAt := jm.TimeSource().Now().Add(jm.HeartbeatInterval())

	err := jm.LoadJob(&runAtJob{
		RunAt: runAt,
		RunDelegate: func(ctx context.Context) error {
			didRun <- true
			return nil
		},
	})
	a.Nil(err)
	jm.Start()
	defer jm.Stop()

	jm.TimeSource().Sleep(2 * jm.HeartbeatInterval())
	before := jm.TimeSource().Now()
	<-didRun

	a.True(jm.TimeSource().Now().Sub(before) < 2*jm.HeartbeatInterval())
}

func TestDisableJob(t *testing.T) {
	a := assert.New(t)

	didRun := new(AtomicFlag)
	runCount := new(AtomicCounter)

	jm := New()
	err := jm.LoadJob(&runAtJob{
		RunDelegate: func(ctx context.Context) error {
			runCount.Increment()
			didRun.Set(true)
			return nil
		}})

	a.Nil(err)
	a.Nil(jm.DisableJob(runAtJobName))
	a.NotEmpty(jm.jobMetas)
	a.True(jm.IsDisabled(runAtJobName))
}

func TestSerialTask(t *testing.T) {
	assert := assert.New(t)

	runCount := new(AtomicCounter)
	ts := worker.NewMockTimeSource()
	jm := New().WithTimeSource(ts)
	task := NewSerialTaskWithName("test", func(ctx context.Context) error {
		runCount.Increment()
		jm.TimeSource().Sleep(10 * time.Millisecond)
		return nil
	})
	jm.RunTask(task)
	jm.RunTask(task)
	ts.Sleep(50 * time.Millisecond)
	assert.Equal(1, runCount.Get())

	// ensure parallel execution is still working as intended
	task = NewTaskWithName("test1", func(ctx context.Context) error {
		runCount.Increment()
		jm.TimeSource().Sleep(10 * time.Millisecond)
		return nil
	})
	runCount = new(AtomicCounter)
	jm.RunTask(task)
	jm.RunTask(task)
	ts.Sleep(50 * time.Millisecond)
	assert.Equal(2, runCount.Get())
}

func TestRunTaskAndCancelWithTimeout(t *testing.T) {
	a := assert.New(t)

	mt := worker.NewMockTimeSource()

	jm := New().WithTimeSource(mt)

	canceled := new(AtomicFlag)
	didCancel := new(AtomicFlag)
	cancelCount := new(AtomicCounter)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	a.Nil(jm.LoadJob(&testJobWithTimeout{
		TimeoutDuration: 250 * time.Millisecond,
		RunDelegate: func(ctx context.Context) error {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					canceled.Set(true)
					return nil
				default:
					jm.TimeSource().Sleep(10 * time.Millisecond)
					continue
				}
			}
		},
		CancellationDelegate: func() {
			cancelCount.Increment()
			didCancel.Set(true)
		},
	}))
	jm.Start()
	defer jm.Stop()

	mt.Sleep(500 * time.Millisecond)

	wg.Wait()

	a.True(didCancel.Get())
	a.True(canceled.Get())
}

func TestRunJobSimultaneously(t *testing.T) {
	a := assert.New(t)
	a.StartTimeout(2000 * time.Millisecond)
	defer a.EndTimeout()

	jm := New().WithHighPrecisionHeartbeat()

	wg := sync.WaitGroup{}
	wg.Add(2)

	jm.LoadJob(&runAtJob{
		RunAt: time.Now().UTC(),
		RunDelegate: func(ctx context.Context) error {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	})

	go func() {
		err := jm.RunJob(runAtJobName)
		a.Nil(err)
	}()
	go func() {
		err := jm.RunJob(runAtJobName)
		a.Nil(err)
	}()

	wg.Wait()
}

func TestJobManagerStartedListener(t *testing.T) {
	assert := assert.New(t)

	jm := New()

	wg := sync.WaitGroup{}
	wg.Add(2)

	output := bytes.NewBuffer(nil)
	agent := logger.New(FlagStarted, logger.Error).WithRecoverPanics(false).
		WithWriter(logger.NewTextWriter(output).
			WithUseColor(false).
			WithShowTimestamp(false))

	defer agent.Close()

	jm.WithLogger(agent)

	var didFireListener bool
	jm.Logger().Listen(FlagStarted, "foo", func(e logger.Event) {
		defer wg.Done()
		if typed, isTyped := e.(*Event); isTyped {
			assert.Equal(FlagStarted, e.Flag())
			assert.False(e.Timestamp().IsZero())
			assert.Equal("test_task", typed.TaskName())
			assert.Zero(typed.Elapsed())
			assert.Nil(typed.Err())
		}
		didFireListener = true
	})

	var didRun bool
	jm.RunTask(NewTaskWithName("test_task", func(ctx context.Context) error {
		defer wg.Done()
		didRun = true
		return nil
	}))
	wg.Wait()
	agent.Drain()

	assert.True(didRun)
	assert.True(didFireListener)
	assert.True(strings.Contains(output.String(), "[cron.started] `test_task`"), output.String())
}

func TestJobManagerCompleteListener(t *testing.T) {
	assert := assert.New(t)

	jm := New().WithTimeSource(worker.NewMockTimeSource())

	wg := sync.WaitGroup{}
	wg.Add(2)
	output := bytes.NewBuffer(nil)
	agent := logger.New(FlagComplete, logger.Error).WithWriter(
		logger.NewTextWriter(output).
			WithUseColor(false).
			WithShowTimestamp(false))

	defer agent.Close()

	jm.WithLogger(agent)

	var didFireListener bool
	jm.Logger().Listen(FlagComplete, "foo", func(e logger.Event) {
		defer wg.Done()
		if typed, isTyped := e.(*Event); isTyped {
			assert.Equal(FlagComplete, e.Flag())
			assert.False(e.Timestamp().IsZero())
			assert.Equal("test_task", typed.TaskName())
			assert.NotZero(typed.Elapsed())
			assert.Nil(typed.Err())
		}
		didFireListener = true
	})

	var didRun bool
	jm.RunTask(NewTaskWithName("test_task", func(ctx context.Context) error {
		defer wg.Done()
		didRun = true
		return nil
	}))
	jm.TimeSource().Sleep(time.Second)
	wg.Wait()
	agent.Drain()

	assert.True(didRun)
	assert.True(didFireListener)
	assert.Contains(output.String(), "[cron.complete] `test_task`")
}

func TestJobManagerCompleteListenerWithError(t *testing.T) {
	assert := assert.New(t)

	jm := New().WithTimeSource(worker.NewMockTimeSource())

	wg := sync.WaitGroup{}
	wg.Add(2)
	output := bytes.NewBuffer(nil)
	agent := logger.New(FlagComplete, logger.Error).WithWriter(
		logger.NewTextWriter(output).
			WithUseColor(false).
			WithShowTimestamp(false))
	defer agent.Close()

	jm.SetLogger(agent)
	var didFireListener bool
	jm.Logger().Listen(FlagComplete, "foo", func(e logger.Event) {
		defer wg.Done()
		if typed, isTyped := e.(*Event); isTyped {
			assert.Equal(FlagComplete, e.Flag())
			assert.False(e.Timestamp().IsZero())
			assert.Equal("test_task", typed.TaskName())
			assert.NotZero(typed.Elapsed())
			assert.NotNil(typed.Err())
		}
		didFireListener = true
	})

	var didRun bool
	jm.RunTask(NewTaskWithName("test_task", func(ctx context.Context) error {
		defer wg.Done()
		didRun = true
		return fmt.Errorf("testError")
	}))
	jm.TimeSource().Sleep(time.Second)
	wg.Wait()
	agent.Drain()

	assert.True(didRun)
	assert.True(didFireListener)
	assert.Contains(output.String(), "[cron.complete] `test_task`")
}

// The goal with this test is to see if panics take down the test process or not.
func TestJobManagerTaskPanicHandling(t *testing.T) {
	a := assert.New(t)

	manager := New().WithTimeSource(worker.NewMockTimeSource())
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)
	err := manager.RunTask(NewTask(func(ctx context.Context) error {
		defer waitGroup.Done()
		array := []int{}
		foo := array[1] //this should index out of bounds
		a.NotZero(foo)
		return nil
	}))

	waitGroup.Wait()
	a.Nil(err)
}

func TestEnabledProvider(t *testing.T) {
	a := assert.New(t)

	manager := New().WithTimeSource(worker.NewMockTimeSource())
	job := &testWithEnabled{
		isEnabled: true,
		action:    func() {},
	}

	name := "testWithEnabled"

	manager.LoadJob(job)
	a.False(manager.IsDisabled(name))
	a.Nil(manager.DisableJob(name))
	a.True(manager.IsDisabled(name))
	job.isEnabled = false
	a.True(manager.IsDisabled(name))
	a.Nil(manager.EnableJob(name))
	a.True(manager.IsDisabled(name))
}

func TestFiresErrorOnTaskError(t *testing.T) {
	a := assert.New(t)

	agent := logger.New(logger.Error)
	defer agent.Close()
	manager := New().WithTimeSource(worker.NewMockTimeSource())
	manager.SetLogger(agent)

	var errorDidFire bool
	var errorMatched bool
	wg := sync.WaitGroup{}
	wg.Add(2)
	agent.Listen(logger.Error, "foo", func(e logger.Event) {
		defer wg.Done()
		errorDidFire = true
		if typed, isTyped := e.(*logger.ErrorEvent); isTyped {
			if typed.Err() != nil {
				errorMatched = typed.Err().Error() == "this is only a test"
			}
		}
	})
	manager.LoadJob(NewJob().WithAction(func(ctx context.Context) error {
		defer wg.Done()
		return fmt.Errorf("this is only a test")
	}).WithName("error_test"))
	manager.RunJob("error_test")
	wg.Wait()

	a.True(errorDidFire)
	a.True(errorMatched)
}

const (
	runAtJobName = "runAt"
)

type runAtJob struct {
	RunAt       time.Time
	RunDelegate func(ctx context.Context) error
}

type runAt time.Time

func (ra runAt) GetNextRunTime(ns NowSource, after *time.Time) *time.Time {
	return Optional(time.Time(ra))
}

func (raj *runAtJob) Name() string {
	return "runAt"
}

func (raj *runAtJob) Schedule() Schedule {
	return runAt(raj.RunAt)
}

func (raj *runAtJob) Execute(ctx context.Context) error {
	return raj.RunDelegate(ctx)
}

type testJobWithTimeout struct {
	RunAt                time.Time
	TimeoutDuration      time.Duration
	RunDelegate          func(ctx context.Context) error
	CancellationDelegate func()
}

func (tj *testJobWithTimeout) Name() string {
	return "testJobWithTimeout"
}

func (tj *testJobWithTimeout) Timeout() time.Duration {
	return tj.TimeoutDuration
}

func (tj *testJobWithTimeout) Schedule() Schedule {
	if !tj.RunAt.IsZero() {
		return At(tj.RunAt)
	}
	return Immediately()
}

func (tj *testJobWithTimeout) Execute(ctx context.Context) error {
	return tj.RunDelegate(ctx)
}

func (tj *testJobWithTimeout) OnCancellation() {
	tj.CancellationDelegate()
}

type testJobInterval struct {
	RunEvery    time.Duration
	RunDelegate func(ctx context.Context) error
}

func (tj *testJobInterval) Name() string {
	return "testJobInterval"
}

func (tj *testJobInterval) Schedule() Schedule {
	return Every(tj.RunEvery)
}

func (tj *testJobInterval) Execute(ctx context.Context) error {
	return tj.RunDelegate(ctx)
}

type testWithEnabled struct {
	isEnabled bool
	action    func()
}

func (twe testWithEnabled) Name() string {
	return "testWithEnabled"
}

func (twe testWithEnabled) Schedule() Schedule {
	return OnDemand()
}

func (twe testWithEnabled) Enabled() bool {
	return twe.isEnabled
}

func (twe testWithEnabled) Execute(ctx context.Context) error {
	twe.action()
	return nil
}
