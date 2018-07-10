package internal

import ()

// A TestProcess is-a process
// intended to be executed by some
// `func TestFoo(){...}
// as part of a
// `go test` invocation
type TestProcess struct {
	TestableProcess // A test process has-a TestableProcess
}

// TestableProcess is the interface for
// starting, stopping, and monitoring a
// process that is intended to be used as part of a test
type TestableProcess interface {
	Start() (err error)                     //Constructor
	Stop() (err error)                      //De-constructor
	HealthCheck() (healthy bool, err error) //Monitor
}

// ExecuteTestProcess starts a testable process
// and returns the executed process and start error(if any)
func ExecuteTestProcess(t TestableProcess) (tp *TestProcess, err error) {
	err = t.Start()
	return &TestProcess{t}, err
}

// VerifyHealthy verifies the given TestProcess is healthy
// as defined by it's monitoring function
// TODO:
// Add timeout and retry ability
func (t *TestProcess) VerifyHealthy() (healthy bool, err error) {
	healthy, err = t.HealthCheck()
	return healthy, err
}

// VerifiedStoped stops and verifies the given TestProcess as stopped
// returning an error if the TestProcess was not already running
// TODO:
// Add timeout and retry ability
func (t *TestProcess) VerifiedStoped() (err error) {
	return t.Stop()
}
