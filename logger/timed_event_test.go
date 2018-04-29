package logger

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestTimedEventListener(t *testing.T) {
	assert := assert.New(t)

	wg := sync.WaitGroup{}
	wg.Add(4)

	textBuffer := bytes.NewBuffer(nil)
	jsonBuffer := bytes.NewBuffer(nil)
	all := New().WithFlags(AllFlags()).WithRecoverPanics(false).
		WithWriter(NewTextWriter(textBuffer)).
		WithWriter(NewJSONWriter(jsonBuffer))
	defer all.Close()

	all.Listen(Flag("test-flag"), "default", NewTimedEventListener(func(te *TimedEvent) {
		defer wg.Done()
		assert.Equal("test-flag", te.Flag())
		assert.NotZero(te.Elapsed())
		assert.Equal("foo bar", te.Message())
	}))

	go func() {
		defer wg.Done()
		all.Trigger(Timedf(Flag("test-flag"), time.Millisecond, "foo %s", "bar"))
	}()
	go func() {
		defer wg.Done()
		all.Trigger(Timedf(Flag("test-flag"), time.Millisecond, "foo %s", "bar"))
	}()
	wg.Wait()
	all.Drain()

	assert.NotEmpty(textBuffer.String())
	assert.NotEmpty(jsonBuffer.String())
}

func TestTimedEventInterfaces(t *testing.T) {
	assert := assert.New(t)

	ee := Timedf(Fatal, time.Millisecond, "foo %s", "bar").WithHeadings("heading").WithLabel("foo", "bar")

	eventProvider, isEvent := MarshalEvent(ee)
	assert.True(isEvent)
	assert.Equal(Fatal, eventProvider.Flag())
	assert.False(eventProvider.Timestamp().IsZero())

	headingProvider, isHeadingProvider := MarshalEventHeadings(ee)
	assert.True(isHeadingProvider)
	assert.Equal([]string{"heading"}, headingProvider.Headings())

	metaProvider, isMetaProvider := MarshalEventMetaProvider(ee)
	assert.True(isMetaProvider)
	assert.Equal("bar", metaProvider.Labels()["foo"])
}

func TestTimedEventProperties(t *testing.T) {
	assert := assert.New(t)

	e := Timedf(Info, 0, "")
	assert.False(e.Timestamp().IsZero())
	assert.True(e.WithTimestamp(time.Time{}).Timestamp().IsZero())

	assert.Empty(e.Labels())
	assert.Equal("bar", e.WithLabel("foo", "bar").Labels()["foo"])

	assert.Empty(e.Annotations())
	assert.Equal("zar", e.WithAnnotation("moo", "zar").Annotations()["moo"])

	assert.Equal(Info, e.Flag())
	assert.Equal(Error, e.WithFlag(Error).Flag())

	assert.Empty(e.Headings())
	assert.Equal([]string{"Heading"}, e.WithHeadings("Heading").Headings())

	assert.Empty(e.Message())
	assert.Equal("Message", e.WithMessage("Message").Message())

	assert.Zero(e.Elapsed())
	assert.Equal(time.Second, e.WithElapsed(time.Second).Elapsed())
}
