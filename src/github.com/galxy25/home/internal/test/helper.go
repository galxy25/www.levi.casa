package internal

import (
	"fmt"
	"github.com/galxy25/home/data"
	"math/rand"
	"time"
)

// A TestProcess is-a process
// intended to be executed by some
// `func TestFoo(t *testing.T) {...}
// as part of a
// `go test` invocation
// A brute force approach: Raise interfaces!
// a.k.a. 👶🏾's first golang interface
type TestProcess struct {
	TestableProcess // A test process has-a TestableProcess
}

// TestableProcess is the interface for
// starting, stopping, calling and monitoring a
// process that is intended to be used as part of a test
type TestableProcess interface {
	Start() (err error) // Constructor
	Stop() (err error)  // De-constructor
	HealthCheck() (healthy bool, err error)
	Call(method string, body interface{}) (response interface{}, err error) // {f = method, x = body, y = response + error; f(x) = y}
}

// ExecuteTestProcess executes a testable process
// and returns the executed test process
// (which may not be running)
// and any execution error
func ExecuteTestProcess(t TestableProcess) (tp *TestProcess, err error) {
	tp = &TestProcess{t}
	err = t.Start()
	if err != nil {
		return tp, err
	}
	// TODO:
	// Refactor up/out/obviate
	// timeout and retry abilities
	// from implementors of TestableProcess
	_, err = t.HealthCheck()
	return tp, err
}

// Stop attempts to stop the TestProcess
// returning any error encountered stopping the process
func (t *TestProcess) Terminate() (err error) {
	// TODO:
	// Add timeout and retry ability
	return t.Stop()
}

const alphaNumeralSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const numberSet = "0123456789"

// RandomString returns a random string of
// length length from randomSet.
// https://www.calhoun.io/6-tips-for-using-strings-in-go/
func RandomString(length int, randomSet string) string {
	if randomSet == "" {
		randomSet = alphaNumeralSet
	}
	source := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = randomSet[source.Int63()%int64(len(randomSet))]
	}
	return string(b)
}

// RandomEmailConnection returns a valid and
// randomly generated email connection.
func RandomEmailConnection() (connection *data.Connection) {
	connectEpoch := time.Now()
	connection = &data.Connection{
		Message:      RandomString(100, alphaNumeralSet),
		Sender:       fmt.Sprintf("%v@%v.com", RandomString(10, alphaNumeralSet), RandomString(10, alphaNumeralSet)),
		Receiver:     fmt.Sprintf("%v@%v.com", RandomString(10, alphaNumeralSet), RandomString(10, alphaNumeralSet)),
		SendEpoch:    connectEpoch.Unix(),
		ReceiveEpoch: connectEpoch.Add(time.Second).Unix(),
	}
	return connection
}

// RandomSmsConnection returns a valid and
// randomly generated sms connection.
func RandomSmsConnection() (connection *data.Connection) {
	connectEpoch := time.Now()
	connection = &data.Connection{
		Message:      RandomString(100, alphaNumeralSet),
		Sender:       fmt.Sprintf("+%v", RandomString(11, numberSet)),
		Receiver:     fmt.Sprintf("+%v", RandomString(11, numberSet)),
		SendEpoch:    connectEpoch.Unix(),
		ReceiveEpoch: connectEpoch.Add(time.Second).Unix(),
	}
	return connection
}

// ConnectionGenerator is a type of function that
// generates a connection.
type ConnectionGenerator func() *data.Connection

// ConnectionGenerators maps a connection type to a ConnectionGenerator for the given type
var ConnectionGenerators = map[string]ConnectionGenerator{
	"email": RandomEmailConnection,
	"sms":   RandomSmsConnection,
}
