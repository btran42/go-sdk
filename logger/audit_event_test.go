package logger

import (
	"bytes"
	"sync"
	"testing"
	"time"

	assert "github.com/blend/go-sdk/assert"
)

func TestAuditEventListener(t *testing.T) {
	assert := assert.New(t)

	textBuffer := bytes.NewBuffer(nil)
	jsonBuffer := bytes.NewBuffer(nil)

	all := New().WithFlags(AllFlags()).
		WithRecoverPanics(false).
		WithWriter(NewTextWriter(textBuffer)).
		WithWriter(NewJSONWriter(jsonBuffer))
	defer all.Close()

	wg := sync.WaitGroup{}
	wg.Add(4)

	all.Listen(Audit, "default", NewAuditEventListener(func(e *AuditEvent) {
		defer wg.Done()

		assert.Equal(Audit, e.Flag())
		assert.Equal("principal", e.Principal())
		assert.Equal("verb", e.Verb())
	}))

	go func() {
		defer wg.Done()
		all.Trigger(NewAuditEvent("principal", "verb", "noun").
			WithSubject("subject").
			WithProperty("property").
			WithUserAgent("user-agent").
			WithRemoteAddress("remote-address").
			WithExtra(Labels{"foo": "bar"}))
	}()
	go func() {
		defer wg.Done()
		all.Trigger(NewAuditEvent("principal", "verb", "noun").
			WithSubject("subject").
			WithProperty("property").
			WithUserAgent("user-agent").
			WithRemoteAddress("remote-address").
			WithExtra(Labels{"foo": "bar"}))
	}()
	wg.Wait()
	all.Drain()

	assert.NotEmpty(textBuffer.String())
	assert.NotEmpty(jsonBuffer.String())
}

func TestAuditEventInterfaces(t *testing.T) {
	assert := assert.New(t)

	ae := NewAuditEvent("principal", "verb", "noun").WithHeadings("heading").WithLabel("foo", "bar")

	eventProvider, isEvent := MarshalEvent(ae)
	assert.True(isEvent)
	assert.Equal(Audit, eventProvider.Flag())
	assert.False(eventProvider.Timestamp().IsZero())

	headingProvider, isHeadingProvider := MarshalEventHeadings(ae)
	assert.True(isHeadingProvider)
	assert.Equal([]string{"heading"}, headingProvider.Headings())

	metaProvider, isMetaProvider := MarshalEventMetaProvider(ae)
	assert.True(isMetaProvider)
	assert.Equal("bar", metaProvider.Labels()["foo"])
}

func TestAuditEventProperties(t *testing.T) {
	assert := assert.New(t)

	ae := NewAuditEvent("", "", "")
	assert.False(ae.Timestamp().IsZero())
	assert.True(ae.WithTimestamp(time.Time{}).Timestamp().IsZero())

	assert.Equal(Audit, ae.Flag())
	assert.Equal(Fatal, ae.WithFlag(Fatal).Flag())

	assert.Empty(ae.Principal())
	assert.Equal("Principal", ae.WithPrincipal("Principal").Principal())

	assert.Empty(ae.Verb())
	assert.Equal("Verb", ae.WithVerb("Verb").Verb())

	assert.Empty(ae.Noun())
	assert.Equal("Noun", ae.WithNoun("Noun").Noun())

	assert.Empty(ae.Headings())
	assert.Equal([]string{"Heading"}, ae.WithHeadings("Heading").Headings())

	assert.Empty(ae.Subject())
	assert.Equal("Subject", ae.WithSubject("Subject").Subject())

	assert.Empty(ae.Property())
	assert.Equal("Property", ae.WithProperty("Property").Property())

	assert.Empty(ae.RemoteAddress())
	assert.Equal("RemoteAddress", ae.WithRemoteAddress("RemoteAddress").RemoteAddress())

	assert.Empty(ae.UserAgent())
	assert.Equal("UserAgent", ae.WithUserAgent("UserAgent").UserAgent())

	assert.Empty(ae.Labels())
	assert.Equal("bar", ae.WithLabel("foo", "bar").Labels()["foo"])

	assert.Empty(ae.Annotations())
	assert.Equal("zar", ae.WithAnnotation("moo", "zar").Annotations()["moo"])

	assert.Empty(ae.Extra())
	assert.Equal("buzz", ae.WithExtra(map[string]string{"wuzz": "buzz"}).Extra()["wuzz"])
}
