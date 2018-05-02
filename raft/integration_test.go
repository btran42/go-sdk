package raft

import (
	"sync"
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/uuid"
)

type cluster struct {
	Nodes      []*Raft
	Transports map[string]map[string]Client
}

func createTestNode() *Raft {
	return New().
		WithID(uuid.V4().String()).
		WithLeaderCheckInterval(time.Microsecond).
		WithHeartbeatInterval(time.Microsecond).
		WithElectionTimeout(time.Microsecond).
		WithServer(NewMockServer())
}

func createCluster(nodeCount int) []*Raft {
	if nodeCount <= 0 {
		return nil
	}
	if nodeCount == 1 {
		return []*Raft{createTestNode()}
	}

	// create all the nodes
	peers := make([]*Raft, nodeCount)
	for index := 0; index < nodeCount; index++ {
		peers[index] = createTestNode()
	}

	// cross wire all the nodes
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < nodeCount; j++ {
			if i != j {
				peers[i].WithPeer(NewMockTransport(peers[j].ID(), peers[j].Server()))
			}
		}
	}
	return peers
}

func TestIntegrationRaftSingleNode(t *testing.T) {
	assert := assert.New(t)
	assert.StartTimeout(time.Millisecond)
	defer assert.EndTimeout()

	cluster := createCluster(1)[0]

	wg := sync.WaitGroup{}
	wg.Add(1)
	var didTransitionToLeader bool
	cluster.SetLeaderHandler(func() {
		defer wg.Done()
		didTransitionToLeader = true
	})
	go func() { cluster.Start() }()
	defer cluster.Stop()
	wg.Wait()
	assert.True(didTransitionToLeader)
}

func TestIntegrationRaftCluster(t *testing.T) {
	t.Skip()

	assert := assert.New(t)
	assert.StartTimeout(time.Millisecond)
	defer assert.EndTimeout()

	cluster := createCluster(3)
	for _, node := range cluster {
		assert.Nil(node.Start())
		defer node.Stop()
	}
}
