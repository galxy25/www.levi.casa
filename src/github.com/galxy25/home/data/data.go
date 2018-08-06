// Package data contains common elements
// and eXecute In Place data structures
// for www.levi.casa servers
package data

import (
	"bufio"
	"encoding/hex"
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

// Connection is the data for a
// single email connection
type Connection struct {
	// Contents of the connection
	Connection string `json:"email_connect"`
	// Address of the sender
	ConnectionId string `json:"email_connect_id"`
	// Whether the sender would like to auto-receive
	// email connections related to this connection
	SubscribeToMailingList bool `json:"subscribe_to_mailing_list"`
	// Time message was received from the sender
	ReceiveEpoch int64 `json:"receive_epoch"`
	// Time message was sent to the receiver
	ConnectEpoch int64 `json:"connect_epoch"`
}

func (e *Connection) baseString() (stringy string) {
	encoded_message := hex.EncodeToString([]byte(e.Connection))
	if e.ConnectionId == "" {
		e.ConnectionId = fmt.Sprintf("%v:%v", ANON_TOKER, toker)
	}
	encodedConnectId := hex.EncodeToString([]byte(e.ConnectionId))
	stringy = fmt.Sprintf("%v:%v %v %t %v", encodedConnectId,
		toker,
		encoded_message,
		e.SubscribeToMailingList,
		e.ReceiveEpoch)
	return stringy
}

func (e *Connection) ToString() (stringy string) {
	base := e.baseString()
	// We really care about whitespace because
	// we're matching off of the string via grep
	stringy = fmt.Sprintf("%v %v\n", base, e.ConnectEpoch)
	return stringy
}

// Matches returns bool indicating whether the current
// connection matches the specified connection
// Matches on 3-tuple of content, sender, send time
func (e *Connection) Matches(other *Connection) (match bool) {
	match = e.Connection == other.Connection && e.ConnectionId == other.ConnectionId && e.ReceiveEpoch == other.ReceiveEpoch
	return match
}

func (e *Connection) ExistsInFile(file string) (exists bool) {
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
		connection_current, err := ConnectionFromString(current_connection)
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

func ConnectionFromString(raw string) (email_connection *Connection, err error) {
	persisted_connection := strings.Split(strings.Replace(raw, "\n", "", -1), " ")
	// 4 because we expect connections to be serialized
	// according to the data.Connection struct field order.
	if len(persisted_connection) < 4 {
		// TODO: Return named error
		return email_connection, errors.New(fmt.Sprintf("Invalid persisted connection: %v", persisted_connection))
	}
	decoded_message, err := hex.DecodeString(persisted_connection[1])
	if err != nil {
		// TODO: Return named error
		return email_connection, err
	}
	mailing_list_subscriber, err := strconv.ParseBool(persisted_connection[2])
	if err != nil {
		// TODO: Return named error
		return email_connection, err
	}
	encoded_sender := strings.Split(persisted_connection[0], fmt.Sprintf(":%v", toker))[0]
	decoded_sender, err := hex.DecodeString(encoded_sender)
	if err != nil {
		// TODO: Return named error
		return email_connection, err
	}
	receiveEpoch, err := strconv.ParseInt(persisted_connection[3], 10, 64)
	if err != nil {
		// TODO: Return named error
		return email_connection, err
	}
	email_connection = &Connection{
		ConnectionId:           string(decoded_sender),
		Connection:             string(decoded_message),
		SubscribeToMailingList: mailing_list_subscriber,
		ReceiveEpoch:           receiveEpoch}
	if len(persisted_connection) > 4 {
		connectEpoch, err := strconv.ParseInt(persisted_connection[4], 10, 64)
		if err != nil {
			// TODO: Return named error
			return email_connection, err
		}
		email_connection.ConnectEpoch = connectEpoch
	}
	return email_connection, err
}

// Connections are an array
// of connections
type Connections struct {
	Connections []Connection `json:"email_connections"`
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
