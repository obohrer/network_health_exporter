package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertEmpty(assert *assert.Assertions, reads chan *readOp) {
	read := &readOp{
		resp: make(chan map[string]*hostResults)}
	reads <- read
	state := <-read.resp
	assert.Empty(state)
}

func TestStateManager(t *testing.T) {
	assert := assert.New(t)
	reads := make(chan *readOp)
	pendings := make(chan *pingPending)
	responses := make(chan *pingResponse)

	cfg := Configuration{IntervalSeconds: 5, TimeoutSeconds: 5}

	// Start the statemanager
	startStateManager(cfg, reads, pendings, responses)

	// Test the initial state
	assertEmpty(assert, reads)

	//insert one pending ping
	respChan := make(chan bool)
	pendings <- &pingPending{id: 42, host: "test", start: 10, resp: respChan}
	assert.True(<-respChan)
	// Still empty
	assertEmpty(assert, reads)

	// Add a bad ping response
	responses <- &pingResponse{id: 43, t: 22, resp: respChan}
	assert.True(<-respChan)
	assertEmpty(assert, reads)

	// Add a correct ping response
	responses <- &pingResponse{id: 42, t: 22, resp: respChan}
	assert.True(<-respChan)

	read := &readOp{
		resp: make(chan map[string]*hostResults)}
	reads <- read
	state := <-read.resp
	assert.Equal(len(state), 1)
	assert.Equal(state["test"], &hostResults{host: "test", lastElapsed: 12, completed: 1, timeouts: 0})

	// Add new ping responses for "test" and one for "test2"
	responses <- &pingResponse{id: 42, t: 22, resp: respChan}
	assert.True(<-respChan)

	pendings <- &pingPending{id: 44, host: "test", start: 50, resp: respChan}
	assert.True(<-respChan)
	responses <- &pingResponse{id: 44, t: 70, resp: respChan}
	assert.True(<-respChan)

	pendings <- &pingPending{id: 45, host: "test2", start: 51, resp: respChan}
	assert.True(<-respChan)
	responses <- &pingResponse{id: 45, t: 61, resp: respChan}
	assert.True(<-respChan)

	reads <- read
	state = <-read.resp
	assert.Equal(len(state), 2)
	assert.Equal(state["test"], &hostResults{host: "test", lastElapsed: 20, completed: 2, timeouts: 0})
	assert.Equal(state["test2"], &hostResults{host: "test2", lastElapsed: 10, completed: 1, timeouts: 0})
}
