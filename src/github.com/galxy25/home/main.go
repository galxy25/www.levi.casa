// Package main runs web server and backend services
// for Levi Schoen's digital home: https://www.levi.casa
package main

import (
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

// Communicator used to receive and send
// connections
var comm = communicator.NewCommunicator(desiredConnectionsFilePath, currentConnectionsFilePath)

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
// HTTP Endpoints
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
	"METRICS": Endpoint{
		Path: "/stats",
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

// init main configures:
//   Project level logging settings:
//     Format: JSON
// 	     Timestamp: RFC3339Nano
//     Output: os.Stdout
//     Level:  INFO
func init() {
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// ping is the HTTP handler for the health check endpoint,
// returning HealthCheckOk if home is not on 🔥.
func ping(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement a real health check, i.e. connection queue size
	response := &Response{
		Message:    HealthCheckOk,
		StatusCode: http.StatusOK}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// connect handles a clients request to connect.
func connect(w http.ResponseWriter, r *http.Request) {
	connectTimestamp := time.Now()
	connectEpoch := connectTimestamp.Unix()
	var connection *data.Connection
	err := json.NewDecoder(r.Body).Decode(&connection)
	if err != nil {
		response := &Response{
			Message:    "Invalid connection sent",
			StatusCode: http.StatusBadRequest}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	if len(connection.Message) == 0 {
		errorResponse(w, "Connection must have message content", nil, http.StatusBadRequest)
		return
	}
	connection.ReceiveEpoch = connectEpoch
	err = comm.Record(connection)
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"executor":   "#connect#Communicator.#Record",
			"connection": connection,
			"error":      err,
		}).Error("failed to record new connection")
		response := &Response{
			Message:    "Error processing connection",
			StatusCode: http.StatusInternalServerError}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
	}
	go func() {
		// TODO: keep count of number of in flight connections
		connected, err := comm.Link(connection)
		// TODO: record link latency
		if err != nil {
			packageLogger.WithFields(log.Fields{
				"executor":   "#Communicator.#Link",
				"connection": connection,
				"error":      err,
			}).Error("failed to link new connection")
		}
		packageLogger.WithFields(log.Fields{
			"executor":   "#Communicator.#Link",
			"connection": connected,
		}).Info("linked connection")
	}()
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
	stop := make(chan struct{})
	defer close(stop)
	currentConnections, err := comm.Sent(stop)
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"executor": "#inbox",
			"error":    err,
		}).Error("failed to generate inbox")
		return
	}
	for connection := range currentConnections {
		connections.Connections = append(connections.Connections, connection)
	}
	packageLogger.WithFields(log.Fields{
		"executor": "#inbox",
		"mail":     connections.Connections,
	}).Info(fmt.Sprintf("inbox has %v items", len(connections.Connections)))
	response := &Response{
		Message:    "Current connections",
		StatusCode: http.StatusOK}
	w.Header().Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(connections)
	response.Json = string(responseBytes)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// stats handles HTTP request to the /stats endpoint
// returning statics about the current home process
func stats(w http.ResponseWriter, r *http.Request) {
	metrics := make(map[string]int64)
	var unlinked, linked int64
	stop := make(chan struct{})
	defer close(stop)
	desiredConnections, err := comm.Received(stop)
	if err != nil {
		errorResponse(w, "error trying to count unlinked connections", nil, http.StatusInternalServerError)
		return
	}
	currentConnections, err := comm.Sent(stop)
	if err != nil {
		errorResponse(w, "error trying to count linked connections", nil, http.StatusInternalServerError)
		return
	}
	for _ = range desiredConnections {
		unlinked++
	}
	for _ = range currentConnections {
		linked++
	}
	metrics["unlinked"] = unlinked
	metrics["linked"] = linked
	response := &Response{
		Message:    "dez metrics",
		StatusCode: http.StatusNonAuthoritativeInfo}
	w.Header().Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(metrics)
	response.Json = string(responseBytes)
	w.WriteHeader(http.StatusNonAuthoritativeInfo)
	json.NewEncoder(w).Encode(response)
}

// jsonLoggingHandler wraps an HTTP handler and logs
// the request and de-serialized JSON body
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

// errorResponse constructs and writes
// an HTTP response with the provided
// message, error, and status code
func errorResponse(w http.ResponseWriter, msg string, err error, statusCode int) {
	response := &Response{
		Message:    msg,
		StatusCode: statusCode}
	if err != nil {
		response.Error = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// main runs the web service and background communicator services
// TODO: extract communicator service into separate cmd/binary & docker images
func main() {
	httpd := http.NewServeMux()
	// Serve web files in the static directory
	httpd.Handle(Endpoints["BASE"].Path, http.FileServer(http.Dir("./web")))
	// Expose an endpoint for health check requests
	httpd.HandleFunc(Endpoints["HEALTH"].Path, ping)
	// Expose an endpoint for process metric requests
	httpd.HandleFunc(Endpoints["METRICS"].Path, stats)
	// Expose an endpoint for connect requests
	httpd.HandleFunc(Endpoints["CONNECT"].Path, connect)
	// Expose an endpoint for inbox requests
	httpd.HandleFunc(Endpoints["INBOX"].Path, inbox)
	// If there are any unconnected connections
	// from a previous run, connect them
	go func() {
		reconciled, err := comm.Reconcile()
		if err != nil {
			packageLogger.WithFields(log.Fields{
				"resource":   "communicator",
				"executor":   "#Communicator.#Reconcile",
				"reconciled": reconciled,
				"error":      err,
			}).Panic("failed to reconcile previous connections")
		}
		packageLogger.WithFields(log.Fields{
			"resource":   "communicator",
			"executor":   "#Communicator.#Reconcile",
			"reconciled": reconciled,
		}).Info("reconciled connections")
	}()
	// Run web service for clients
	// of www.levi.casa
	err := http.ListenAndServe(fmt.Sprintf(":%v", homePort), jsonLoggingHandler(httpd))
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/port",
			"executor": "#main",
			"port":     homePort,
		}).Fatal("failed to run HTTP server")
	}
}
