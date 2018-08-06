// Package communicator provides the ability to send
// e-mail, SMS, or chat messages
// over HTTP via
//      AWS Simple Notification Service
package communicator

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	data "github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// --- BEGIN INIT ---

// Initialize environment dependent variables
var toker = os.Getenv("TOKER")
var sns_topic_arn = os.Getenv("HOME_SNS_TOPIC")

// Configure package logging context
var package_logger = log.WithFields(log.Fields{
	"package": "home/communicator",
	"file":    "connect.go",
})

// Sigh, only a "variable" so we can stub it in tests
var sns_publisher = func(message string) (resp interface{}, err error) {
	sess := session.Must(session.NewSession())
	svc := sns.New(sess)
	params := &sns.PublishInput{
		Message:  aws.String(message),
		TopicArn: aws.String(sns_topic_arn),
		Subject:  aws.String("Message to www.levi.casa"),
	}
	resp, err = svc.Publish(params)
	return resp, err
}

// --- END Init ---

// --- BEGIN Globals ---

// SweepConnections lists, verifies, and sets
// the current status of any desired connections via algorithm:
//      for each desired connection in the input file
//          verify it is not in the output file
//      if it is
//          remove it from the input file
// When SweepConnections returns
// all past connections are swept
// and future connections can be swept
// by pushing them to the connected channel
// sweeping will stop when a message is sent on
// the done channel
func SweepConnections(desired_connections string, current_connections string, connected <-chan *data.Connection) {
	// Get iterator on desired_connections
	input_file, err := os.OpenFile(desired_connections, os.O_RDONLY|os.O_CREATE, 0644)
	defer input_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#SweepConnections",
			"error":    err,
		}).Fatal(fmt.Sprintf("Failed to open %v", desired_connections))
	}
	// Iterate over desired_connections
	input_scanner := bufio.NewScanner(input_file)
	input_scanner.Split(bufio.ScanLines)
	for input_scanner.Scan() {
		// Check to see if desired connection
		// is in the list of current connections
		desired_connection := input_scanner.Text()
		connection_desired, err := data.ConnectionFromString(desired_connection)
		if err != nil {
			package_logger.WithFields(log.Fields{
				"connection": desired_connection,
				"executor":   "#SweepConnections",
				"err":        err,
			}).Info("Skipping invalid desired connection")
			continue
		}
		connected := connection_desired.ExistsInFile(current_connections)
		if !connected {
			continue
		}
		// Sweep the made connection
		sweep_err := sweepConnection(connection_desired, desired_connections)
		if sweep_err == nil {
			package_logger.WithFields(log.Fields{
				"swept":      connection_desired,
				"swept_from": desired_connections,
				"executor":   "#SweepConnections",
			}).Info("Swept!")
		}
	}
	// Sweep new connections as they occur
	go sweeperD(desired_connections, connected)
}

// Connect sends messages from one person to another
// for each desired connection in the input file
//      attempt to make the connection
//      if connection successful
//          write it to the output file
//      else
//          no-op
//          (☝🏾 will get reconciled on the next loop)
func Connect(desired_connections string, current_connections string, connected chan<- *data.Connection, newConnections <-chan *data.Connection) {
	// Create reader for persisted connection state
	input_file, err := os.OpenFile(desired_connections, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer input_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#Connect",
			"error":    err,
			"io":       desired_connections,
		}).Fatal("Failed to open file", desired_connections)
	}
	input_scanner := bufio.NewScanner(input_file)
	input_scanner.Split(bufio.ScanLines)
	// Iterate over each desired connection
	// and attempt to make the connection
	for input_scanner.Scan() {
		current_line := input_scanner.Text()
		if current_line == "" {
			package_logger.WithFields(log.Fields{
				"resource": "io/file",
				"io":       desired_connections,
				"executor": "#Connect",
			}).Info("Skipping empty connection")
			continue
		} else {
			connection, err := data.ConnectionFromString(current_line)
			if err != nil {
				package_logger.WithFields(log.Fields{
					"connection": current_line,
					"executor":   "#Connect",
					"error":      err,
				}).Error("Unable to de-serialize desired connection")
				continue
			}
			package_logger.WithFields(log.Fields{
				"executor": "#Connect",
				"resource": "io/channel",
				"io":       connection,
			}).Info("Sending connection to channel for processing")
			// Pass this connection to the
			// buffered connection handler
			go doConnection(connection, current_connections, connected)
		}
	}
	go connectD(current_connections, connected, newConnections)
}

// --- END Globals ---

// --- BEGIN Library ---
func connectD(current_connections string, connected chan<- *data.Connection, newConnections <-chan *data.Connection) {
	for {
		select {
		case newConnection, more := <-newConnections:
			if !more {
				return
			}
			go doConnection(newConnection, current_connections, connected)
		}
	}
}

// sweeperD sweeps made connections from desired_connections as they arrive on the
// connected channel until a message on the done channel is received
func sweeperD(desired_connections string, connected <-chan *data.Connection) {
	for {
		select {
		case made_connection, more := <-connected:
			if !more {
				return
			}
			// Sweep the made connection
			sweep_err := sweepConnection(made_connection, desired_connections)
			if sweep_err == nil {
				package_logger.WithFields(log.Fields{
					"swept":      made_connection,
					"swept_from": desired_connections,
					"executor":   "#SweepConnections",
				}).Info("Swept!")
			}
		}
	}
}

// sweepConnection sweeps a connection from a file
// returning error if any
func sweepConnection(connection *data.Connection, sweep_file string) (err error) {
	sed_command := "sed"
	// Sanitize input for sed by
	// removing whitespace & newlines,
	// and escaping sed regex characters
	// Put a backslash before $.*/[\]^
	// and only those characters
	// https://unix.stackexchange.com/questions/32907/what-characters-do-i-need-to-escape-when-using-sed-in-a-sh-script
	// because I'm "encoding" messages
	// into base64
	// https://en.wikipedia.org/wiki/Base64
	// HACK: Only doing '/' for now
	// TODO: Escape all of $.*/[\]^
	sed_safe_connection := strings.Replace(strings.
		Replace(connection.ToString(), "/", "\\/", -1), "\n", "", -1)
	sed_sweep_args := []string{"-i", fmt.Sprintf("s/%v//g", sed_safe_connection), sweep_file}
	if runtime.GOOS == "darwin" {
		// cuz, BSD ain't GNU
		sed_sweep_args = []string{"-i", "", fmt.Sprintf("s/%v//g", sed_safe_connection), sweep_file}
	}
	//  Remove the realized from the desired
	_, err = exec.Command(sed_command, sed_sweep_args...).Output()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource":           "io/file",
			"io":                 sweep_file,
			"executor":           "#sweepConnection.sed",
			"error":              err,
			"command_parameters": sed_sweep_args,
		}).Error("Failed to sweep connection")
	}
	return err
}

// doConnection makes email connections by
// sending me an email via AWS SNS
func doConnection(email_connection *data.Connection, current_connections string, connected chan<- *data.Connection) {
	package_logger.WithFields(log.Fields{
		"executor":           "#doConnection",
		"command_parameters": email_connection,
	}).Info("Processing email connection to initiate")
	message := email_connection.Connection
	resp, err := sns_publisher(message)
	if err != nil {
		// Cast err to awserr.Error
		// to get the Code and Message from an error.
		package_logger.WithFields(log.Fields{
			"executor":           "#doConnection.#sns_publisher",
			"command_parameters": message,
			"error":              err.Error(),
		}).Error("Error making email connection")
		return
	} else {
		// Record the time this connection was made
		so_connect_timestamp := time.Now()
		so_connect_epoch := so_connect_timestamp.Unix()
		email_connection.ConnectEpoch = so_connect_epoch
		// Send processed email connection to
		// next stage in pipeline
		package_logger.WithFields(log.Fields{
			"executor":           "#doConnection.#sns_publisher",
			"command_parameters": email_connection,
			"command_response":   resp,
		}).Info("Successfully made email connection")
		soConnection(email_connection, current_connections, connected)
	}
}

// soConnection does the things we do after an email connection
// answering the question: "You made an email connection. So what?"
func soConnection(current_connection *data.Connection, current_connections string, connected chan<- *data.Connection) {
	// Acquire connection publishing appendix
	in_file, err := os.OpenFile(current_connections, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer in_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#soConnection",
			"io_name":  current_connections,
		}).Fatal("Failed to open file")
	}
	package_logger.WithFields(log.Fields{
		"resource": current_connection,
		"executor": "#soConnection",
	}).Info("Processing email connection")
	email_connection_data := current_connection.ToString()
	// Persist desired connection
	_, err = in_file.WriteString(email_connection_data)
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource":           "io/file",
			"executor":           "#soConnection",
			"command_parameters": email_connection_data,
			"io_name":            current_connections,
		}).Fatal("Failed to persist current email connection")
	}
	package_logger.WithFields(log.Fields{
		"resource":           "io/file",
		"executor":           "#soConnection",
		"command_parameters": email_connection_data,
		"io_name":            current_connections,
	}).Info("Successfully persisted current email connection")
	connected <- current_connection
}

// --- END Library ---
