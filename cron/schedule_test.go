package cron

import (
	"fmt"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/util"
	"github.com/blend/go-sdk/worker"
)

func TestIntervalSchedule(t *testing.T) {
	a := assert.New(t)

	schedule := EveryHour()

	ts := worker.SystemTimeSource{}

	now := ts.Now()

	firstRun := schedule.GetNextRunTime(ts, nil)
	firstRunDiff := firstRun.Sub(now)
	a.InDelta(float64(firstRunDiff), float64(1*time.Hour), float64(1*time.Second))

	next := schedule.GetNextRunTime(ts, &now)
	a.True(next.After(now))
}

func TestDailyScheduleEveryDay(t *testing.T) {
	a := assert.New(t)
	schedule := DailyAt(12, 0, 0) //noon
	ts := worker.SystemTimeSource{}
	now := ts.Now()
	beforenoon := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, time.UTC)
	afternoon := time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, time.UTC)
	todayAtNoon := schedule.GetNextRunTime(ts, &beforenoon)
	tomorrowAtNoon := schedule.GetNextRunTime(ts, &afternoon)

	a.True(todayAtNoon.Before(afternoon))
	a.True(tomorrowAtNoon.After(afternoon))
}

func TestDailyScheduleSingleDay(t *testing.T) {
	a := assert.New(t)
	schedule := WeeklyAt(12, 0, 0, time.Monday)                  //every monday at noon
	beforenoon := time.Date(2016, 01, 11, 11, 0, 0, 0, time.UTC) //these are both a monday
	afternoon := time.Date(2016, 01, 11, 13, 0, 0, 0, time.UTC)  //these are both a monday

	sundayBeforeNoon := time.Date(2016, 01, 17, 11, 0, 0, 0, time.UTC) //to gut check that it's monday

	ts := worker.SystemTimeSource{}

	todayAtNoon := schedule.GetNextRunTime(ts, &beforenoon)
	nextWeekAtNoon := schedule.GetNextRunTime(ts, &afternoon)

	a.NonFatal().True(todayAtNoon.Before(afternoon))
	a.NonFatal().True(nextWeekAtNoon.After(afternoon))
	a.NonFatal().True(nextWeekAtNoon.After(sundayBeforeNoon))
	a.NonFatal().Equal(time.Monday, nextWeekAtNoon.Weekday())
}

func TestDayOfWeekFunctions(t *testing.T) {
	assert := assert.New(t)

	for _, wd := range WeekendDays {
		assert.True(IsWeekendDay(wd))
		assert.False(IsWeekDay(wd))
	}

	for _, wd := range WeekDays {
		assert.False(IsWeekendDay(wd))
		assert.True(IsWeekDay(wd))
	}
}

func TestOnTheHourAt(t *testing.T) {
	assert := assert.New(t)

	ts := worker.SystemTimeSource{}
	now := ts.Now()

	schedule := EveryHourAt(40)

	fromNil := schedule.GetNextRunTime(ts, nil)
	assert.NotNil(fromNil)

	fromNilExpected := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 40, 0, 0, time.UTC)
	if fromNilExpected.Before(now) {
		fromNilExpected = fromNilExpected.Add(time.Hour)
	}
	assert.InTimeDelta(fromNilExpected, *fromNil, time.Second)

	fromHalfStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 45, 0, 0, time.UTC)
	fromHalfExpected := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 40, 0, 0, time.UTC)

	fromHalf := schedule.GetNextRunTime(ts, util.OptionalTime(fromHalfStart))

	assert.NotNil(fromHalf)
	assert.InTimeDelta(fromHalfExpected, *fromHalf, time.Second)
}

func TestImmediatelyThen(t *testing.T) {
	assert := assert.New(t)

	ts := worker.SystemTimeSource{}

	s := Immediately().Then(EveryHour())
	assert.NotNil(s.GetNextRunTime(ts, nil))
	now := ts.Now()
	next := Deref(s.GetNextRunTime(ts, Optional(now)))
	assert.True(next.Sub(now) > time.Minute, fmt.Sprintf("%v", next.Sub(now)))
	assert.True(next.Sub(now) < (2 * time.Hour))
}
