package cron

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/logger"
)

func TestEventStartedListener(t *testing.T) {
	assert := assert.New(t)

	wg := sync.WaitGroup{}
	wg.Add(4)

	textBuffer := bytes.NewBuffer(nil)
	jsonBuffer := bytes.NewBuffer(nil)

	all := logger.New().WithFlags(logger.AllFlags()).
		WithRecoverPanics(false).
		WithWriter(logger.NewTextWriter(textBuffer)).
		WithWriter(logger.NewJSONWriter(jsonBuffer))

	defer all.Close()

	assert.True(all.IsEnabled(FlagStarted))
	assert.False(all.IsHidden(FlagStarted))

	all.Listen(FlagStarted, "default", NewEventListener(func(e *Event) {
		defer wg.Done()

		assert.Equal(FlagStarted, e.Flag())
		assert.False(e.Timestamp().IsZero())
		assert.Equal("test_task", e.TaskName())

		assert.False(e.Complete())
		assert.Nil(e.Err())
		assert.Zero(e.Elapsed())
	}))

	go func() {
		defer wg.Done()
		all.Trigger(NewEvent(FlagStarted, "test_task"))
	}()
	go func() {
		defer wg.Done()
		all.Trigger(NewEvent(FlagStarted, "test_task"))
	}()

	all.Drain()
	wg.Wait()

	assert.NotEmpty(textBuffer.String())
	assert.NotEmpty(jsonBuffer.String())
}

func TestEventInterfaces(t *testing.T) {
	assert := assert.New(t)

	e := NewEvent(FlagComplete, "test_task").WithHeadings("heading").WithLabel("foo", "bar")

	eventProvider, isEvent := logger.MarshalEvent(e)
	assert.True(isEvent)
	assert.Equal(FlagComplete, eventProvider.Flag())
	assert.False(eventProvider.Timestamp().IsZero())

	headingProvider, isHeadingProvider := logger.MarshalEventHeadings(e)
	assert.True(isHeadingProvider)
	assert.Equal([]string{"heading"}, headingProvider.Headings())

	enabledProvider, isEnabledProvider := logger.MarshalEventEnabled(e)
	assert.True(isEnabledProvider)
	assert.True(enabledProvider.IsEnabled())

	metaProvider, isMetaProvider := logger.MarshalEventMetaProvider(e)
	assert.True(isMetaProvider)
	assert.Equal("bar", metaProvider.Labels()["foo"])
}

func TestNewEvent(t *testing.T) {
	assert := assert.New(t)

	e := NewEvent(FlagComplete, "test_task")
	assert.Equal(FlagComplete, e.Flag())
	assert.Equal("test_task", e.TaskName())
	assert.False(e.Timestamp().IsZero())
	assert.True(e.IsEnabled())
	assert.True(e.IsWritable())
}

func TestEventProperties(t *testing.T) {
	assert := assert.New(t)

	e := NewEvent("", "")
	assert.False(e.Timestamp().IsZero())
	assert.True(e.WithTimestamp(time.Time{}).Timestamp().IsZero())

	assert.Empty(e.Labels())
	assert.Equal("bar", e.WithLabel("foo", "bar").Labels()["foo"])

	assert.Empty(e.Annotations())
	assert.Equal("zar", e.WithAnnotation("moo", "zar").Annotations()["moo"])

	assert.Empty(e.Flag())
	assert.Equal(FlagComplete, e.WithFlag(FlagComplete).Flag())

	assert.True(e.Complete())
	assert.False(e.WithFlag(FlagStarted).Complete())

	assert.Empty(e.Headings())
	assert.Equal([]string{"Heading"}, e.WithHeadings("Heading").Headings())

	assert.Empty(e.TaskName())
	assert.Equal("test_task", e.WithTaskName("test_task").TaskName())

	assert.Zero(e.Elapsed())
	assert.Equal(time.Second, e.WithElapsed(time.Second).Elapsed())

	assert.True(e.IsEnabled())
	assert.False(e.WithIsEnabled(false).IsEnabled())

	assert.True(e.IsWritable())
	assert.False(e.WithIsWritable(false).IsEnabled())

	assert.Nil(e.Err())
	assert.Equal(fmt.Errorf("test"), e.WithErr(fmt.Errorf("test")).Err())
}
