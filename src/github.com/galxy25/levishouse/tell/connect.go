// Package tell provides the ability to send
// e-mail, SMS, or chat messages
// over HTTP via
//      AWS Simple Notification Service
package tell

// yada yada yada, #def you
import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	xip "github.com/galxy25/levishouse/xip"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- BEGIN Globals ---
const (
	// Number of desired connections to buffer for connecting
	CONNECTION_BUFFER_SIZE = 64
	// Number of go-routines to run for handling connections
	CONNECTION_SOCKET_POOL_SIZE = 4
)

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
func SweepConnections(desired_connections string, current_connections string, done <-chan struct{}, connected <-chan string) {
	// Get iterator on desired_connections
	input_file, err := os.OpenFile(desired_connections, os.O_RDONLY|os.O_CREATE, 0644)
	defer input_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#SweepConnections",
			"error":    err,
		}).Fatal(fmt.Sprintf("Failed to open %v", desired_connections))
		panic(err)
	}
	// Iterate over desired_connections
	input_scanner := bufio.NewScanner(input_file)
	input_scanner.Split(bufio.ScanLines)
	for input_scanner.Scan() {
		// Check to see if iterated desired connection
		// is in the list of current connections
		desired_connection := input_scanner.Text()
		connection_desired, err := xip.EmailConnectFromString(desired_connection)
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
		sweep_err := sweepConnection(desired_connection, desired_connections)
		if sweep_err == nil {
			package_logger.WithFields(log.Fields{
				"swept":      desired_connection,
				"swept_from": desired_connections,
				"executor":   "#SweepConnections",
			}).Info("Sweeped!")
		}
	}
	// Kick off a go-routine
	// to sweep new connections as they occur
	go func() {
		for {
			select {
			case <-done:
				return
			case made_connection, more := <-connected:
				if more {
					// Sweep the made connection
					sweep_err := sweepConnection(made_connection, desired_connections)
					if sweep_err == nil {
						package_logger.WithFields(log.Fields{
							"swept":      made_connection,
							"swept_from": desired_connections,
							"executor":   "#SweepConnections",
						}).Info("Sweeped!")
					}
				}
			}
		}
	}()
}

// sweepConnection sweeps a connection from a file
// returning error if any
func sweepConnection(connection string, sweep_file string) (err error) {
	sed_command := "sed"
	// Sanitize input for sed by
	// removing whitespace and lazily
	// escaping sed regex characters
	// Put a backslash before $.*/[\]^
	// and only those characters
	// https://unix.stackexchange.com/questions/32907/what-characters-do-i-need-to-escape-when-using-sed-in-a-sh-script
	// because I'm "encoding" messages
	// into base64
	// https://en.wikipedia.org/wiki/Base64
	// HACK: Only doing '/' for now
	// TODO: Escape all of $.*/[\]^
	sed_safe_connection := strings.TrimSpace(strings.
		Replace(connection, "/", "\\/", -1))
	sed_sweep_args := []string{"-i", fmt.Sprintf("s/%v//g", sed_safe_connection), sweep_file}
	if runtime.GOOS == "darwin" {
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

// Connect sends messages from one person to another
// for each desired connection in the input file
//      attempt to make the connection
//      if connection successful
//          write it to the output file
//      else
//          no-op
//          (☝🏾 will get reconciled on the next loop)
func Connect(desired_connections string, current_connections string) {
	var wait_group sync.WaitGroup
	// Set up a done channel that's shared by the whole pipeline,
	// and close that channel when this pipeline exits, as a signal
	// for all the drain goroutines we start to exit.
	flush := make(chan struct{})
	defer close(flush)
	// Buffered channel which all persisted connections
	// flow through for
	//     persisting the current state of the socket
	socket_sink := make(chan *xip.EmailConnect, CONNECTION_SOCKET_POOL_SIZE)
	defer close(socket_sink)
	// Buffered channel which all desired connections
	// flow through for
	//     making the connection
	socket_drain := make(chan *xip.EmailConnect, CONNECTION_SOCKET_POOL_SIZE)
	defer close(socket_drain)
	// pipeline, sync, wait, close, defer all the channels
	// Create reader for persisted connection state
	input_file, err := os.Open(desired_connections)
	defer input_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#Connect",
		}).Fatal(fmt.Sprintf("Failed to open %v", desired_connections))
		panic(err)
	}
	input_scanner := bufio.NewScanner(input_file)
	input_scanner.Split(bufio.ScanLines)
	// Iterate over each desired connection
	// and attempt to make the connection
	for input_scanner.Scan() {
		current_line := input_scanner.Text()
		if current_line == "" {
			continue
		} else {
			connection, err := xip.EmailConnectFromString(current_line)
			if err != nil {
				continue
			}
			package_logger.WithFields(log.Fields{
				"executor": "#Connect",
				"resource": "io/channel",
				"io":       fmt.Sprintf("%v", socket_drain),
				"io_value": connection,
			}).Info("Sending email connection to channel for processing")
			// Pass this connection to the
			// buffered connection handler
			go doEmailConnect(connection, current_connections, &wait_group)
			wait_group.Add(1)
		}
	}
	// chronically cellularize TPS
	// HACK: Sleep 1-5 minutes
	// n number stream consumers
	// TODO: batch and rate limited connection endpoint
	// (cron + uniq + |)
	// or
	// (cloudwatchevents + lambda + dynamo)
	// versus naive clock limited TPS impl
	wait_group.Wait()
}

// --- END Globals ---

// --- BEGIN INIT ---
// Configure package logging context
var package_logger = log.WithFields(log.Fields{
	"package": "levishouse/tell",
	"file":    "connect.go",
})

// Initialize environment dependent variables
var toker = os.Getenv("TOKER")

// --- BEGIN Library ---
// doEmailConnect makes email connections by sending me an email via AWS SNS
func doEmailConnect(email_connection *xip.EmailConnect, current_connections string, wait_group *sync.WaitGroup) {
	defer wait_group.Done()
	package_logger.WithFields(log.Fields{
		"executor":           "#doEmailConnect",
		"command_parameters": email_connection,
	}).Info("Processing email connection to initiate")
	message := email_connection.EmailConnect
	sess := session.Must(session.NewSession())
	svc := sns.New(sess)
	params := &sns.PublishInput{
		Message:  aws.String(message),
		TopicArn: aws.String("arn:aws:sns:us-west-2:540120437916:www_levi_casa"),
	}
	resp, err := svc.Publish(params)
	if err != nil {
		// Cast err to awserr.Error
		// to get the Code and Message from an error.
		package_logger.WithFields(log.Fields{
			"executor":           "#doEmailConnect.sns.#Publish",
			"command_parameters": params,
			"error":              err.Error(),
		}).Error("Error making email connection")
		return
	} else {
		// Record the time this connection was made
		so_connect_timestamp := time.Now()
		so_connect_epoch := strconv.Itoa(int(so_connect_timestamp.Unix()))
		email_connection.ConnectEpoch = so_connect_epoch
		// Send processed email connection to
		// next stage in pipeline
		package_logger.WithFields(log.Fields{
			"executor":           "#doEmailConnect.sns.#Publish",
			"command_parameters": email_connection,
			"command_response":   resp,
		}).Info("Successfully made email connection")
		soEmailConnect(email_connection, current_connections)
	}
}

// soEmailConnect does the things we do after an email connection
// answering the question: "You made an email connection. So what?"
func soEmailConnect(current_connection *xip.EmailConnect, current_connections string) {
	// Acquire connection publishing appendix
	in_file, err := os.OpenFile(current_connections, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer in_file.Close()
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#soEmailConnect",
			"io_name":  current_connections,
		}).Fatal("Failed to open file")
		panic(err)
	}
	package_logger.WithFields(log.Fields{
		"resource": current_connection,
		"executor": "#soEmailConnect",
	}).Info("Processing email connection")
	// extract to xip.EmailConnect interface
	// along with inline/ugly toStringing() below
	encoded_message := base64.StdEncoding.EncodeToString([]byte(current_connection.EmailConnect))
	email_connection_xip := fmt.Sprintf("%v:%v %v %t %v %v\n", current_connection.EmailConnectId,
		toker,
		encoded_message,
		current_connection.SubscribeToMailingList,
		current_connection.ReceiveEpoch,
		current_connection.ConnectEpoch)
	// Persist desired connection
	_, err = in_file.WriteString(email_connection_xip)
	if err != nil {
		package_logger.WithFields(log.Fields{
			"resource":           "io/file",
			"executor":           "#soEmailConnect",
			"command_parameters": email_connection_xip,
			"io_name":            current_connections,
		}).Fatal("Failed to persist current email connection")
		panic(err)
	}
	package_logger.WithFields(log.Fields{
		"resource":           "io/file",
		"executor":           "#soEmailConnect",
		"command_parameters": email_connection_xip,
		"io_name":            current_connections,
	}).Info("Successfully persisted current email connection")
}

// --- END Library ---
