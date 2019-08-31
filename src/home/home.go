// Package main runs web server and backend services
// for Levi Schoen's digital home: https://www.levi.casa
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/galxy25/www.levi.casa/home/communicator"
	"github.com/galxy25/www.levi.casa/home/data"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
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

// Directory where tls certs are stored
var tlsCacheDir = os.Getenv("TLS_CACHE_DIR")

// Port that web server should listen on
var homePort, _ = strconv.Atoi(os.Getenv("HOME_PORT"))

// Port to use for making Automatic Certificate Management Environment requests
var acmePort = os.Getenv("ACME_PORT")

// Address that clients can use to find Levi's digital home.
var homeAddress = os.Getenv("HOME_ADDRESS")

// Address for receiving email communications.
var homeEmail = os.Getenv("HOME_EMAIL")

// Phone number for personal telephonic communications.
var homePhone = os.Getenv("HOME_PHONE_NUMBER")

// Universal communicator for receiving and sending connections
var comm = communicator.New(desiredConnectionsFilePath, currentConnectionsFilePath)

// The backend server to reverse proxy requests for home vod assets
var vodReverseProxyURL = os.Getenv("VOD_ORIGIN_SERVER_URL")

// The password required to access the vod server
var vodPassword = os.Getenv("VOD_PASSWORD")

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
	"NEWEMAIL": Endpoint{
		Path: "/email",
		Verb: "POST"},
	"NEWSMS": Endpoint{
		Path: "/sms",
		Verb: "POST"},
	"INBOX": Endpoint{
		Path: "/inbox",
		Verb: "GET"},
	"METRICS": Endpoint{
		Path: "/stats",
		Verb: "GET"},
	"VOD": Endpoint{
		Path: "/vod/",
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

// connect handles clients connection requests
func connect(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
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
		var sender communicator.Sender
		switch r.URL.RequestURI() {
		case "/email":
			connection.Receiver = homeEmail
			sender, err = communicator.EmailFromConnection(connection)
			if err != nil {
				errorResponse(w, err.Error(), err, http.StatusBadRequest)
				return
			}
		case "/sms":
			connection.Receiver = homePhone
			sender, err = communicator.SmsFromConnection(connection)
			if err != nil {
				errorResponse(w, err.Error(), err, http.StatusBadRequest)
				return
			}
		default:
			w.WriteHeader(http.StatusMisdirectedRequest)
			return
		}
		connection.SendEpoch = connectEpoch
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
			return
		}
		go func() {
			// TODO: keep count of number of in flight connections
			connected, err := comm.Link(connection, sender)
			// TODO: record link latency
			if err != nil {
				packageLogger.WithFields(log.Fields{
					"executor":    "#Communicator.#Link",
					"connection":  connection,
					"error":       err,
					"sender_type": fmt.Sprintf("%T", sender),
				}).Error("failed to link new connection")
			} else {
				packageLogger.WithFields(log.Fields{
					"executor":    "#Communicator.#Link",
					"connection":  connected,
					"sender_type": fmt.Sprintf("%T", sender),
				}).Info("linked connection")
			}
		}()
		response := &Response{
			Message:    "Connection initiated",
			StatusCode: http.StatusAccepted}
		responseBytes, _ := json.Marshal(connection)
		response.Json = string(responseBytes)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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

// vodReverseProxy reverse proxies external requests to the vod server endpoint
// `/vod?password=somePassword` if the supplied url query param for password is correct
// (reverse proxy versus separate server because tls & frugality)
func vodReverseProxy(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	passwordPrompt := []byte(fmt.Sprintf("Please supply the correct vod password as a url query param, e.g. https://%s/vod?password=password", homeAddress))
	if len(queryParams["password"]) == 0 {
		log.Printf("vod request with no auth %+v, returning 404\n", r)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(passwordPrompt)
		return
	}
	userPassword := queryParams["password"][0]
	// Not constant time 😬
	if userPassword != vodPassword {
		log.Printf("vod request with invalid auth %+v, returning 404\n", r)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(passwordPrompt)
		return
	}
	// log.Printf("LEVI: %s userPassword")
	rpURL, err := url.Parse(vodReverseProxyURL)
	if err != nil {
		log.Errorf("error %s trying to parse vodReverseProxyURL %s\n", err, vodReverseProxyURL)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(rpURL)
	reverseProxy.ServeHTTP(w, r)
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
	// Serve website based off files in the web directory
	httpd.Handle(Endpoints["BASE"].Path, http.FileServer(http.Dir("./web")))
	// Expose an endpoint for health check requests
	httpd.HandleFunc(Endpoints["HEALTH"].Path, ping)
	// Expose an endpoint for process metric requests
	httpd.HandleFunc(Endpoints["METRICS"].Path, stats)
	// Expose an endpoint for new email requests
	httpd.HandleFunc(Endpoints["NEWEMAIL"].Path, connect)
	// Expose an endpoint for new sms requests
	httpd.HandleFunc(Endpoints["NEWSMS"].Path, connect)
	// Expose an endpoint for inbox requests
	httpd.HandleFunc(Endpoints["INBOX"].Path, inbox)
	// Expose an endpoint for reverse proxying requests to vod media server
	httpd.HandleFunc(Endpoints["VOD"].Path, vodReverseProxy)
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
	// Set up automatic X.509 certificate management
	// via Lets Encrypt.
	// https://goenning.net/2017/11/08/free-and-automated-ssl-certificates-with-go/
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(tlsCacheDir),
		HostPolicy: autocert.HostWhitelist(homeAddress, fmt.Sprintf("www.%v", homeAddress)),
	}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", homePort),
		Handler: jsonLoggingHandler(httpd),
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}
	// Run http server to respond to ACME http-01 challenges
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%v", acmePort), certManager.HTTPHandler(nil))
		packageLogger.WithFields(log.Fields{
			"resource": "io/port",
			"executor": "#main",
			"port":     homePort,
			"error":    err.Error(),
		}).Fatal("failed running ACME http-01 challenge server")

	}()
	// Run web service for clients
	// of https://www.levi.casa
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/port",
			"executor": "#main",
			"port":     homePort,
			"error":    err.Error(),
			"server":   fmt.Sprintf("%+v", server),
		}).Fatal("failed running HTTP server")
	}
}
