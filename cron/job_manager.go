package cron

// NOTE: ALL TIMES ARE IN UTC. JUST USE UTC.

import (
	"context"
	"time"

	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/logger"
	"github.com/blend/go-sdk/worker"
)

// New returns a new job manager.
func New() *JobManager {
	jm := JobManager{
		heartbeatInterval: DefaultHeartbeatInterval,
		jobs:              &SyncJobMap{},
		jobMetas:          &SyncJobMetaMap{},
		tasks:             &SyncTaskMetaMap{},
		timeSource:        worker.SystemTimeSource{},
	}

	jm.schedulerWorker = worker.NewInterval(jm.runDueJobs, DefaultHeartbeatInterval)
	jm.killHangingTasksWorker = worker.NewInterval(jm.killHangingTasks, DefaultHeartbeatInterval)
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
	heartbeatInterval time.Duration
	log               *logger.Logger
	timeSource        worker.TimeSource

	schedulerWorker        *worker.Interval
	killHangingTasksWorker *worker.Interval

	jobs     *SyncJobMap
	jobMetas *SyncJobMetaMap
	tasks    *SyncTaskMetaMap
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

// WithHighPrecisionHeartbeat sets the heartbeat interval to the high precision interval and returns a reference.
func (jm *JobManager) WithHighPrecisionHeartbeat() *JobManager {
	jm.heartbeatInterval = DefaultHighPrecisionHeartbeatInterval
	return jm
}

// WithDefaultPrecisionHeartbeat sets the heartbeat interval to the high precision interval and returns a reference.
func (jm *JobManager) WithDefaultPrecisionHeartbeat() *JobManager {
	jm.heartbeatInterval = DefaultHeartbeatInterval
	return jm
}

// WithHeartbeatInterval sets the heartbeat interval explicitly and returns the job manager.
func (jm *JobManager) WithHeartbeatInterval(interval time.Duration) *JobManager {
	jm.SetHeartbeatInterval(interval)
	return jm
}

// SetHeartbeatInterval sets the heartbeat interval explicitly.
func (jm *JobManager) SetHeartbeatInterval(interval time.Duration) {
	jm.heartbeatInterval = interval
}

// HeartbeatInterval returns the current heartbeat interval.
func (jm *JobManager) HeartbeatInterval() time.Duration {
	return jm.heartbeatInterval
}

// ShouldTriggerListeners is a helper function to determine if we should trigger listeners for a given task.
func (jm *JobManager) ShouldTriggerListeners(taskName string) bool {
	if job, hasJob := jm.jobs.Get(taskName); hasJob {
		if typed, isTyped := job.(EventTriggerListenersProvider); isTyped {
			return typed.ShouldTriggerListeners()
		}
	}

	return true
}

// ShouldWriteOutput is a helper function to determine if we should write logging output for a task.
func (jm *JobManager) ShouldWriteOutput(taskName string) bool {
	if job, hasJob := jm.jobs.Get(taskName); hasJob {
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
	jm.log.Trigger(NewEvent(FlagStarted, taskName).WithIsEnabled(jm.ShouldTriggerListeners(taskName)).WithIsWritable(jm.ShouldWriteOutput(taskName)))
}

// fireTaskListeners fires the currently configured task listeners.
func (jm *JobManager) fireTaskCompleteListeners(taskName string, elapsed time.Duration, err error) {
	if jm.log == nil {
		return
	}

	jm.log.Trigger(NewEvent(FlagComplete, taskName).
		WithIsEnabled(jm.ShouldTriggerListeners(taskName)).
		WithIsWritable(jm.ShouldWriteOutput(taskName)).
		WithElapsed(elapsed).
		WithErr(err))

	if err != nil {
		jm.log.Error(err)
	}
}

// shouldRunJob returns whether it is legal to run a job based off of a job's attributes and status.
// Use this function to set logic for whether a job should run
func (jm *JobManager) shouldRunJob(job Job) bool {
	return !jm.IsDisabled(job.Name())
}

// ----------------------------------------------------------------------------
// Informational Methods
// ----------------------------------------------------------------------------

// HasJob returns if a jobName is loaded or not.
func (jm *JobManager) HasJob(jobName string) (hasJob bool) {
	hasJob = jm.jobs.Contains(jobName)
	return
}

// Job returns a job instance by name.
func (jm *JobManager) Job(jobName string) (job Job) {
	job, _ = jm.jobs.Get(jobName)
	return
}

// IsDisabled returns if a job is disabled.
func (jm *JobManager) IsDisabled(jobName string) (value bool) {
	if job, hasJob := jm.jobMetas.Get(jobName); hasJob {
		value = job.Disabled
		if job.EnabledProvider != nil {
			value = value || !job.EnabledProvider()
		}
	}
	return
}

// IsRunning returns if a task is currently running.
func (jm *JobManager) IsRunning(taskName string) (isRunning bool) {
	isRunning = jm.tasks.Contains(taskName)
	return
}

// ReadAllJobs allows the consumer to do something with the full job list, using a read lock.
func (jm *JobManager) ReadAllJobs(action func(jobs []Job)) {
	action(jm.jobs.All())
}

// --------------------------------------------------------------------------------
// Core Methods
// --------------------------------------------------------------------------------

// LoadJobs loads a variadic list of jobs.
func (jm *JobManager) LoadJobs(jobs ...Job) error {
	var err error
	for _, job := range jobs {
		err = jm.LoadJob(job)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadJob adds a job to the manager.
func (jm *JobManager) LoadJob(j Job) error {
	jobName := j.Name()

	if jm.HasJob(jobName) {
		return exception.NewFromErr(ErrJobAlreadyLoaded).WithMessagef("job: %s", j.Name())
	}

	jm.setJob(j)
	jm.setNextRunTime(jobName, j.Schedule().GetNextRunTime(nil))
	return nil
}

// DisableJobs disables a variadic list of job names.
func (jm *JobManager) DisableJobs(jobNames ...string) error {
	var err error
	for _, jobName := range jobNames {
		err = jm.DisableJob(jobName)
		if err != nil {
			return err
		}
	}
	return nil
}

// DisableJob stops a job from running but does not unload it.
func (jm *JobManager) DisableJob(jobName string) error {
	if !jm.HasJob(jobName) {
		return exception.NewFromErr(ErrJobNotLoaded).WithMessagef("job: %s", jobName)
	}

	jm.setJobDisabled(jobName)
	return nil
}

// EnableJobs enables a variadic list of job names.
func (jm *JobManager) EnableJobs(jobNames ...string) error {
	var err error
	for _, jobName := range jobNames {
		err = jm.EnableJob(jobName)
		if err != nil {
			return err
		}
	}
	return nil
}

// EnableJob enables a job that has been disabled.
func (jm *JobManager) EnableJob(jobName string) error {
	if !jm.HasJob(jobName) {
		return exception.NewFromErr(ErrJobNotLoaded).WithMessagef("job: %s", jobName)
	}

	jm.setJobEnabled(jobName)
	job := jm.getJob(jobName)
	jobSchedule := job.Schedule()
	jm.setNextRunTime(jobName, jobSchedule.GetNextRunTime(nil))

	return nil
}

// RunJobs runs a variadic list of job names.
func (jm *JobManager) RunJobs(jobNames ...string) error {
	for _, jobName := range jobNames {
		if job, hasJob := jm.jobs.Get(jobName); hasJob {
			if !jm.IsDisabled(jobName) {
				jobErr := jm.RunTask(job)
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
	if job, hasJob := jm.jobs.Get(jobName); hasJob {
		if jm.shouldRunJob(job) {
			now := time.Now().UTC()
			jm.setLastRunTime(jobName, now)
			err := jm.RunTask(job)
			return err
		}
		return nil
	}
	return exception.NewFromErr(ErrJobNotLoaded).WithMessagef("job: %s", jobName)
}

// RunAllJobs runs every job that has been loaded in the JobManager at once.
func (jm *JobManager) RunAllJobs() error {
	for _, job := range jm.jobs.All() {
		if !jm.IsDisabled(job.Name()) {
			jobErr := jm.RunTask(job)
			if jobErr != nil {
				return jobErr
			}
		}
	}
	return nil
}

// RunTask runs a task on demand.
func (jm *JobManager) RunTask(t Task) error {
	if !jm.shouldRunTask(t) {
		return nil
	}

	taskName := t.Name()
	start := Now()
	ctx, cancel := jm.createContext()

	jm.setRunningTask(t)
	jm.setContext(ctx, taskName)
	jm.setCancelFunc(taskName, cancel)

	// this is the main goroutine that runs the task
	go func() {
		var err error

		defer func() {
			jm.cleanupTask(taskName)
			jm.fireTaskCompleteListeners(taskName, Since(start), err)
		}()

		// panic recovery
		defer func() {
			if r := recover(); r != nil {
				err = exception.Newf("%v", r)
			}
		}()

		jm.onTaskStart(t)
		jm.fireTaskStartedListeners(taskName)
		err = t.Execute(ctx)
		jm.onTaskComplete(t, err)
	}()

	return nil
}

// CancelTask cancels (sends the cancellation signal) to a running task.
func (jm *JobManager) CancelTask(taskName string) (err error) {
	if task, hasTask := jm.tasks.Get(taskName); hasTask {
		cancel := jm.getCancelFunc(taskName)
		jm.onTaskCancellation(task.Task)
		cancel()
	} else {
		err = exception.NewFromErr(ErrTaskNotFound).WithMessagef("task: %s", taskName)
	}
	return
}

// Start begins the schedule runner for a JobManager.
func (jm *JobManager) Start() {
	jm.schedulerWorker.Start()
	jm.killHangingTasksWorker.Start()

}

// Stop stops the schedule runner for a JobManager.
func (jm *JobManager) Stop() {
	jm.schedulerWorker.Stop()
	jm.killHangingTasksWorker.Stop()
}

// --------------------------------------------------------------------------------
// Utility Methods
// --------------------------------------------------------------------------------

func (jm *JobManager) createContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// shouldRunTask returns whether a task should be executed based on its status
func (jm *JobManager) shouldRunTask(t Task) bool {
	_, serial := t.(SerialProvider)
	if serial {
		return !jm.IsRunning(t.Name())
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

func (jm *JobManager) cleanupTask(taskName string) {
	jm.deleteTask(taskName)
}

func (jm *JobManager) runDueJobs() error {
	now := jm.timeSource.Now()

	var taskErr error
	var jobName string
	var nextRunTime *time.Time
	for _, job := range jm.jobs.All() {
		jobName = job.Name()
		nextRunTime = jm.getNextRunTime(jobName)
		if nextRunTime != nil {
			if jm.shouldRunJob(job) {
				if nextRunTime.Before(now) {
					jm.setNextRunTime(jobName, jm.getSchedule(jobName).GetNextRunTime(&now))
					jm.setLastRunTime(jobName, now)
					if taskErr = jm.RunTask(job); taskErr != nil {
						jm.log.Error(taskErr)
					}
				}
			}
		}
	}
	return nil
}

func (jm *JobManager) killHangingTasks() (err error) {
	for _, taskMeta := range jm.tasks.All() {
		taskName := taskMeta.Name
		if taskMeta.Timeout.IsZero() {
			return
		}
		currentTime := jm.timeSource.Now()
		if hasJobMeta := jm.jobMetas.Contains(taskName); hasJobMeta {
			nextRuntime := jm.getNextRunTime(taskName)

			// we need to calculate the effective timeout
			// either startedTime+timeout or the next runtime, whichever is closer.

			// t1 represents the absolute timeout time.
			t1 := taskMeta.Timeout
			// t2 represents the next runtime, or an effective time we need to stop by.
			t2 := *nextRuntime

			// the effective timeout is whichever is more soon.
			effectiveTimeout := Min(t1, t2)

			// if the effective timeout is in the past, or it's within the next heartbeat.
			if currentTime.After(effectiveTimeout) || effectiveTimeout.Sub(currentTime) < jm.heartbeatInterval {
				err = jm.killHangingJob(taskName)
				if err != nil {
					jm.log.Error(err)
				}
			}
		} else if taskMeta.Timeout.Before(currentTime) {
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
	hasTask := jm.tasks.Contains(taskName)
	if !hasTask {
		return exception.Newf("task not found").WithMessagef("Task: %s", taskName)
	}

	meta, hasMeta := jm.tasks.Get(taskName)
	if !hasMeta {
		return nil
	}

	meta.Cancel()
	jm.onTaskCancellation(meta.Task)
	jm.tasks.Delete(taskName)

	return nil
}

// --------------------------------------------------------------------------------
// Atomic Access/Mutating Methods
// --------------------------------------------------------------------------------

func (jm *JobManager) deleteJob(jobName string) {
	jm.jobs.Delete(jobName)
}

func (jm *JobManager) deleteJobMeta(jobName string) {
	jm.jobMetas.Delete(jobName)
}

func (jm *JobManager) deleteTask(taskName string) {
	jm.tasks.Delete(taskName)
}

func (jm *JobManager) getContext(taskName string) (ctx context.Context) {
	if t, has := jm.tasks.Get(taskName); has {
		ctx = t.Context
	}
	return
}

// note; this setter is out of order because of the linter.
func (jm *JobManager) setContext(ctx context.Context, taskName string) {
	jm.tasks.Do(taskName, func(t *TaskMeta) {
		t.Context = ctx
	})
}

func (jm *JobManager) getCancelFunc(taskName string) (cancel context.CancelFunc) {
	if t, has := jm.tasks.Get(taskName); has {
		cancel = t.Cancel
	}
	return
}

func (jm *JobManager) setCancelFunc(taskName string, cancel context.CancelFunc) {
	jm.tasks.Do(taskName, func(t *TaskMeta) {
		t.Cancel = cancel
	})
}

func (jm *JobManager) setJobDisabled(jobName string) {
	jm.jobMetas.Do(jobName, func(j *JobMeta) {
		j.Disabled = true
	})
}

func (jm *JobManager) setJobEnabled(jobName string) {
	jm.jobMetas.Do(jobName, func(j *JobMeta) {
		j.Disabled = false
	})
}

func (jm *JobManager) getNextRunTime(jobName string) (nextRunTime *time.Time) {
	if j, has := jm.jobMetas.Get(jobName); has {
		nextRunTime = j.NextRunTime
	}
	return
}

func (jm *JobManager) setNextRunTime(jobName string, t *time.Time) {
	jm.jobMetas.Do(jobName, func(j *JobMeta) {
		j.NextRunTime = t
	})
}

func (jm *JobManager) getLastRunTime(jobName string) (lastRunTime *time.Time) {
	if j, has := jm.jobMetas.Get(jobName); has {
		lastRunTime = j.LastRunTime
	}
	return
}

func (jm *JobManager) setLastRunTime(jobName string, t time.Time) {
	jm.jobMetas.Do(jobName, func(j *JobMeta) {
		j.LastRunTime = &t
	})
}

func (jm *JobManager) getJob(jobName string) (job Job) {
	job, _ = jm.jobs.Get(jobName)
	return
}

func (jm *JobManager) setJob(j Job) {
	jm.jobs.Set(j)
	meta := &JobMeta{Name: j.Name()}
	if typed, isTyped := j.(EnabledProvider); isTyped {
		meta.EnabledProvider = typed.Enabled
	}
	jm.jobMetas.Set(meta)
}

func (jm *JobManager) getRunningTask(taskName string) (task Task) {
	if t, has := jm.tasks.Get(taskName); has {
		task = t.Task
	}
	return
}

func (jm *JobManager) setRunningTask(t Task) {
	tm := &TaskMeta{StartTime: jm.timeSource.Now(), Name: t.Name(), Task: t}
	if typed, isTyped := t.(TimeoutProvider); isTyped {
		tm.Timeout = tm.StartTime.Add(typed.Timeout())
	}
	jm.tasks.Set(tm)
}

func (jm *JobManager) getRunningTaskStartTime(taskName string) (startTime time.Time) {
	if t, has := jm.tasks.Get(taskName); has {
		startTime = t.StartTime
	}
	return
}

func (jm *JobManager) setRunningTaskStartTime(taskName string, startTime time.Time) {
	if t, has := jm.tasks.Get(taskName); has {
		t.StartTime = startTime
	}
}

func (jm *JobManager) getSchedule(jobName string) (schedule Schedule) {
	if j, has := jm.jobs.Get(jobName); has {
		schedule = j.Schedule()
	}
	return
}
