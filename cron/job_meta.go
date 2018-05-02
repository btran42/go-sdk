package cron

import (
	"time"
)

// JobMeta is runtime metadata for a job.
type JobMeta struct {
	Name            string      `json:"name"`
	Disabled        bool        `json:"disabled"`
	Schedule        Schedule    `json:"-"`
	EnabledProvider func() bool `json:"-"`
	NextRunTime     time.Time   `json:"nextRunTime"`
	LastRunTime     time.Time   `json:"lastRunTime"`
}
