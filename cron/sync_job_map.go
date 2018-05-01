package cron

import "sync"

// SyncJobMap is a concurrent job map.
type SyncJobMap struct {
	sync.Mutex
	jobs map[string]Job
}

// Get gets a job.
func (jm *SyncJobMap) Get(name string) (job Job, has bool) {
	jm.Lock()
	if jm.jobs == nil {
		jm.Unlock()
		return
	}
	job, has = jm.jobs[name]
	jm.Unlock()
	return
}

// Set adds a job.
func (jm *SyncJobMap) Set(job Job) {
	jm.Lock()
	if jm.jobs == nil {
		jm.jobs = map[string]Job{}
	}
	jm.jobs[job.Name()] = job
	jm.Unlock()
	return
}

// Delete deletes a job.
func (jm *SyncJobMap) Delete(name string) {
	jm.Lock()
	delete(jm.jobs, name)
	jm.Unlock()
	return
}

// Contains returns if the map has a given name.
func (jm *SyncJobMap) Contains(name string) (has bool) {
	jm.Lock()
	if jm.jobs == nil {
		jm.Unlock()
		return
	}
	_, has = jm.jobs[name]
	jm.Unlock()
	return
}

// All returns all the jobs.
func (jm *SyncJobMap) All() (all []Job) {
	jm.Lock()
	for _, job := range jm.jobs {
		all = append(all, job)
	}
	jm.Unlock()
	return
}

// Each performs an action for each item in the map.
func (jm *SyncJobMap) Each(action func(string, Job)) {
	jm.Lock()
	defer jm.Unlock()
	for key, job := range jm.jobs {
		action(key, job)
	}
}
