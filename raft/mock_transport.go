package raft

import "github.com/blend/go-sdk/logger"

var (
	_ Client = &MockTransport{}
	_ Client = NoOpTransport("")
)

// NoOpTransport implements client but does nothing.
type NoOpTransport string

// Open implements Client.
func (t NoOpTransport) Open() error { return nil }

// Close implements Client.
func (t NoOpTransport) Close() error { return nil }

// RemoteAddr implements Client.
func (t NoOpTransport) RemoteAddr() string { return string(t) }

// AppendEntries implements Client.
func (t NoOpTransport) AppendEntries(args *AppendEntries) (*AppendEntriesResults, error) {
	return nil, nil
}

// RequestVote implements Client.
func (t NoOpTransport) RequestVote(args *RequestVote) (*RequestVoteResults, error) {
	return nil, nil
}

// NewMockTransport returns a new mock transport
func NewMockTransport(remoteAddress string, peer Server) *MockTransport {
	return &MockTransport{
		remoteAddr:           remoteAddress,
		appendEntriesHandler: peer.AppendEntriesHandler(),
		requestVoteHandler:   peer.RequestVoteHandler(),
	}
}

// MockTransport implements both Client + Server.
type MockTransport struct {
	appendEntriesHandler AppendEntriesHandler
	requestVoteHandler   RequestVoteHandler
	remoteAddr           string
	log                  *logger.Logger
}

// WithRemoteAddr sets the remote addr for the mock transport.
func (mt *MockTransport) WithRemoteAddr(remoteAddr string) *MockTransport {
	mt.remoteAddr = remoteAddr
	return mt
}

// RemoteAddr returns the remote addr.
func (mt *MockTransport) RemoteAddr() string {
	return mt.remoteAddr
}

// Open is a no-op.
func (mt *MockTransport) Open() error { return nil }

// Close is a no-op.
func (mt *MockTransport) Close() error { return nil }

// RequestVote sends a mock request vote to the injected handlers.
func (mt *MockTransport) RequestVote(args *RequestVote) (*RequestVoteResults, error) {
	var results RequestVoteResults
	if err := mt.requestVoteHandler(args, &results); err != nil {
		return nil, err
	}
	return &results, nil
}

// AppendEntries sends a mock append entries to the injected handler.
func (mt *MockTransport) AppendEntries(args *AppendEntries) (*AppendEntriesResults, error) {
	var results AppendEntriesResults
	if err := mt.appendEntriesHandler(args, &results); err != nil {
		return nil, err
	}
	return &results, nil
}
