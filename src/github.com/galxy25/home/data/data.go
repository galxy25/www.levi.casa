// Package data contains common elements
// and eXecute In Place data structures
// for www.levi.casa servers
package data

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Default anonymous user token, e.g a token's antonym
const ANON_TOKER = "antonym"

var toker = os.Getenv("TOKER")

// IsEmpty returns bool as to whether string s is empty.
// TODO: extract => levisutils
func isEmpty(s *string) (empty bool) {
	return *s == ""
}

// EmailConnect is the data for a
// single email connection
type EmailConnect struct {
	// Contents of the email message
	EmailConnect string `json:"email_connect"`
	// Address of the sender
	EmailConnectId string `json:"email_connect_id"`
	// Whether the sender would like to auto-receive
	// email connections related to this connection
	SubscribeToMailingList bool `json:"subscribe_to_mailing_list"`
	// Time message was received from the sender
	ReceiveEpoch string `json:"receive_epoch"`
	// Time message was sent to the receiver
	ConnectEpoch string `json:"connect_epoch"`
}

func (e *EmailConnect) BaseString() (stringy string) {
	encoded_message := base64.StdEncoding.EncodeToString([]byte(e.EmailConnect))
	if isEmpty(&e.EmailConnectId) {
		stringy = fmt.Sprintf("%v:%v %v %t %v", ANON_TOKER,
			toker,
			encoded_message,
			e.SubscribeToMailingList,
			e.ReceiveEpoch)
	} else {
		stringy = fmt.Sprintf("%v:%v %v %t %v", e.EmailConnectId,
			toker,
			encoded_message,
			e.SubscribeToMailingList,
			e.ReceiveEpoch)
	}
	return stringy
}

func (e *EmailConnect) ToString() (stringy string) {
	base := e.BaseString()
	// We really care about whitespace because
	// we're matching off of the string via grep
	if isEmpty(&e.ConnectEpoch) {
		stringy = fmt.Sprintf("%v\n", base)
	} else {
		stringy = fmt.Sprintf("%v %v\n", base, e.ConnectEpoch)
	}
	return stringy
}

// Matches returns bool indicating whether the current
// connection matches the specified connection
// Matches on 3-tuple of content, sender, send time
func (e *EmailConnect) Matches(other *EmailConnect) (match bool) {
	match = e.EmailConnect == other.EmailConnect && e.EmailConnectId == other.EmailConnectId && e.ReceiveEpoch == other.ReceiveEpoch
	return match
}

func (e *EmailConnect) ExistsInFile(file string) (exists bool) {
	// 🙏🏾🙏🏾🙏🏾
	// https://nathanleclaire.com/blog/2014/12/29/shelled-out-commands-in-golang/
	// search current connections for matching desired connection
	cmdName := "grep"
	cmdArgs := []string{"-iw", e.ToString(), file}
	// i.e. grep -iw "here@go.com:sky SGVyZQ== false 1529615331" current_connections.txt
	// -w https://unix.stackexchange.com/questions/206903/match-exact-string-using-grep
	cmd := exec.Command(cmdName, cmdArgs...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating StdoutPipe for Cmd: %v\n", err)
		return exists
	}
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Error starting Cmd: %v\n", err)
		return exists
	}
	cmd_scanner := bufio.NewScanner(cmdReader)
	cmd_scanner.Split(bufio.ScanLines)
	// Non-nil match was found for desired connection in file
	for cmd_scanner.Scan() && !exists {
		current_connection := cmd_scanner.Text()
		connection_current, err := EmailConnectFromString(current_connection)
		if err != nil {
			fmt.Println("Failed to convert persisted connection to struct")
			return exists
		}
		exists = e.Matches(connection_current)
	}
	// Wait waits until the grep command
	// for a matching desired and current connection
	// finishes cleanly and ensures
	// closure of any pipes siphoning from it's output.
	err = cmd.Wait()
	if err != nil {
		// Ignoring as grep returns non-zero if no match found
	}
	return exists
}

func EmailConnectFromString(raw string) (email_connection *EmailConnect, err error) {
	// Read persisted connection
	// into an EmailConnection interface
	// TODO, add this as an interface method
	persisted_connection := strings.Split(raw, " ")
	// HACKs to stop runtime panics
	// due to blindly reading any garbage line in desired
	// as a valid connection
	// 4 because we expect connections to be serialized
	// according to the data.EmailConnect struct field order.
	if len(persisted_connection) < 4 {
		return email_connection, errors.New(fmt.Sprintf("Invalid persisted connection: %v", persisted_connection))
	}
	decoded_message, _ := base64.StdEncoding.DecodeString(persisted_connection[1])
	mailing_list_subscriber, _ := strconv.ParseBool(persisted_connection[2])
	email_connection = &EmailConnect{
		EmailConnectId:         strings.Split(persisted_connection[0], fmt.Sprintf(":%v", toker))[0],
		EmailConnect:           string(decoded_message),
		SubscribeToMailingList: mailing_list_subscriber,
		ReceiveEpoch:           persisted_connection[3]}
	if len(persisted_connection) > 4 {
		email_connection.ConnectEpoch = persisted_connection[4]
	}
	return email_connection, err
}

// Connections is the data for
// collections of EmailConnect's
type Connections struct {
	EmailConnections []EmailConnect `json:"email_connections"`
}

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
