package main

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func connect(w http.ResponseWriter, r *http.Request) {
	response := &Response{Message: "Connected"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type Response struct {
	Message string `json:"message"`
}

// type EmailConnect struct {
// 	EmailConnect           string `json:"email_connect"`
// 	EmailConnectId         string `json:"email_connect_id"`
// 	SubscribeToMailingList bool   `json:"subscribe_to_mailing_list"`
// }

func loggingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request_body interface{}
		json.NewDecoder(r.Body).Decode(&request_body)
		log.WithFields(log.Fields{
			"request_method":    r.Method,
			"request_uri":       r.RequestURI,
			"requester_address": r.RemoteAddr,
			"requester_host":    r.Host,
			"request_body":      request_body,
		}).Info("Request received by loggingHandler")
		// And now set a new body, which will simulate the same data we read:
		json_bytes := new(bytes.Buffer)
		json.NewEncoder(json_bytes).Encode(request_body)
		r.Body = ioutil.NopCloser(json_bytes)
		h.ServeHTTP(w, r)
	})
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the info severity or above.
	log.SetLevel(log.InfoLevel)
}

func main() {
	httpd := http.NewServeMux()
	// Serve web files in the static directory
	httpd.Handle("/", http.FileServer(http.Dir("./static")))
	// Expose a health check endpoint
	httpd.HandleFunc("/ping", ping)
	// Expose an endpoint for connect requests
	httpd.HandleFunc("/connect", connect)
	if err := http.ListenAndServe(":8081", loggingHandler(httpd)); err != nil {
		panic(err)
	}
}
