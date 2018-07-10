package main

import (
	"errors"
	"flag"
	"fmt"
	helper "github.com/galxy25/levishouse/internal/test"
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

type LevishouseTestProcess struct {
	test_context *testing.T //Interface to a `go test` invocation
	pid          int
}

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
	run_cmd := exec.Command("sh", "-c", fmt.Sprintf("make stop run -f %v/Makefile -C %v", *test_bin_dir, *test_bin_dir))
	out, err := run_cmd.CombinedOutput()
	if err != nil {
		l.test_context.Errorf("Failed to run levishouse: %v, %v", string(out), err)
	}
	// XXX: `make run` returns
	// echo "LEVISHOUSE_PID: $$!"
	// as it's last line of output
	l.test_context.Logf("Run output: %v", string(out))
	sliced_output := strings.Split(string(out), "LEVISHOUSE_PID:")
	string_pid := strings.TrimSpace(strings.Split(sliced_output[len(sliced_output)-1], " ")[1])
	l.pid, err = strconv.Atoi(string_pid)
	if err != nil {
		l.test_context.Errorf("Failed to convert %v to int", string_pid)
	}
	l.test_context.Logf("levishouse pid: %v", l.pid)
	return err
}

func (l *LevishouseTestProcess) Stop() (err error) {
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
		l.test_context.Error("kill -s 0 on levishouse returned non nil, unable to stop non-running server.")
	}
	stop_cmd := exec.Command("sh", "-c", fmt.Sprintf("make stop -f %v/Makefile -C %v", *test_bin_dir, *test_bin_dir))
	out, err := stop_cmd.CombinedOutput()
	if err != nil {
		l.test_context.Errorf("Failed to stop levishouse: %v, %v", string(out), err)
	}
	l.test_context.Logf("Stop output: %v", string(out))
	stop_response := levishouse.Signal(syscall.Signal(0))
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

func (l *LevishouseTestProcess) HealthCheck() (healthy bool, err error) {
	// Call the health endpoint to verify
	// it is running
	resp, err := http.Get("http://127.0.0.1:8081/ping")
	tries := 10
	for err != nil && tries > 1 {
		resp, err = http.Get("http://127.0.0.1:8081/ping")
		tries--
		l.test_context.Logf("Health check retries left: %v", tries)
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		l.test_context.Errorf("Unable to ping levishouse: %v \n", err)
	}
	ping_resp, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		l.test_context.Errorf("Unable to parse ping response: %v \n", resp)
	}
	l.test_context.Logf("Ping response from levishouse: %v \n", string(ping_resp))
	if string(ping_resp) != "pong" {
		healthy = false
	} else {
		healthy = true
	}
	return healthy, err
}

func TestItRunsAndStops(t *testing.T) {
	t.Log("Testing everything works...")
	test_house := LevishouseTestProcess{test_context: t}
	house_under_test, err := helper.ExecuteTestProcess(&test_house)
	if err != nil {
		t.Fatalf("Failed to run levishouse: %v\n", err)
	}
	healthy, err := house_under_test.VerifyHealthy()
	if !healthy {
		defer house_under_test.VerifiedStoped()
		t.Fatalf("Got non pong response from ping endpoint: %v \n", err)
	}
	stop_error := house_under_test.VerifiedStoped()
	if stop_error != nil {
		t.Fatalf("Failed to stop levishouse: %v\n ", stop_error)
	}
}
