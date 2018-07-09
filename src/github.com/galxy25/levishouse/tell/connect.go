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
//      else
//          query the output event log for desired connection
//          if attempted and successful
//              write it to the output file
//          else
//              leave it in the input file
func SweepConnections(desired_connections string, current_connections string) {
	// Read persisted connection state
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
	input_scanner := bufio.NewScanner(input_file)
	input_scanner.Split(bufio.ScanLines)
	// Check to see if each desired connection is
	// in the list of persisted or logged connections
	for input_scanner.Scan() {
		// 🙏🏾🙏🏾🙏🏾
		// https://nathanleclaire.com/blog/2014/12/29/shelled-out-commands-in-golang/
		// search current connections for matching desired connection
		cmdName := "grep"
		cmdArgs := []string{"-iw", input_scanner.Text(), current_connections}
		// i.e. grep -iw "here@go.com:sky SGVyZQ== false 1529615331" current_connections.txt
		// -w https://unix.stackexchange.com/questions/206903/match-exact-string-using-grep
		cmd := exec.Command(cmdName, cmdArgs...)
		cmdReader, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
			os.Exit(1)
		}

		err = cmd.Start()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
			// continue
			os.Exit(1)
		}

		cmd_scanner := bufio.NewScanner(cmdReader)
		cmd_scanner.Split(bufio.ScanLines)
		// If a non-nil match was found for a desired and current connection
		// e.g. we successfully made the connection!
		if cmd_scanner.Scan() && input_scanner.Text() != "" {
			fmt.Printf("Matched: %v | %v \n", input_scanner.Text(), cmd_scanner.Text())
			// XXXX: Brutal hack for greedy matching
			// TODO: Refactor, compose and move guard higher
			// (like into an interface perhaps that can guard against garbage input 🤷🏾‍♂️)
			if input_scanner.Text() != cmd_scanner.Text() {
				greedy_match := true
				// Loosest of validation that
				// connections to compare are functional for comparing
				desired_connection_parts := strings.Split(input_scanner.Text(), " ")
				// 4 because we expect connections to be serialized
				// according to the xip.EmailConnect struct field order.
				if len(desired_connection_parts) < 4 {
					fmt.Println("Skipping desired connection with less than three tokens")
					greedy_match = false
					continue
				}
				current_connection_parts := strings.Split(cmd_scanner.Text(), " ")
				if len(current_connection_parts) < 4 {
					fmt.Println("Skipping current connection with less than three tokens")
					greedy_match = false
					continue
				}
				match_criteria := make(map[int]string)
				// Match on message content, message sender,
				match_criteria[0] = desired_connection_parts[0]
				// message sender,
				match_criteria[1] = desired_connection_parts[1]
				// and message ReceiveEpoch
				match_criteria[3] = desired_connection_parts[3]
				for key, value := range match_criteria {
					if current_connection_parts[key] != value {
						fmt.Println("Skipping current connection for non-greedy matching with desired_connection")
						fmt.Printf("Desired: %v | Current: %v\n", value, current_connections[key])
						greedy_match = false
						break
					}
				}
				if !greedy_match {
					fmt.Println("Skipping incomplete match")
					continue
				}
			}
			sed_command := "sed"
			current_connection := input_scanner.Text()
			// Escape sed regex characters
			// lazily:
			// Put a backslash before $.*/[\]^
			// and only those characters
			// https://unix.stackexchange.com/questions/32907/what-characters-do-i-need-to-escape-when-using-sed-in-a-sh-script
			// because I'm "encoding" messages
			// into base64
			// https://en.wikipedia.org/wiki/Base64
			// HACK:
			// only doing '/' for now
			sed_safe_connection := strings.
				Replace(current_connection, "/", "\\/", -1)
			sed_sweep_args := []string{"-i", fmt.Sprintf("s/%v//g", sed_safe_connection), desired_connections}
			//  Remove the realized from the desired
			_, sweep_err := exec.Command(sed_command, sed_sweep_args...).Output()
			if sweep_err != nil {
				package_logger.WithFields(log.Fields{
					"resource":           "io/file",
					"io":                 desired_connections,
					"executor":           "#SweepConnections.sed",
					"error":              sweep_err,
					"command_parameters": sed_sweep_args,
				}).Fatal(fmt.Sprintf("Failed to sed | %v", desired_connections))
				panic(sweep_err)
			} else {
				package_logger.WithFields(log.Fields{
					"resource":           "io/file",
					"io":                 desired_connections,
					"executor":           "#SweepConnections.sed",
					"command_parameters": sed_sweep_args,
				}).Info(fmt.Sprintf("Sweeped! connection | %v", input_scanner.Text()))
			}
		}
		// Wait waits until the grep command
		// for a matching desired and current connection
		// finishes cleanly and ensures
		// closure of any pipes siphoning from it's output.
		err = cmd.Wait()
		if err != nil {
			// Ignoring as grep returns non-zero if no match found
			fmt.Sprintf("No match for %v | in: %v", input_scanner.Text(), desired_connections)
		}
	}
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
			// Read persisted connection
			// into an EmailConnection interface
			// TODO, add this as an interface method
			persisted_connection := strings.Split(current_line, " ")
			fmt.Printf("Persisted connection: %v \n", persisted_connection)
			// HACKs to stop runtime panics
			// due to blindly reading any garbage line in desired
			// as a valid
			// TODO: Refactor into new Interface error for an EmailConnection{}
			if len(persisted_connection) < 3 {
				continue
			}
			decoded_message, _ := base64.StdEncoding.DecodeString(persisted_connection[1])
			mailing_list_subscriber, _ := strconv.ParseBool(persisted_connection[2])
			connection := xip.EmailConnect{
				EmailConnectId:         strings.Split(persisted_connection[0], ":sky")[0],
				EmailConnect:           string(decoded_message),
				SubscribeToMailingList: mailing_list_subscriber,
				ReceiveEpoch:           persisted_connection[3]}
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
func doEmailConnect(email_connection xip.EmailConnect, current_connections string, wait_group *sync.WaitGroup) {
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
			"command_parameters": email_connection,
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
func soEmailConnect(current_connection xip.EmailConnect, current_connections string) {
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
