package cron

// NOTE: ALL TIMES ARE IN UTC. JUST USE UTC.

import (
	"context"
	"sync"
	"time"

	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/logger"
	"github.com/blend/go-sdk/worker"
)

// New returns a new job manager.
func New() *JobManager {
	jm := JobManager{
		heartbeatInterval: DefaultHeartbeatInterval,
		jobs:              map[string]Job{},
		jobMetas:          map[string]*JobMeta{},
		tasks:             map[string]*TaskMeta{},
		timeSource:        worker.SystemTimeSource{},
	}

	jm.schedulerWorker = worker.NewInterval(jm.runDueJobs, DefaultHeartbeatInterval).WithTimeSource(jm.timeSource)
	jm.killHangingTasksWorker = worker.NewInterval(jm.killHangingTasks, DefaultHeartbeatInterval).WithTimeSource(jm.timeSource)
	return &jm
}

// NewFromConfig returns a new job manager from a given config.
func NewFromConfig(cfg *Config) *JobManager {
	return New().WithHeartbeatInterval(cfg.GetHeartbeatInterval())
}

// NewFromEnv returns a new job manager from the environment.
func NewFromEnv() *JobManager {
	return NewFromConfig(NewConfigFromEnv())
}

// JobManager is the main orchestration and job management object.
type JobManager struct {
	sync.Mutex

	heartbeatInterval time.Duration
	log               *logger.Logger
	timeSource        worker.TimeSource

	schedulerWorker        *worker.Interval
	killHangingTasksWorker *worker.Interval

	jobs     map[string]Job
	jobMetas map[string]*JobMeta
	tasks    map[string]*TaskMeta
}

// Logger returns the diagnostics agent.
func (jm *JobManager) Logger() *logger.Logger {
	return jm.log
}

// WithLogger sets the logger and returns a reference to the job manager.
func (jm *JobManager) WithLogger(log *logger.Logger) *JobManager {
	jm.SetLogger(log)
	return jm
}

// SetLogger sets the logger.
func (jm *JobManager) SetLogger(log *logger.Logger) {
	jm.log = log
}

// WithTimeSource sets the time source.
func (jm *JobManager) WithTimeSource(ts worker.TimeSource) *JobManager {
	jm.timeSource = ts
	jm.killHangingTasksWorker.WithTimeSource(ts)
	jm.schedulerWorker.WithTimeSource(ts)
	return jm
}

// TimeSource returns the time source.
func (jm *JobManager) TimeSource() worker.TimeSource {
	return jm.timeSource
}

// WithHighPrecisionHeartbeat sets the heartbeat interval to the high precision interval and returns the job manager.
func (jm *JobManager) WithHighPrecisionHeartbeat() *JobManager {
	jm.SetHeartbeatInterval(DefaultHighPrecisionHeartbeatInterval)
	return jm
}

// WithDefaultHeartbeat sets the heartbeat interval to the default interval and returns the job manager.
func (jm *JobManager) WithDefaultHeartbeat() *JobManager {
	jm.SetHeartbeatInterval(DefaultHeartbeatInterval)
	return jm
}

// WithHeartbeatInterval sets the heartbeat interval explicitly and returns the job manager.
func (jm *JobManager) WithHeartbeatInterval(interval time.Duration) *JobManager {
	jm.SetHeartbeatInterval(interval)
	return jm
}

// SetHeartbeatInterval sets the heartbeat interval explicitly.
func (jm *JobManager) SetHeartbeatInterval(interval time.Duration) {
	jm.schedulerWorker.WithInterval(interval)
	jm.killHangingTasksWorker.WithInterval(interval)
	jm.heartbeatInterval = interval
}

// HeartbeatInterval returns the current heartbeat interval.
func (jm *JobManager) HeartbeatInterval() time.Duration {
	return jm.heartbeatInterval
}

// ----------------------------------------------------------------------------
// Informational Methods
// ----------------------------------------------------------------------------

// HasJob returns if a jobName is loaded or not.
func (jm *JobManager) HasJob(jobName string) (hasJob bool) {
	jm.Lock()
	_, hasJob = jm.jobs[jobName]
	jm.Unlock()
	return
}

// Job returns a job instance by name.
func (jm *JobManager) Job(jobName string) (job Job) {
	jm.Lock()
	job, _ = jm.jobs[jobName]
	jm.Unlock()
	return
}

// IsDisabled returns if a job is disabled.
func (jm *JobManager) IsDisabled(jobName string) (value bool) {
	jm.Lock()
	defer jm.Unlock()

	if job, hasJob := jm.jobMetas[jobName]; hasJob {
		value = job.Disabled
		if job.EnabledProvider != nil {
			value = value || !job.EnabledProvider()
		}
	}
	return
}

// IsRunning returns if a task is currently running.
func (jm *JobManager) IsRunning(taskName string) (isRunning bool) {
	jm.Lock()
	_, isRunning = jm.tasks[taskName]
	jm.Unlock()
	return
}

// ReadAllJobs allows the consumer to do something with the full job list, using a read lock.
func (jm *JobManager) ReadAllJobs(action func(jobs map[string]Job)) {
	jm.Lock()
	defer jm.Unlock()

	action(jm.jobs)
}

// --------------------------------------------------------------------------------
// Core Methods
// --------------------------------------------------------------------------------

// LoadJobs loads a variadic list of jobs.
func (jm *JobManager) LoadJobs(jobs ...Job) error {
	jm.Lock()
	defer jm.Unlock()

	var err error
	for _, job := range jobs {
		err = jm.loadJobUnsafe(job)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadJob loads a job.
func (jm *JobManager) LoadJob(job Job) error {
	jm.Lock()
	defer jm.Unlock()

	return jm.loadJobUnsafe(job)
}

// DisableJobs disables a variadic list of job names.
func (jm *JobManager) DisableJobs(jobNames ...string) error {
	jm.Lock()
	defer jm.Unlock()

	var err error
	for _, jobName := range jobNames {
		err = jm.setJobDisabledUnsafe(jobName, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// DisableJob stops a job from running but does not unload it.
func (jm *JobManager) DisableJob(jobName string) error {
	jm.Lock()
	defer jm.Unlock()

	return jm.setJobDisabledUnsafe(jobName, true)
}

// EnableJobs enables a variadic list of job names.
func (jm *JobManager) EnableJobs(jobNames ...string) error {
	var err error
	for _, jobName := range jobNames {
		err = jm.setJobDisabledUnsafe(jobName, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// EnableJob enables a job that has been disabled.
func (jm *JobManager) EnableJob(jobName string) error {
	jm.Lock()
	defer jm.Unlock()

	return jm.setJobDisabledUnsafe(jobName, false)
}

// RunJobs runs a variadic list of job names.
func (jm *JobManager) RunJobs(jobNames ...string) error {
	jm.Lock()
	defer jm.Unlock()

	for _, jobName := range jobNames {
		if job, hasJob := jm.jobs[jobName]; hasJob {
			if !jm.IsDisabled(jobName) {
				jobErr := jm.runTaskUnsafe(job)
				if jobErr != nil {
					return jobErr
				}
			}
		} else {
			return exception.NewFromErr(ErrJobNotLoaded).WithMessagef("job: %s", jobName)
		}
	}
	return nil
}

// RunJob runs a job by jobName on demand.
func (jm *JobManager) RunJob(jobName string) error {
	jm.Lock()
	defer jm.Unlock()

	if job, hasJob := jm.jobs[jobName]; hasJob {
		if jm.shouldRunJob(job) {
			jm.jobMetas[jobName].LastRunTime = jm.timeSource.Now()
			err := jm.runTaskUnsafe(job)
			return err
		}
		return nil
	}
	return exception.NewFromErr(ErrJobNotLoaded).WithMessagef("job: %s", jobName)
}

// RunAllJobs runs every job that has been loaded in the JobManager at once.
func (jm *JobManager) RunAllJobs() error {
	jm.Lock()
	defer jm.Unlock()

	for _, job := range jm.jobs {
		if !jm.IsDisabled(job.Name()) {
			jobErr := jm.runTaskUnsafe(job)
			if jobErr != nil {
				return jobErr
			}
		}
	}
	return nil
}

// RunTask runs a task.
func (jm *JobManager) RunTask(task Task) error {
	jm.Lock()
	defer jm.Unlock()

	return jm.runTaskUnsafe(task)
}

// CancelTask cancels (sends the cancellation signal) to a running task.
func (jm *JobManager) CancelTask(taskName string) (err error) {
	jm.Lock()
	defer jm.Unlock()

	if task, hasTask := jm.tasks[taskName]; hasTask {
		jm.onTaskCancellation(task.Task)
		task.Cancel()
	} else {
		err = exception.NewFromErr(ErrTaskNotFound).WithMessagef("task: %s", taskName)
	}
	return
}

// Start begins the schedule runner for a JobManager.
func (jm *JobManager) Start() {
	jm.Lock()
	defer jm.Unlock()

	jm.schedulerWorker.Start()
	jm.killHangingTasksWorker.Start()

}

// Stop stops the schedule runner for a JobManager.
func (jm *JobManager) Stop() {
	jm.Lock()
	defer jm.Unlock()

	jm.schedulerWorker.Stop()
	jm.killHangingTasksWorker.Stop()
}

// --------------------------------------------------------------------------------
// lifecycle methods
// --------------------------------------------------------------------------------

func (jm *JobManager) runDueJobs() error {
	jm.Lock()
	defer jm.Unlock()

	now := jm.timeSource.Now()
	var taskErr error
	var jobName string
	for _, job := range jm.jobs {
		jobName = job.Name()
		meta := jm.jobMetas[jobName]
		nextRunTime := meta.NextRunTime
		if !nextRunTime.IsZero() {
			if jm.shouldRunJob(job) {
				if nextRunTime.Before(now) {
					newNext := Deref(meta.Schedule.GetNextRunTime(&now))
					jm.jobMetas[jobName].NextRunTime = newNext
					jm.jobMetas[jobName].LastRunTime = now
					jm.debugf("scheduling %s for %v from now", jobName, newNext.Sub(now))
					if taskErr = jm.runTaskUnsafe(job); taskErr != nil {
						jm.log.Error(taskErr)
					}
				}
			}
		}
	}
	return nil
}

// RunTask runs a task on demand.
func (jm *JobManager) runTaskUnsafe(t Task) error {
	if !jm.shouldRunTask(t) {
		return nil
	}

	taskName := t.Name()
	start := jm.timeSource.Now()
	ctx, cancel := jm.createContext()

	tm := &TaskMeta{
		Name:    taskName,
		Task:    t,
		Context: ctx,
		Cancel:  cancel,
	}

	if typed, isTyped := t.(TimeoutProvider); isTyped {
		tm.Timeout = start.Add(typed.Timeout())
	}
	jm.tasks[taskName] = tm

	started := make(chan struct{})
	go func() {
		close(started)
		var err error
		defer func() {
			if r := recover(); r != nil {
				err = exception.Newf("%v", r)
			}

			jm.Lock()
			delete(jm.tasks, taskName)
			jm.Unlock()

			jm.fireTaskCompleteListeners(taskName, jm.timeSource.Now().Sub(start), err)
		}()

		jm.onTaskStart(t)
		jm.fireTaskStartedListeners(taskName)
		err = t.Execute(ctx)
		jm.onTaskComplete(t, err)
	}()
	<-started

	return nil
}

func (jm *JobManager) killHangingTasks() (err error) {
	jm.Lock()
	defer jm.Unlock()

	for _, taskMeta := range jm.tasks {
		taskName := taskMeta.Name
		if taskMeta.Timeout.IsZero() {
			return
		}

		now := jm.timeSource.Now()
		if jobMeta, hasJobMeta := jm.jobMetas[taskName]; hasJobMeta {
			nextRuntime := jobMeta.NextRunTime

			// we need to calculate the effective timeout
			// either startedTime+timeout or the next runtime, whichever is closer.

			// t1 represents the absolute timeout time.
			t1 := taskMeta.Timeout
			// t2 represents the next runtime, or an effective time we need to stop by.
			t2 := nextRuntime

			// the effective timeout is whichever is more soon.
			effectiveTimeout := Min(t1, t2)

			// if the effective timeout is in the past, or it's within the next heartbeat.
			if now.After(effectiveTimeout) || effectiveTimeout.Sub(now) < jm.heartbeatInterval {
				err = jm.killHangingJob(taskName)
				if err != nil {
					jm.log.Error(err)
				}
			}
		} else if taskMeta.Timeout.Before(now) {
			err = jm.killHangingJob(taskName)
			if err != nil {
				jm.log.Error(err)
			}
		}
	}
	return nil
}

// killHangingJob cancels (sends the cancellation signal) to a running task that has exceeded its timeout.
// it assumes that the following locks are held:
// - runningTasksLock
// - runningTaskStartTimesLock
// - contextsLock
// otherwise, chaos, mayhem, deadlocks. You should *rarely* need to call this explicitly.
func (jm *JobManager) killHangingJob(taskName string) error {
	task, hasTask := jm.tasks[taskName]
	if !hasTask {
		return exception.Newf("task not found").WithMessagef("Task: %s", taskName)
	}

	task.Cancel()
	jm.onTaskCancellation(task.Task)
	delete(jm.tasks, task.Name)
	return nil
}

// --------------------------------------------------------------------------------
// Utility Methods
// --------------------------------------------------------------------------------

// LoadJob adds a job to the manager.
func (jm *JobManager) loadJobUnsafe(j Job) error {
	jobName := j.Name()

	if _, hasJob := jm.jobs[jobName]; hasJob {
		return exception.NewFromErr(ErrJobAlreadyLoaded).WithMessagef("job: %s", j.Name())
	}

	schedule := j.Schedule()
	jm.jobs[jobName] = j
	jm.jobMetas[jobName] = &JobMeta{
		Name:        jobName,
		NextRunTime: Deref(schedule.GetNextRunTime(nil)),
		Schedule:    schedule,
	}
	return nil
}

func (jm *JobManager) setJobDisabledUnsafe(jobName string, disabled bool) error {
	if _, hasJob := jm.jobs[jobName]; !hasJob {
		return exception.NewFromErr(ErrJobNotLoaded).WithMessagef("job: %s", jobName)
	}

	jm.jobMetas[jobName].Disabled = disabled
	return nil
}

func (jm *JobManager) createContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// shouldRunJob returns whether it is legal to run a job based off of a job's attributes and status.
// Use this function to set logic for whether a job should run
func (jm *JobManager) shouldRunJob(job Job) bool {
	if meta, hasMeta := jm.jobMetas[job.Name()]; hasMeta {
		return !meta.Disabled
	}
	return false
}

// shouldRunTask returns whether a task should be executed based on its status
func (jm *JobManager) shouldRunTask(t Task) bool {
	_, serial := t.(SerialProvider)
	if serial {
		_, hasTask := jm.tasks[t.Name()]
		return !hasTask
	}
	return true
}

func (jm *JobManager) onTaskStart(t Task) {
	if receiver, isReceiver := t.(OnStartReceiver); isReceiver {
		receiver.OnStart()
	}
}

func (jm *JobManager) onTaskComplete(t Task, result error) {
	if receiver, isReceiver := t.(OnCompleteReceiver); isReceiver {
		receiver.OnComplete(result)
	}
}

func (jm *JobManager) onTaskCancellation(t Task) {
	if receiver, isReceiver := t.(OnCancellationReceiver); isReceiver {
		receiver.OnCancellation()
	}
}

// ShouldTriggerListeners is a helper function to determine if we should trigger listeners for a given task.
func (jm *JobManager) shouldTriggerListeners(jobName string) bool {
	if job, hasJob := jm.jobs[jobName]; hasJob {
		if typed, isTyped := job.(EventTriggerListenersProvider); isTyped {
			return typed.ShouldTriggerListeners()
		}
	}

	return true
}

// ShouldWriteOutput is a helper function to determine if we should write logging output for a task.
func (jm *JobManager) shouldWriteOutput(jobName string) bool {
	if job, hasJob := jm.jobs[jobName]; hasJob {
		if typed, isTyped := job.(EventShouldWriteOutputProvider); isTyped {
			return typed.ShouldWriteOutput()
		}
	}
	return true
}

// fireTaskStartedListeners fires the currently configured task listeners.
func (jm *JobManager) fireTaskStartedListeners(taskName string) {
	if jm.log == nil {
		return
	}
	jm.log.Trigger(NewEvent(FlagStarted, taskName).WithIsEnabled(jm.shouldTriggerListeners(taskName)).WithIsWritable(jm.shouldWriteOutput(taskName)))
}

// fireTaskListeners fires the currently configured task listeners.
func (jm *JobManager) fireTaskCompleteListeners(taskName string, elapsed time.Duration, err error) {
	if jm.log == nil {
		return
	}

	jm.log.Trigger(NewEvent(FlagComplete, taskName).
		WithIsEnabled(jm.shouldTriggerListeners(taskName)).
		WithIsWritable(jm.shouldWriteOutput(taskName)).
		WithElapsed(elapsed).
		WithErr(err))

	if err != nil {
		jm.log.Error(err)
	}
}

func (jm *JobManager) errorf(format string, args ...interface{}) {
	if jm.log != nil {
		jm.log.SyncErrorf(format, args...)
	}
}

func (jm *JobManager) debugf(format string, args ...interface{}) {
	if jm.log != nil {
		jm.log.SyncDebugf(format, args...)
	}
}
