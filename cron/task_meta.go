package cron

import (
	"context"
	"time"
)

// TaskMeta is metadata for a running task.
type TaskMeta struct {
	Name      string             `json:"name"`
	Task      Task               `json:"-"`
	StartTime time.Time          `json:"startTime"`
	Timeout   time.Time          `json:"timeout"`
	Context   context.Context    `json:"-"`
	Cancel    context.CancelFunc `json:"-"`
}
