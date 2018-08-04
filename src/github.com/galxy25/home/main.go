// Package main runs web server and backend services
// for Levi Schoen's digital home: https://www.levi.casa
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	communicator "github.com/galxy25/home/communicator"
	data "github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// --- BEGIN Globals ---

// Initialize environment dependent variables
var toker = os.Getenv("TOKER")

var home_port, _ = strconv.Atoi(os.Getenv("CASA_PORT"))

// File path where desired connections data is stored
var DESIRED_CONNECTIONS_FILEPATH = os.Getenv("DESIRED_CONNECTIONS_FILEPATH")

// File path where current connection data is stored
var CURRENT_CONNECTIONS_FILEPATH = os.Getenv("CURRENT_CONNECTIONS_FILEPATH")

var newConnectionsQueue = make(chan *data.EmailConnect)

// Endpoint represents an HTTP endpoint
// exposed and serviced by home
type Endpoint struct {
	Path, Verb string
}

// Response represents an HTTP response
// returned by a call to a home endpoint
type Response struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
	Json       string `json:"json"`
}

// Affirmative response to a health check
const HEALTH_CHECK_OK = "pong"

// Purposes and paths of
// http endpoints
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

// Package logging context
var package_logger = log.WithFields(log.Fields{
	"package": "home",
	"file":    "main.go",
})

// --- END Globals ---

// --- BEGIN INIT ---
// init configures:
//   Project level logging
//     Format: JSON
// 	     Timestamp: RFC3339Nano
//     Output: os.Stdout
//     Level:  INFO
func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	// Output to stdout instead of the default stderr
	// N.B.: Could be any io.Writer
	log.SetOutput(os.Stdout)
	// Only log at level INFO
	log.SetLevel(log.InfoLevel)
}

// --- END INIT ---

// --- BEGIN Library ---
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

// connect handles a clients
// request to connect
func connect(w http.ResponseWriter, r *http.Request) {
	// Record the time this connection was initiated
	// 🤔 hmmm maybe the client should set and send this?
	do_connect_timestamp := time.Now()
	do_connect_epoch := do_connect_timestamp.Unix()
	// Blindly decode the request
	// as an email connection
	var email_connection data.EmailConnect
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
	email_connection.ReceiveEpoch = strconv.Itoa(int(do_connect_epoch))

	email_connection_data := email_connection.ToString()
	// Persist desired connection
	_, err = in_file.WriteString(email_connection_data)
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource":           "io/file",
			"executor":           "#connect",
			"command_parameters": email_connection_data,
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

// inbox returns the list of current connections
func inbox(w http.ResponseWriter, r *http.Request) {
	var connections data.Connections
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
		connection, err := data.EmailConnectFromString(connection_scanner.Text())
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

// jsonLoggingHandler wraps an HTTP handler and logs
// the request, de-serializing the body as JSON
func jsonLoggingHandler(h http.Handler) http.Handler {
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

// --- END Library ---

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
		done := make(chan struct{})
		// Where the buffer size is how far ahead
		// we will allow the publishers to outrun
		// the consumers of this channel
		work := make(chan *data.EmailConnect, 10)
		connected := make(chan *data.EmailConnect, 10)
		defer close(done)
		defer close(connected)
		defer close(work)
		for {
			// Sweep!
			//  for each desired connection in the input file
			//    verify it is not in the output file
			//    if it is
			//      remove it from the input file
			//    else
			//      leave it in the input file
			communicator.SweepConnections(DESIRED_CONNECTIONS_FILEPATH, CURRENT_CONNECTIONS_FILEPATH, done, connected)
			// Connect!
			// for each desired connection in the input file
			//  attempt to make the connection
			//  if connection successful
			//      write it to the output file
			//  else
			//    no-op
			//    (☝🏾 will get reconciled on the next loop)
			communicator.Connect(DESIRED_CONNECTIONS_FILEPATH, CURRENT_CONNECTIONS_FILEPATH, connected, newConnectionsQueue)
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
		}
	}()
	// Engage and Segment audience
	// go func() {
	//      for {
	//          go viewer.MapAudience(CURRENT_CONNECTIONS_FILEPATH)
	//          go viewer.SegmentAudience(CURRENT_CONNECTIONS_FILEPATH)
	//      }
	// }()
	// Run the web service for interested clients of levi.casa
	err := http.ListenAndServe(fmt.Sprintf(":%v", home_port), jsonLoggingHandler(httpd))
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/port",
			"executor": "#main",
			"port":     home_port,
		}).Fatal("Failed to run HTTP server")
		panic(err)
	}
}
