package cron

import "time"

// Now returns the current time in utc.
func Now() time.Time {
	return time.Now().UTC()
}

// Deref derefs a time safely.
func Deref(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// Optional returns an optional time.
func Optional(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// Since returns the time delta since a given time.
func Since(t time.Time) time.Duration {
	return time.Now().UTC().Sub(t)
}

// Min returns the minimum of two times.
func Min(t1, t2 time.Time) time.Time {
	if t1.IsZero() && t2.IsZero() {
		return time.Time{}
	}
	if t1.IsZero() {
		return t2
	}
	if t2.IsZero() {
		return t1
	}
	if t1.Before(t2) {
		return t1
	}
	return t2
}

// Max returns the maximum of two times.
func Max(t1, t2 time.Time) time.Time {
	if t1.Before(t2) {
		return t2
	}
	return t1
}

// FormatTime returns a string for a time.
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
