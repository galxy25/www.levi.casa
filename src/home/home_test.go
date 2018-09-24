package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/galxy25/www.levi.casa/home/communicator"
	"github.com/galxy25/www.levi.casa/home/data"
	await "github.com/galxy25/www.levi.casa/home/internal/await"
	helper "github.com/galxy25/www.levi.casa/home/internal/test"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Mapping from endpoint lookup key to connection type
var newConnectionEndpoints = map[string]string{
	"email": "NEWEMAIL",
	"sms":   "NEWSMS",
}

// Path to test executables dir for use by the test run
var projectRoot = os.Getenv("PROJECT_ROOT")

// An instance of the home process
// executed as part of integration testing
type HomeTestProcess struct {
	test_context *testing.T // Interface to a `go test` invocation
	pid          int        // Unique runtime identifier for this process
	host_name    string     // Name of the host for this process
	host_port    int        // Host port used by this process
}

// Start starts a test instance of home
// returning any error associated with the start
func (l *HomeTestProcess) Start() (err error) {
	// Ensure other levihouse processes are stopped
	// running before testing a new one as otherwise
	// we could fail to start our test server
	// yet still get a healthy response from
	// a previous instance
	run_cmd := exec.Command("sh", "-c", fmt.Sprintf("make restart -f %v/Makefile -C %v", projectRoot, projectRoot))
	out, err := run_cmd.CombinedOutput()
	if err != nil {
		l.test_context.Logf("Failed to run home: %v, %v", string(out), err)
		return err
	}
	// HACK: `make restart` returns
	// echo "home_PID: $$!"
	// as it's last line of output
	// TODO: use losf -i :`l.host_port` to get
	// the pid of `l` we launched to listen on `l.host_port`
	l.test_context.Logf("Run output: %v", string(out))
	sliced_output := strings.Split(string(out), "HOME_PID:")
	string_pid := strings.TrimSpace(strings.Split(sliced_output[len(sliced_output)-1], " ")[1])
	// Set runtime values
	l.pid, err = strconv.Atoi(string_pid)
	l.host_port = homePort
	l.host_name = homeAddress
	if err != nil {
		l.test_context.Logf("Failed to convert %v to int", string_pid)
		return err
	}
	l.test_context.Logf("home pid: %v", l.pid)
	return err
}

// Stop stops a test instance of home
// returning any error associated with the stop
func (l *HomeTestProcess) Stop() (err error) {
	// Check the process is still running
	// On *nix systems, always succeeds
	// so discard error
	home, _ := os.FindProcess(l.pid)
	// https://stackoverflow.com/questions/15204162/check-if-a-process-exists-in-go-way
	tap_response := home.Signal(syscall.Signal(0))
	l.test_context.Logf("process.Signal on pid %d returned: %v\n", l.pid, tap_response)
	// http://man7.org/linux/man-pages/man2/kill.2.html#RETURN_VALUE
	// No news is good news
	// (which for the initiated is not news)
	if tap_response != nil {
		l.test_context.Log("kill -s 0 on home returned non nil, unable to stop non-running server.")
		return errors.New("kill -s 0 on home returned non nil, unable to stop non-running server.")
	}
	// Stop the process
	stop_cmd := exec.Command("sh", "-c", fmt.Sprintf("make stop -f %v/Makefile -C %v", projectRoot, projectRoot))
	out, err := stop_cmd.CombinedOutput()
	if err != nil {
		l.test_context.Logf("Failed to stop home: %v, %v", string(out), err)
		return err
	}
	l.test_context.Logf("Stop output: %v", string(out))
	// Verify the process is stopped
	stop_response := home.Signal(syscall.Signal(0))
	// TODO: Extract retry logic to TestableProcess
	for stop_response == nil {
		time.Sleep(1 * time.Millisecond)
		// TODO: Don't loop longer than
		// 1 seconds
		// waiting for the server to heed stop command
		l.test_context.Logf("process.Signal on pid %d returned: %v\n", l.pid, stop_response)
		stop_response = home.Signal(syscall.Signal(0))
	}
	// XXX: RUNTIME dependent
	// macOS
	// linux (ubuntu)
	if !strings.Contains(stop_response.Error(), "process already finished") && !strings.Contains(stop_response.Error(), "No such process") {
		err = errors.New(fmt.Sprintf("kill -s 0 on home %v returned %v", home, stop_response))
	}
	return err
}

// HealthCheck performs a health check on a home test instance
// returning bool to indicate process health
// and any associated error encountered during the health check
func (l *HomeTestProcess) HealthCheck() (healthy bool, err error) {
	// Pessimistic assumption of
	// sick until proven healthy!
	healthy = false
	// Call the health endpoint to verify
	// it is running
	resp, err := l.Call("HEALTH", nil)
	// TODO: Extract timeout and retry logic to TestableProcess
	tries := 10
	for err != nil && tries > 0 {
		resp, err = l.Call("HEALTH", nil)
		tries--
		l.test_context.Logf("Health check retries left: %v", tries)
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		l.test_context.Logf("Unable to ping home: %v \n", err)
		return healthy, err
	}
	// Parse health check response
	ping_resp := castToResponse(resp)
	// Check health check response
	if ping_resp.Message == HealthCheckOk && ping_resp.StatusCode == http.StatusOK {
		healthy = true
	}
	return healthy, err
}

// Call calls a home process
// returning the call response and error
func (l *HomeTestProcess) Call(method string, body interface{}) (response interface{}, err error) {
	endpoint, exists := Endpoints[method]
	if !exists {
		return response, errors.New(fmt.Sprintf("No matching endpoint found for method: %v\n Valid Endpoints are: %v\n", method, Endpoints))
	}
	response, err = l.client(endpoint, body)
	return response, err
}

// client wraps an http client for making calls
// to www.levi.casa servers.
func (l *HomeTestProcess) client(endpoint Endpoint, body interface{}) (response Response, err error) {
	call_path := l.endpoint_uri(endpoint)
	switch endpoint.Verb {
	case "GET":
		resp, err := http.Get(call_path)
		if err != nil {
			return response, err
		}
		json.NewDecoder(resp.Body).Decode(&response)
		return response, err
	case "POST":
		body_bytes := new(bytes.Buffer)
		json.NewEncoder(body_bytes).Encode(body)
		encoded_body := ioutil.NopCloser(body_bytes)
		resp, err := http.Post(call_path, "application/json", encoded_body)
		if err != nil {
			return response, err
		}
		json.NewDecoder(resp.Body).Decode(&response)
		return response, err
	default:
		return response, err
	}
}

// host_address returns the network address of the
// host running the home test process
func (l *HomeTestProcess) host_address() (address string) {
	address = fmt.Sprintf("%v", l.host_name)
	return address
}

// endpoint_uri returns the network URI
// exposed by the running home for the provided endpoint
func (l *HomeTestProcess) endpoint_uri(endpoint Endpoint) (endpoint_uri string) {
	endpoint_uri = fmt.Sprintf("https://%v%v", l.host_address(), endpoint.Path)
	return endpoint_uri
}

// castToResponse casts any interface to
// type Response, the format used for valid
// levihouse's responses.
func castToResponse(anyInterface interface{}) (cast_response Response) {
	cast_response, _ = anyInterface.(Response)
	return cast_response
}

// Blackest of black box testing:
// "Is the power light on?"
// "Is the power light off?"
func TestItRunsAndStops(t *testing.T) {
	test_house := HomeTestProcess{test_context: t}
	house_under_test, err := helper.ExecuteTestProcess(&test_house)
	if err != nil {
		t.Fatalf("Failed to run home: %v\n", err)
	}
	stop_error := house_under_test.Terminate()
	if stop_error != nil {
		t.Fatalf("Failed to stop home: %v\n ", stop_error)
	}
}

// E2E integration test
// connection -> home => connected
func TestHomeMakesConnectionsInUnderOneSecond(t *testing.T) {
	test_house := HomeTestProcess{test_context: t}
	house_under_test, _ := helper.ExecuteTestProcess(&test_house)
	defer house_under_test.Terminate()
	for connectionType, endpoint := range newConnectionEndpoints {
		// Construct connection to make
		connection := helper.ConnectionGenerators[connectionType]()
		// Send connection to home
		resp, err := house_under_test.Call(endpoint, connection)
		if err != nil {
			t.Errorf("%v\nFailed to initiate connection %v\n%v\n", err, connection, resp)
		}
		var persisted_connection data.Connection
		err = json.Unmarshal([]byte(castToResponse(resp).Json), &persisted_connection)
		if err != nil {
			t.Errorf("connect endpoint responded with invalid connection response: %v\n%v\n", resp, err)
		}
		// Verify test connection registered
		match := false
		tries := 20
		// Get the current list of connections
		var connections data.Connections
		for tries > 0 && !match {
			resp, err = house_under_test.Call("INBOX", nil)
			err = json.Unmarshal([]byte(castToResponse(resp).Json), &connections)
			for _, connected := range connections.Connections {
				if connected.Equals(&persisted_connection) {
					match = true
					break
				}
			}
			if match {
				break
			}
			tries--
			time.Sleep(100 * time.Millisecond)
		}
		if !match {
			t.Errorf("Test connection %v \n not present in list of connections %v ", persisted_connection, connections)
		}
	}
}

// Test on restart unmade connections get made
func TestHomeReconcilesUnlinkedConnectionsOnStartup(t *testing.T) {
	connections := []*data.Connection{
		helper.RandomEmailConnection(),
		helper.RandomSmsConnection(),
	}
	// messaging services frugally frown
	// on sending undeliverable messages
	connections[0].Receiver = homeEmail
	connections[1].Receiver = homePhone
	desiredConnectionsFile := communicator.NewConnectionFile(fmt.Sprintf("%v/%v", projectRoot, desiredConnectionsFilePath))
	err := desiredConnectionsFile.WriteConnections(connections)
	if err != nil {
		t.Errorf("failed to write connections %v to %v", connections, desiredConnectionsFilePath)
	}
	test_house := HomeTestProcess{test_context: t}
	house_under_test, _ := helper.ExecuteTestProcess(&test_house)
	defer house_under_test.Terminate()
	var madeConnections data.Connections
	connectionsMade := func() (made bool, err error) {
		resp, err := house_under_test.Call("INBOX", nil)
		if err != nil {
			return made, err
		}
		err = json.Unmarshal([]byte(castToResponse(resp).Json), &madeConnections)
		if err != nil {
			return made, err
		}
		for _, connection := range connections {
			made = false
			for _, connected := range madeConnections.Connections {
				if connected.Equals(connection) {
					made = true
					break
				}
			}
			if !made {
				return made, err
			}
		}
		return made, err
	}
	cancel := make(chan struct{})
	defer close(cancel)
	made, err := await.Await(connectionsMade, cancel)
	if err != nil {
		t.Error(err)
	}
	if !made {
		var unmadeConnections []*data.Connection
		unlinkedIterator, _ := comm.Received(cancel)
		for unlinked := range unlinkedIterator {
			unmadeConnections = append(unmadeConnections, unlinked)
		}
		t.Errorf("failed to connect all of %v\n current unmade connections %v\n", connections, unmadeConnections)
	}
}

func TestHomeReconcileOnStartupNoOpsForLinkedConnections(t *testing.T) {
	connections := []*data.Connection{
		helper.RandomEmailConnection(),
		helper.RandomEmailConnection(),
	}
	desiredConnectionsFile := communicator.NewConnectionFile(fmt.Sprintf("%v/%v", projectRoot, desiredConnectionsFilePath))
	err := desiredConnectionsFile.WriteConnections(connections)
	if err != nil {
		t.Errorf("failed to write connections %v to %v", connections, desiredConnectionsFilePath)
	}
	currentConnectionFile := communicator.NewConnectionFile(fmt.Sprintf("%v/%v", projectRoot, currentConnectionsFilePath))
	err = currentConnectionFile.WriteConnections(connections)
	if err != nil {
		t.Errorf("failed to write connections %v to %v", connections, currentConnectionsFilePath)
	}
	test_house := HomeTestProcess{test_context: t}
	house_under_test, _ := helper.ExecuteTestProcess(&test_house)
	defer house_under_test.Terminate()
	resp, err := house_under_test.Call("METRICS", nil)
	if err != nil {
		t.Error(err)
	}
	metrics := make(map[string]int64)
	err = json.Unmarshal([]byte(castToResponse(resp).Json), &metrics)
	if err != nil {
		t.Error(err)
	}
	if metrics["unlinked"] != metrics["linked"] {
		t.Errorf("expected same number of linked and unlinked connections, got %v\n", metrics)
	}
}

func TestHomeMakesConnectionWhenSenderIsNotReplyable(t *testing.T) {
	test_house := HomeTestProcess{test_context: t}
	house_under_test, _ := helper.ExecuteTestProcess(&test_house)
	defer house_under_test.Terminate()
	// Construct connection to make
	connection := helper.RandomEmailConnection()
	connection.Sender = "not a valid email address"
	connection.Receiver = homeEmail
	// Send connection to home
	resp, err := house_under_test.Call("NEWEMAIL", connection)
	if err != nil {
		t.Fatalf("%v\nFailed to initiate connection %v\n%v\n", err, connection, resp)
	}
	var persisted_connection data.Connection
	err = json.Unmarshal([]byte(castToResponse(resp).Json), &persisted_connection)
	if err != nil {
		t.Errorf("connect endpoint responded with invalid connection response: %v\n%v\n", resp, err)
	}
	// Verify test connection registered
	match := false
	tries := 10
	// Get the current list of connections
	var connections data.Connections
	for tries > 0 && !match {
		resp, err = house_under_test.Call("INBOX", nil)
		err = json.Unmarshal([]byte(castToResponse(resp).Json), &connections)
		for _, connected := range connections.Connections {
			if connected.Equals(&persisted_connection) {
				match = true
				break
			}
		}
		if match {
			break
		}
		tries--
		time.Sleep(100 * time.Millisecond)
	}
	if !match {
		t.Fatalf("Test connection %v \n not present in list of connections %v ", persisted_connection, connections)
	}
}
