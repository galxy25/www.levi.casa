package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	helper "github.com/galxy25/levishouse/internal/test"
	xip "github.com/galxy25/levishouse/xip"
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

// Path to test executables dir for use by the test run
var project_root = flag.String("project_root", "", "App root directory for the package under test")

// An instance of the levishouse process
// executed as part of integration testing
type LevishouseTestProcess struct {
	test_context *testing.T // Interface to a `go test` invocation
	pid          int        // Unique runtime identifier for this process
	host_name    string     // Name of the host for this process
	host_port    int        // Host port used by this process
}

// Start starts a test instance of levishouse
// returning any error associated with the start
func (l *LevishouseTestProcess) Start() (err error) {
	// Ensure other levihouse processes are stopped
	// running before testing a new one as otherwise
	// we could fail to start our test server
	// yet still get a healthy response from
	// a previous instance
	run_cmd := exec.Command("sh", "-c", fmt.Sprintf("make restart -f %v/Makefile -C %v", *project_root, *project_root))
	out, err := run_cmd.CombinedOutput()
	if err != nil {
		l.test_context.Logf("Failed to run levishouse: %v, %v", string(out), err)
		return err
	}
	// HACK: `make restart` returns
	// echo "LEVISHOUSE_PID: $$!"
	// as it's last line of output
	// TODO: use losf -i :`l.host_port` to get
	// the pid of `l` we launched to listen on `l.host_port`
	l.test_context.Logf("Run output: %v", string(out))
	sliced_output := strings.Split(string(out), "LEVISHOUSE_PID:")
	string_pid := strings.TrimSpace(strings.Split(sliced_output[len(sliced_output)-1], " ")[1])
	// Set runtime values
	l.pid, err = strconv.Atoi(string_pid)
	l.host_port = home_port
	l.host_name = "localhost"
	if err != nil {
		l.test_context.Logf("Failed to convert %v to int", string_pid)
		return err
	}
	l.test_context.Logf("levishouse pid: %v", l.pid)
	return err
}

// Stop stops a test instance of levishouse
// returning any error associated with the stop
func (l *LevishouseTestProcess) Stop() (err error) {
	// Check the process is still running
	// On *nix systems, always succeeds
	// so discard error
	levishouse, _ := os.FindProcess(l.pid)
	// https://stackoverflow.com/questions/15204162/check-if-a-process-exists-in-go-way
	tap_response := levishouse.Signal(syscall.Signal(0))
	l.test_context.Logf("process.Signal on pid %d returned: %v\n", l.pid, tap_response)
	// http://man7.org/linux/man-pages/man2/kill.2.html#RETURN_VALUE
	// No news is good news
	// (which for the initiated is not news)
	if tap_response != nil {
		l.test_context.Log("kill -s 0 on levishouse returned non nil, unable to stop non-running server.")
		return errors.New("kill -s 0 on levishouse returned non nil, unable to stop non-running server.")
	}
	// Stop the process
	stop_cmd := exec.Command("sh", "-c", fmt.Sprintf("make stop -f %v/Makefile -C %v", *project_root, *project_root))
	out, err := stop_cmd.CombinedOutput()
	if err != nil {
		l.test_context.Logf("Failed to stop levishouse: %v, %v", string(out), err)
		return err
	}
	l.test_context.Logf("Stop output: %v", string(out))
	// Verify the process is stopped
	stop_response := levishouse.Signal(syscall.Signal(0))
	// TODO: Extract retry logic to TestableProcess
	for stop_response == nil {
		time.Sleep(1 * time.Millisecond)
		// TODO: Don't loop longer than
		// 1 seconds
		// waiting for the server to heed stop command
		l.test_context.Logf("process.Signal on pid %d returned: %v\n", l.pid, stop_response)
		stop_response = levishouse.Signal(syscall.Signal(0))
	}
	// XXX: RUNTIME dependent
	// macOS
	// linux (ubuntu)
	if !strings.Contains(stop_response.Error(), "process already finished") && !strings.Contains(stop_response.Error(), "No such process") {
		err = errors.New(fmt.Sprintf("kill -s 0 on levishouse %v returned %v", levishouse, stop_response))
	}
	return err
}

// HealthCheck performs a health check on a levishouse test instance
// returning bool to indicate process health
// and any associated error encountered during the health check
func (l *LevishouseTestProcess) HealthCheck() (healthy bool, err error) {
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
		l.test_context.Logf("Unable to ping levishouse: %v \n", err)
		return healthy, err
	}
	// Parse health check response
	ping_resp := castToResponse(resp)
	l.test_context.Logf("Ping response from levishouse: %v \n", ping_resp)
	// Check health check response
	if ping_resp.Message == HEALTH_CHECK_OK && ping_resp.StatusCode == http.StatusOK {
		healthy = true
	}
	return healthy, err
}

// Call calls a levishouse process
// returning the call response and error
func (l *LevishouseTestProcess) Call(method string, body interface{}) (response interface{}, err error) {
	endpoint, exists := ENDPOINTS[method]
	if !exists {
		return response, errors.New(fmt.Sprintf("No matching endpoint found for method: %v\n Valid endpoints are: %v\n", method, ENDPOINTS))
	}
	response, err = l.client(endpoint, body)
	return response, err
}

func (l *LevishouseTestProcess) client(endpoint Endpoint, body interface{}) (response Response, err error) {
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
// host running the levishouse test process
func (l *LevishouseTestProcess) host_address() (address string) {
	address = fmt.Sprintf("%v:%v", l.host_name, l.host_port)
	return address
}

// endpoint_uri returns the network URI
// exposed by the running levishouse for the provided endpoint
func (l *LevishouseTestProcess) endpoint_uri(endpoint Endpoint) (endpoint_uri string) {
	endpoint_uri = fmt.Sprintf("http://%v%v", l.host_address(), endpoint.Path)
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
	test_house := LevishouseTestProcess{test_context: t}
	house_under_test, err := helper.ExecuteTestProcess(&test_house)
	if err != nil {
		t.Fatalf("Failed to run levishouse: %v\n", err)
	}
	stop_error := house_under_test.Terminate()
	if stop_error != nil {
		t.Fatalf("Failed to stop levishouse: %v\n ", stop_error)
	}
}

// E2E integration test
// connection -> levishouse => connected
func TestLevishouseMakesEmailConnections(t *testing.T) {
	test_house := LevishouseTestProcess{test_context: t}
	house_under_test, _ := helper.ExecuteTestProcess(&test_house)
	defer house_under_test.Terminate()
	// Construct connection to make
	connection := xip.EmailConnect{
		EmailConnect:           "Salutations,Body,Farewell",
		EmailConnectId:         "tester@test.com",
		SubscribeToMailingList: false,
	}
	// Send connection to levishouse
	resp, err := house_under_test.Call("CONNECT", connection)
	if err != nil {
		t.Fatalf("%v\nFailed to initiate connection %v\n%v\n", err, connection, resp)
	}
	// Get the current list of connections
	tries := 10
	var connections xip.Connections
	for tries > 0 {
		resp, err = house_under_test.Call("INBOX", nil)
		err = json.Unmarshal([]byte(castToResponse(resp).Json), &connections)
		if len(connections.EmailConnections) > 0 {
			break
		}
		tries--
		time.Sleep(100 * time.Millisecond)
	}
	// Verify test connection registered
	match := false
	for _, connected := range connections.EmailConnections {
		if connected.EmailConnectId == connection.EmailConnectId && connected.EmailConnect == connection.EmailConnect {
			match = true
			break
		}
	}
	if !match {
		t.Fatalf("Test connection %v \n not present in list of connections %v ", connection, connections)
	}
}
