// Package main runs web server and backend services
// for Levi Schoen's digital home: https://www.levi.casa
package main

// yada yada yada, #def you
import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	tell "github.com/galxy25/levishouse/tell"
	xip "github.com/galxy25/levishouse/xip"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// --- BEGIN Globals ---
// Default anonymous user token, e.g a token's antonym
const ANON_TOKER = "antonym"

// Affirmative response to a health check
const HEALTH_CHECK_OK = "pong"

// List of HTTP endpoints exposed
// via levishouse
var ENDPOINTS = map[string]Endpoint{
	"BASE": Endpoint{
		Path: "/",
		Verb: "GET"},
	"HEALTH": Endpoint{
		Path: "/ping",
		Verb: "GET"},
	"CONNECT": Endpoint{
		Path: "/connect",
		Verb: "POST"},
	"INBOX": Endpoint{
		Path: "/inbox",
		Verb: "GET"},
}

// IsEmpty returns bool as to whether string s is empty.
// TODO: extract => levisutils
func IsEmpty(s *string) (empty bool) {
	return *s == ""
}

// --- END Globals ---
// --- BEGIN INIT ---
// Initialize environment dependent variables
var toker = os.Getenv("TOKER")

var home_port, _ = strconv.Atoi(os.Getenv("CASA_PORT"))

// File path where desired connections data is stored
var DESIRED_CONNECTIONS_FILEPATH = os.Getenv("DESIRED_CONNECTIONS_FILEPATH")

// File path where current connection data is stored
var CURRENT_CONNECTIONS_FILEPATH = os.Getenv("CURRENT_CONNECTIONS_FILEPATH")

// Purposes and paths of exposed endpoints

// Configure package logging context
var package_logger = log.WithFields(log.Fields{
	"package": "levishouse",
	"file":    "main.go",
})

// init configures:
//   Project level logging
//     Format: JSON
//     Output: os.Stdout
//     Level:  INFO
func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
	// Output to stdout instead of the default stderr
	// N.B.: Could be any io.Writer
	log.SetOutput(os.Stdout)
	// Only log the info severity or above.
	log.SetLevel(log.InfoLevel)
}

// --- END INIT ---
// --- BEGIN Data ---
// TODO: Investigate if there is a way
// to codegen this client/server boilerplate
// swagger mayhaps?

// Endpoint represents an HTTP endpoint
// exposed and serviced by levishouse
type Endpoint struct {
	Path, Verb string
}

// Response represents an HTTP response
// returned by a call to a levishouse endpoint
type Response struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
	Json       string `json:"json"`
}

// --- END Data ---
// ping returns pong
// HTTP health check handler
func ping(w http.ResponseWriter, r *http.Request) {
	// Let the interested party know
	// we're still
	// alive and kicking...it
	// TODO: Implement a real health check
	response := &Response{
		Message:    HEALTH_CHECK_OK,
		StatusCode: http.StatusOK}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTeapot)
	json.NewEncoder(w).Encode(response)
}

// connect processes a clients
// http request to connect
// over email, SMS, chat
func connect(w http.ResponseWriter, r *http.Request) {
	// Record the time this connection was initiated
	// 🤔 hmmm maybe the client should set and send this?
	do_connect_timestamp := time.Now()
	do_connect_epoch := do_connect_timestamp.Unix()
	// Blindly decode the request
	// as an email connection
	var email_connection xip.EmailConnect
	json.NewDecoder(r.Body).Decode(&email_connection)
	// Acquire connection publishing appendix
	in_file, err := os.OpenFile(DESIRED_CONNECTIONS_FILEPATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer in_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#connect",
		}).Fatal(fmt.Sprintf("Failed to open %v", DESIRED_CONNECTIONS_FILEPATH))
		panic(err)
	}
	// extract to xip.EmailConnect interface
	// along with inline/ugly toStringing() below
	if IsEmpty(&email_connection.EmailConnectId) {
		email_connection.EmailConnectId = ANON_TOKER
	}
	email_connection.ReceiveEpoch = strconv.Itoa(int(do_connect_epoch))
	encoded_message := base64.StdEncoding.EncodeToString([]byte(email_connection.EmailConnect))
	email_connection_xip := fmt.Sprintf("%v:%v %v %t %v\n", email_connection.EmailConnectId,
		toker,
		encoded_message,
		email_connection.SubscribeToMailingList,
		email_connection.ReceiveEpoch)
	// Persist desired connection
	_, err = in_file.WriteString(email_connection_xip)
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource":           "io/file",
			"executor":           "#connect",
			"command_parameters": email_connection_xip,
			"io_name":            DESIRED_CONNECTIONS_FILEPATH,
		}).Fatal("Failed to persist desired connection")
		panic(err)
	}
	// Return to the user success in
	// persisting the desired connection
	response := &Response{
		Message:    "Connection initiated",
		StatusCode: http.StatusAccepted}
	response_bytes, _ := json.Marshal(email_connection)
	response.Json = string(response_bytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// inbox returns an inbox of current connections
func inbox(w http.ResponseWriter, r *http.Request) {
	var connections xip.Connections
	// Open current connections list
	connection_file, err := os.OpenFile(CURRENT_CONNECTIONS_FILEPATH, os.O_CREATE|os.O_RDONLY, 0644)
	defer connection_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#inbox",
		}).Fatal(fmt.Sprintf("Failed to open %v", CURRENT_CONNECTIONS_FILEPATH))
		panic(err)
	}
	// Iterate over each connection and add to response
	connection_scanner := bufio.NewScanner(connection_file)
	connection_scanner.Split(bufio.ScanLines)
	for connection_scanner.Scan() {
		connection, err := xip.EmailConnectFromString(connection_scanner.Text())
		if err != nil {
			continue
		}
		connections.EmailConnections = append(connections.EmailConnections, *connection)
	}
	// Return to the user all current connections
	response := &Response{
		Message:    "Current connections",
		StatusCode: http.StatusOK}
	w.Header().Set("Content-Type", "application/json")
	response_bytes, err := json.Marshal(connections)
	response.Json = string(response_bytes)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// loggingHandler wraps an HTTP handler and logs
// the request, blindly de-serializing the body as JSON
func loggingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request_body interface{}
		json.NewDecoder(r.Body).Decode(&request_body)
		package_logger.WithFields(log.Fields{
			"request_method":    r.Method,
			"request_uri":       r.RequestURI,
			"requester_address": r.RemoteAddr,
			"requester_host":    r.Host,
			"request_body":      request_body,
		}).Info("levi.casa")
		// And now set a new body,
		// which replicates the same data we read:
		json_bytes := new(bytes.Buffer)
		json.NewEncoder(json_bytes).Encode(request_body)
		r.Body = ioutil.NopCloser(json_bytes)
		h.ServeHTTP(w, r)
	})
}

// main
//  runs the web service
//  runs the connection reconciliation service
//  runs the audience service
// TODO: move each service into separate binary & docker image
func main() {
	httpd := http.NewServeMux()
	// Serve web files in the static directory
	httpd.Handle(ENDPOINTS["BASE"].Path, http.FileServer(http.Dir("./static")))
	// Expose a health check endpoint
	httpd.HandleFunc(ENDPOINTS["HEALTH"].Path, ping)
	// Expose an endpoint for connect requests
	httpd.HandleFunc(ENDPOINTS["CONNECT"].Path, connect)
	// Expose an endpoint for inbox requests
	httpd.HandleFunc(ENDPOINTS["INBOX"].Path, inbox)
	// Start the connection reconciliation service
	// for ensuring
	// with the monotonic progression of time and loop iterations
	// that all
	// desired connections
	// become
	// realized connections
	// TODO extract into separate cmd,binary, and docker image
	go func() {
		for {
			// Sweep!
			//  for each desired connection in the input file
			//    verify it is not in the output file
			//    if it is
			//      remove it from the input file
			//    else
			//      query the output event log for desired connection
			//        if attempted and successful
			//              write it to the output file
			//        else
			//           leave it in the input file
			tell.SweepConnections(DESIRED_CONNECTIONS_FILEPATH, CURRENT_CONNECTIONS_FILEPATH)
			// Connect!
			// for each desired connection in the input file
			//  attempt to make the connection
			//  if connection successful
			//      write it to the output file
			//  else
			//    no-op
			//    (☝🏾 will get reconciled on the next loop)
			tell.Connect(DESIRED_CONNECTIONS_FILEPATH, CURRENT_CONNECTIONS_FILEPATH)
			// N.B. could do both Sweep! and Connect! concurrently
			// but for frugality and programming for the
			// average case doing them sequentially is correct
			// as it avoids API calls
			// to check on the status of connections that are in progress
			// N.B. an in memory implementation of this should
			// use streaming queries and channels
			// versus persistent(e.g. stale) files
			// to map and communicate connection state
			// but current cost model favors
			// cheap local storage
			// over
			// metered requests and bandwidth
			// TODO: Add a bounded
			// 60s back off sleep here
		}
	}()
	// Engage and Segment audience
	// go func() {
	//      for {
	//          go show.MapAudience(CURRENT_CONNECTIONS_FILEPATH)
	//          go show.SegmentAudience(CURRENT_CONNECTIONS_FILEPATH)
	//      }
	// }()
	// Run the web service for interested clients of levi.casa
	err := http.ListenAndServe(fmt.Sprintf(":%v", home_port), loggingHandler(httpd))
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/port",
			"executor": "#main",
			"port":     home_port,
		}).Fatal("Failed to run HTTP server")
		panic(err)
	}
}
