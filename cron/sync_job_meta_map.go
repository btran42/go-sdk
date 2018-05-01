package cron

import "sync"

// SyncJobMetaMap is a concurrent job meta map.
type SyncJobMetaMap struct {
	sync.Mutex
	jobs map[string]*JobMeta
}

// Get gets a job.
func (jm *SyncJobMetaMap) Get(name string) (job *JobMeta, has bool) {
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
func (jm *SyncJobMetaMap) Set(jobMeta *JobMeta) {
	jm.Lock()
	if jm.jobs == nil {
		jm.jobs = map[string]*JobMeta{}
	}
	jm.jobs[jobMeta.Name] = jobMeta
	jm.Unlock()
	return
}

// Delete deletes a job.
func (jm *SyncJobMetaMap) Delete(name string) {
	jm.Lock()
	delete(jm.jobs, name)
	jm.Unlock()
	return
}

// Contains returns if the map has a given name.
func (jm *SyncJobMetaMap) Contains(name string) (has bool) {
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
func (jm *SyncJobMetaMap) All() (all []*JobMeta) {
	jm.Lock()
	for _, job := range jm.jobs {
		all = append(all, job)
	}
	jm.Unlock()
	return
}

// Each performs an action for each item in the map.
func (jm *SyncJobMetaMap) Each(action func(string, *JobMeta)) {
	jm.Lock()
	defer jm.Unlock()
	for key, job := range jm.jobs {
		action(key, job)
	}
}
