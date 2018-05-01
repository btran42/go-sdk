package cron

import "sync"

// SyncTaskMetaMap is a concurrent task meta map.
type SyncTaskMetaMap struct {
	sync.Mutex
	tasks map[string]*TaskMeta
}

// Get gets a job.
func (sm *SyncTaskMetaMap) Get(name string) (task *TaskMeta, has bool) {
	sm.Lock()
	if sm.tasks == nil {
		sm.Unlock()
		return
	}
	task, has = sm.tasks[name]
	sm.Unlock()
	return
}

// Set adds a job.
func (sm *SyncTaskMetaMap) Set(taskMeta *TaskMeta) {
	sm.Lock()
	if sm.tasks == nil {
		sm.tasks = map[string]*TaskMeta{}
	}
	sm.tasks[taskMeta.Name] = taskMeta
	sm.Unlock()
	return
}

// Delete deletes a job.
func (sm *SyncTaskMetaMap) Delete(name string) {
	sm.Lock()
	delete(sm.tasks, name)
	sm.Unlock()
	return
}

// Contains returns if the map has a given name.
func (sm *SyncTaskMetaMap) Contains(name string) (has bool) {
	sm.Lock()
	if sm.tasks == nil {
		sm.Unlock()
		return
	}
	_, has = sm.tasks[name]
	sm.Unlock()
	return
}

// All returns all the jobs.
func (sm *SyncTaskMetaMap) All() (all []*TaskMeta) {
	sm.Lock()
	for _, job := range sm.tasks {
		all = append(all, job)
	}
	sm.Unlock()
	return
}

// Each performs an action for each item in the map.
func (sm *SyncTaskMetaMap) Each(action func(string, *TaskMeta)) {
	sm.Lock()
	defer sm.Unlock()
	for key, task := range sm.tasks {
		action(key, task)
	}
}

// Do runs an action for a given name safely.
func (sm *SyncTaskMetaMap) Do(name string, action func(*TaskMeta)) {
	sm.Lock()
	if taskMeta, has := sm.tasks[name]; has {
		action(taskMeta)
	}
	sm.Unlock()
}
