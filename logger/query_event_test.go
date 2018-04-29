package logger

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestQueryEventListener(t *testing.T) {
	assert := assert.New(t)

	wg := sync.WaitGroup{}
	wg.Add(4)

	textBuffer := bytes.NewBuffer(nil)
	jsonBuffer := bytes.NewBuffer(nil)
	all := New().WithFlags(AllFlags()).WithRecoverPanics(false).
		WithWriter(NewTextWriter(textBuffer)).
		WithWriter(NewJSONWriter(jsonBuffer))
	defer all.Close()

	all.Listen(Query, "default", NewQueryEventListener(func(e *QueryEvent) {
		defer wg.Done()

		assert.Equal(Query, e.Flag())
		assert.Equal("moo", e.QueryLabel())
		assert.Equal("foo bar", e.Body())
	}))

	go func() { defer wg.Done(); all.Trigger(NewQueryEvent("foo bar", time.Second).WithQueryLabel("moo")) }()
	go func() { defer wg.Done(); all.Trigger(NewQueryEvent("foo bar", time.Second).WithQueryLabel("moo")) }()
	wg.Wait()
	all.Drain()

	assert.NotEmpty(textBuffer.String())
	assert.NotEmpty(jsonBuffer.String())
}

func TestQueryEventInterfaces(t *testing.T) {
	assert := assert.New(t)

	ee := NewQueryEvent("this is a test", time.Second).
		WithHeadings("heading").
		WithLabel("foo", "bar")

	eventProvider, isEvent := MarshalEvent(ee)
	assert.True(isEvent)
	assert.Equal(Query, eventProvider.Flag())
	assert.False(eventProvider.Timestamp().IsZero())

	headingProvider, isHeadingProvider := MarshalEventHeadings(ee)
	assert.True(isHeadingProvider)
	assert.Equal([]string{"heading"}, headingProvider.Headings())

	metaProvider, isMetaProvider := MarshalEventMetaProvider(ee)
	assert.True(isMetaProvider)
	assert.Equal("bar", metaProvider.Labels()["foo"])
}

func TestQueryEventProperties(t *testing.T) {
	assert := assert.New(t)

	e := NewQueryEvent("", 0)
	assert.False(e.Timestamp().IsZero())
	assert.True(e.WithTimestamp(time.Time{}).Timestamp().IsZero())

	assert.Empty(e.Labels())
	assert.Equal("bar", e.WithLabel("foo", "bar").Labels()["foo"])

	assert.Empty(e.Annotations())
	assert.Equal("zar", e.WithAnnotation("moo", "zar").Annotations()["moo"])

	assert.Equal(Query, e.Flag())
	assert.Equal(Error, e.WithFlag(Error).Flag())

	assert.Empty(e.Headings())
	assert.Equal([]string{"Heading"}, e.WithHeadings("Heading").Headings())

	assert.Empty(e.Body())
	assert.Equal("Body", e.WithBody("Body").Body())

	assert.Empty(e.QueryLabel())
	assert.Equal("QueryLabel", e.WithQueryLabel("QueryLabel").QueryLabel())

	assert.Empty(e.Engine())
	assert.Equal("Engine", e.WithEngine("Engine").Engine())

	assert.Empty(e.Database())
	assert.Equal("Database", e.WithDatabase("Database").Database())

	assert.Zero(e.Elapsed())
	assert.Equal(time.Second, e.WithElapsed(time.Second).Elapsed())

	assert.Nil(e.Err())
	assert.Equal(fmt.Errorf("test"), e.WithErr(fmt.Errorf("test")).Err())
}
