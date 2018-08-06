// Package main runs web server and backend services
// for Levi Schoen's digital home: https://www.levi.casa
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/galxy25/home/communicator"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Affirmative response to a health check
const HealthCheckOk = "pong"

// File path where desired connections data is stored
var desiredConnectionsFilePath = os.Getenv("DESIRED_CONNECTIONS_FILEPATH")

// File path where current connection data is stored
var currentConnectionsFilePath = os.Getenv("CURRENT_CONNECTIONS_FILEPATH")

// Port that web server should listen on
var homePort, _ = strconv.Atoi(os.Getenv("CASA_PORT"))

// In memory storage and signaling
// for new connections to make
var newConnectionsQueue = make(chan *data.Connection)

// Package logging context
var packageLogger = log.WithFields(log.Fields{
	"package": "home",
	"file":    "main.go",
})

// Endpoint represents an HTTP endpoint
// exposed and serviced by home
type Endpoint struct {
	Path, Verb string
}

// Purposes and paths of
// http Endpoints
var Endpoints = map[string]Endpoint{
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

// Response represents an HTTP response
// returned by a call to a home endpoint
type Response struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
	Json       string `json:"json"`
}

// init configures:
//   Project level logging settings:
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

// ping is the http handler for the health check endpoint,
// returning HealthCheckOk if home is not on 🔥.
func ping(w http.ResponseWriter, r *http.Request) {
	// Let the interested party know
	// we're still
	// alive and kicking...it
	// TODO: Implement a real health check
	response := &Response{
		Message:    HealthCheckOk,
		StatusCode: http.StatusOK}
	w.Header().Set("Content-Type", "application/json")
	// for no particular reason
	w.WriteHeader(http.StatusTeapot)
	json.NewEncoder(w).Encode(response)
}

// connect handles a clients request to connect.
func connect(w http.ResponseWriter, r *http.Request) {
	// Record the time this connection was initiated
	// 🤔 hmmm maybe the client should set and send this?
	doConnectTimestamp := time.Now()
	doConnectEpoch := doConnectTimestamp.Unix()
	var connection data.Connection
	err := json.NewDecoder(r.Body).Decode(&connection)
	if err != nil {
		// Return to the user failure in
		// persisting the desired connection
		response := &Response{
			Message:    "Invalid connection sent",
			StatusCode: http.StatusBadRequest}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	// Acquire connection publishing appendix
	desiredConnections, err := os.OpenFile(desiredConnectionsFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer desiredConnections.Close()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#connect",
			"error":    err,
			"io":       desiredConnectionsFilePath,
		}).Fatal("failed to open file")
		// TODO: log level=Error & return 5xx response
	}
	connection.ReceiveEpoch = doConnectEpoch
	connectionData := connection.ToString()
	// Persist desired connection
	_, err = desiredConnections.WriteString(connectionData)
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource":   "io/file",
			"executor":   "#connect",
			"connection": connectionData,
			"io":         desiredConnectionsFilePath,
		}).Fatal("failed to persist desired connection")
		// TODO: log level=Error & return 5xx response
	}
	go func() {
		newConnectionsQueue <- &connection
	}()
	// Return to the user success in
	// persisting the desired connection
	response := &Response{
		Message:    "Connection initiated",
		StatusCode: http.StatusAccepted}
	responseBytes, _ := json.Marshal(connection)
	response.Json = string(responseBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// inbox returns the list of current connections.
func inbox(w http.ResponseWriter, r *http.Request) {
	var connections data.Connections
	// Open current connections list
	currentConnections, err := os.OpenFile(currentConnectionsFilePath, os.O_CREATE|os.O_RDONLY, 0644)
	defer currentConnections.Close()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#inbox",
			"io":       currentConnectionsFilePath,
		}).Fatal("failed to open file")
		// TODO: log level=Error & return 5xx response
	}
	// Iterate over each connection and add to response
	connectionScanner := bufio.NewScanner(currentConnections)
	connectionScanner.Split(bufio.ScanLines)
	for connectionScanner.Scan() {
		connection, err := data.ConnectionFromString(connectionScanner.Text())
		if err != nil {
			continue
		}
		connections.Connections = append(connections.Connections, *connection)
	}
	// Return to the user all current connections
	response := &Response{
		Message:    "Current connections",
		StatusCode: http.StatusOK}
	w.Header().Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(connections)
	response.Json = string(responseBytes)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// jsonLoggingHandler wraps an HTTP handler and logs
// the request and JSON deserialzied body
func jsonLoggingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)
		packageLogger.WithFields(log.Fields{
			"request_method":    r.Method,
			"request_uri":       r.RequestURI,
			"requester_address": r.RemoteAddr,
			"requester_host":    r.Host,
			"request_body":      requestBody,
		}).Info("levi.casa request")
		// Repopulate body with the data read
		jsonBytes := new(bytes.Buffer)
		json.NewEncoder(jsonBytes).Encode(requestBody)
		r.Body = ioutil.NopCloser(jsonBytes)
		h.ServeHTTP(w, r)
	})
}

// main runs the web service and background communicator services
// TODO: move communicator services into separate cmd/binary & docker images
func main() {
	httpd := http.NewServeMux()
	// Serve web files in the static directory
	httpd.Handle(Endpoints["BASE"].Path, http.FileServer(http.Dir("./static")))
	// Expose a health check endpoint
	httpd.HandleFunc(Endpoints["HEALTH"].Path, ping)
	// Expose an endpoint for connect requests
	httpd.HandleFunc(Endpoints["CONNECT"].Path, connect)
	// Expose an endpoint for inbox requests
	httpd.HandleFunc(Endpoints["INBOX"].Path, inbox)
	// Buffered pre-maturely for performance
	madeConnections := make(chan *data.Connection, 10)
	defer close(madeConnections)
	go func() {
		// Both functions spawn go-routines that run for the lifetime of main, and take channels
		// for ensuring new connections are made and swept without having to busy poll
		communicator.SweepConnections(desiredConnectionsFilePath, currentConnectionsFilePath, madeConnections)
		communicator.Connect(desiredConnectionsFilePath, currentConnectionsFilePath, madeConnections, newConnectionsQueue)
	}()
	// Run the web service for interested clients of www.levi.casa
	err := http.ListenAndServe(fmt.Sprintf(":%v", homePort), jsonLoggingHandler(httpd))
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/port",
			"executor": "#main",
			"port":     homePort,
		}).Fatal("failed to run HTTP server")
	}
}
