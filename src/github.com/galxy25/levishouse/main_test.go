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
var test_bin_dir = flag.String("test_bin_dir", "", "Dir of executables for use by the test run")

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
	// HACK:
	// We ensure the server is stopped before
	// running one to test as otherwise
	// we could fail to start our test server
	// yet still get a healthy response from
	// a previous instance
	// TODO:
	// flag/variable-ize web server port
	// get a free port
	// https://github.com/phayes/freeport/blob/master/freeport.go#L9
	// run and test against selected free port
	run_cmd := exec.Command("sh", "-c", fmt.Sprintf("make restart -f %v/Makefile -C %v", *test_bin_dir, *test_bin_dir))
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
	l.host_port = 8081
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
	stop_cmd := exec.Command("sh", "-c", fmt.Sprintf("make stop -f %v/Makefile -C %v", *test_bin_dir, *test_bin_dir))
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
	health_check_endpoint := l.endpoint_uri(ENDPOINTS["HEALTH"])
	resp, err := http.Get(health_check_endpoint)
	// TODO: Extract timeout and retry logic to TestableProcess
	tries := 10
	for err != nil && tries > 1 {
		resp, err = http.Get(health_check_endpoint)
		tries--
		l.test_context.Logf("Health check retries left: %v", tries)
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		l.test_context.Logf("Unable to ping levishouse: %v \n", err)
		return healthy, err
	}
	// Parse health check response
	ping_resp, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		l.test_context.Logf("Unable to parse ping response: %v \n", resp)
		return healthy, err
	}
	l.test_context.Logf("Ping response from levishouse: %v \n", string(ping_resp))
	// Check health check response
	if string(ping_resp) == HEALTH_CHECK_OK {
		healthy = true
	}
	return healthy, err
}

func (l *LevishouseTestProcess) Call(method string, body interface{}) (response interface{}, err error) {
	response, err = l.client(ENDPOINTS[method], body)
	return response, err
}

func (l *LevishouseTestProcess) client(endpoint Endpoint, body interface{}) (response Response, err error) {
	call_path := l.endpoint_uri(endpoint)
	switch endpoint.Verb {
	case "GET":
		resp, err := http.Get(call_path)
		json.NewDecoder(resp.Body).Decode(&response)
		return response, err
	case "POST":
		body_bytes := new(bytes.Buffer)
		json.NewEncoder(body_bytes).Encode(body)
		encoded_body := ioutil.NopCloser(body_bytes)
		resp, err := http.Post(call_path, "application/json", encoded_body)
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

// castToResponse casts any interface to
// type Response, the format used for valid
// levihouse's responses.
func castToResponse(anyInterface interface{}) (cast_response Response) {
	cast_response, _ = anyInterface.(Response)
	return cast_response
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
	// // Get the current list of connections
	// resp, err = house_under_test.Call("INBOX", nil)
	// var connections xip.Connections
	// json.NewDecoder(castToResponse(resp).Data).Decode(&connections)
	// t.Log(connections)
	// if err != nil {
	// 	t.Fatalf("Failed to get list of connections: %v\n", err)
	// }
	// // Verify test connection registered
	// match := false
	// for connected := range connections {
	// 	if connected.EmailConnectId == connection.EmailConnectId && connected.EmailConnect == connection.EmailConnect {
	// 		match = true
	// 		break
	// 	}
	// }
	// if !match {
	// 	t.Fatalf("Test connection %v \n not present in list of connections %v ", connection, connections)
	// }
}
