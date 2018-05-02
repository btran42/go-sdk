package raft

import (
	"testing"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/worker"
)

func TestRaftCountVotes(t *testing.T) {
	assert := assert.New(t)

	r := New()

	assert.Equal(-1, r.voteOutcome(0, 0))
	assert.Equal(-1, r.voteOutcome(1, 0))

	assert.Equal(-1, r.voteOutcome(0, 1))
	assert.Equal(-1, r.voteOutcome(1, 1))

	assert.Equal(-1, r.voteOutcome(0, 2))
	assert.Equal(0, r.voteOutcome(1, 2))
	assert.Equal(1, r.voteOutcome(2, 2))

	assert.Equal(-1, r.voteOutcome(0, 3))
	assert.Equal(-1, r.voteOutcome(1, 3))
	assert.Equal(1, r.voteOutcome(2, 3))
	assert.Equal(1, r.voteOutcome(3, 3))

	assert.Equal(-1, r.voteOutcome(0, 4))
	assert.Equal(-1, r.voteOutcome(1, 4))
	assert.Equal(0, r.voteOutcome(2, 4))
	assert.Equal(1, r.voteOutcome(3, 4))
	assert.Equal(1, r.voteOutcome(4, 4))

	assert.Equal(-1, r.voteOutcome(0, 4))
	assert.Equal(-1, r.voteOutcome(1, 4))
	assert.Equal(0, r.voteOutcome(2, 4))
	assert.Equal(1, r.voteOutcome(3, 4))
	assert.Equal(1, r.voteOutcome(4, 4))
}

func TestRaftSoloStart(t *testing.T) {
	assert := assert.New(t)

	solo := New()
	solo.Start()
	assert.Empty(solo.Peers())
	assert.Nil(solo.Server())
	assert.Equal(Leader, solo.State())
	assert.Nil(solo.leaderCheckTicker)
	assert.Nil(solo.heartbeatTicker)
}

func TestRaftStart(t *testing.T) {
	t.Skip()
	assert := assert.New(t)

	node := New().WithTimeSource(worker.NewMockTimeSource())
	node.WithServer(NewMockServer())
	node.WithPeer(NoOpTransport("one"))
	node.WithPeer(NoOpTransport("two"))
	node.WithPeer(NoOpTransport("three"))

	node.Start()
	defer node.Stop()

	assert.NotNil(node.Server())
	assert.Len(node.Peers(), 3)

	assert.True(node.leaderCheckTicker.Running())
	assert.True(node.heartbeatTicker.Running())
}
