package secrets

import (
	"fmt"
)

var _ Client = &MockClient{}

// NewMockClient creates a new mock client.
func NewMockClient() *MockClient {
	return &MockClient{
		SecretValues: make(map[string]Values),
	}
}

// MockClient is a mock events client
type MockClient struct {
	SecretValues map[string]Values
}

// Put puts a value.
func (c *MockClient) Put(key string, data Values, options ...Option) error {
	c.SecretValues[key] = data

	return nil
}

// Get gets a value at a given key.
func (c *MockClient) Get(key string, options ...Option) (Values, error) {
	val, exists := c.SecretValues[key]
	if !exists {
		return nil, fmt.Errorf("Key not found: %s", key)
	}

	return val, nil
}

// Delete deletes a key.
func (c *MockClient) Delete(key string, options ...Option) error {
	if _, exists := c.SecretValues[key]; !exists {
		return fmt.Errorf("Key not found: %s", key)
	}

	delete(c.SecretValues, key)

	return nil
}
