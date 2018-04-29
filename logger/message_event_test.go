package logger

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestMessageEventListener(t *testing.T) {
	assert := assert.New(t)

	wg := sync.WaitGroup{}
	wg.Add(4)

	textBuffer := bytes.NewBuffer(nil)
	jsonBuffer := bytes.NewBuffer(nil)
	all := New().WithFlags(AllFlags()).WithRecoverPanics(false).
		WithWriter(NewTextWriter(textBuffer)).
		WithWriter(NewJSONWriter(jsonBuffer))

	defer all.Close()
	all.Listen(Flag("test-flag"), "default", NewMessageEventListener(func(e *MessageEvent) {
		defer wg.Done()
		assert.Equal("test-flag", e.Flag())
		assert.Equal("foo bar", e.Message())
	}))

	go func() { defer wg.Done(); all.Trigger(Messagef(Flag("test-flag"), "foo %s", "bar")) }()
	go func() { defer wg.Done(); all.Trigger(Messagef(Flag("test-flag"), "foo %s", "bar")) }()
	wg.Wait()
	all.Drain()

	assert.NotEmpty(textBuffer.String())
	assert.NotEmpty(jsonBuffer.String())
}

func TestMessageEventInterfaces(t *testing.T) {
	assert := assert.New(t)

	ee := Messagef(Info, "this is a test").
		WithHeadings("heading").
		WithLabel("foo", "bar")

	eventProvider, isEvent := MarshalEvent(ee)
	assert.True(isEvent)
	assert.Equal(Info, eventProvider.Flag())
	assert.False(eventProvider.Timestamp().IsZero())

	headingProvider, isHeadingProvider := MarshalEventHeadings(ee)
	assert.True(isHeadingProvider)
	assert.Equal([]string{"heading"}, headingProvider.Headings())

	metaProvider, isMetaProvider := MarshalEventMetaProvider(ee)
	assert.True(isMetaProvider)
	assert.Equal("bar", metaProvider.Labels()["foo"])
}

func TestMessageEventProperties(t *testing.T) {
	assert := assert.New(t)

	e := Messagef(Info, "")

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

	assert.Empty(e.FlagTextColor())
	assert.Equal(ColorWhite, e.WithFlagTextColor(ColorWhite).FlagTextColor())
}
