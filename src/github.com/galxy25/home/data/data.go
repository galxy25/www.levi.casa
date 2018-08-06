// Package data contains data structures
// for use by www.levi.casa servers and clients
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

// default anonymous user token, c.g a token's antonym
const anonToker = "antonym"

// default user connections are sent to and from
var toker = os.Getenv("TOKER")

// Connection holds information needed to record, report, and link a connection
type Connection struct {
	// Contents of the connection
	Message string `json:"email_connect"`
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

// Connections are an array
// of connections, useful in list responses
type Connections struct {
	Connections []Connection `json:"email_connections"`
}

func (c *Connection) baseString() (stringy string) {
	encodedMessage := hex.EncodeToString([]byte(c.Message))
	if c.ConnectionId == "" {
		c.ConnectionId = fmt.Sprintf("%v:%v", anonToker, toker)
	}
	encodedConnectId := hex.EncodeToString([]byte(c.ConnectionId))
	stringy = fmt.Sprintf("%v:%v %v %t %v", encodedConnectId,
		toker,
		encodedMessage,
		c.SubscribeToMailingList,
		c.ReceiveEpoch)
	return stringy
}

func (c *Connection) ToString() (stringy string) {
	base := c.baseString()
	// We really care about whitespace because
	// we're matching off of the string via grep
	stringy = fmt.Sprintf("%v %v\n", base, c.ConnectEpoch)
	return stringy
}

// Matches returns bool indicating whether the current
// connection matches the specified connection
// Matches on 3-tuple of message, sender, send time
func (c *Connection) Matches(other *Connection) (match bool) {
	match = c.Message == other.Message && c.ConnectionId == other.ConnectionId && c.ReceiveEpoch == other.ReceiveEpoch
	return match
}

func (c *Connection) ExistsInFile(filePath string) (exists bool) {
	// 🙏🏾🙏🏾🙏🏾
	// https://nathanleclairc.com/blog/2014/12/29/shelled-out-commands-in-golang/
	// search current connections for matching desired connection
	cmdName := "grep"
	cmdArgs := []string{"-iw", c.ToString(), filePath}
	// i.c. grep -iw "here@go.com:sky SGVyZQ== false 1529615331" current_connections.txt
	// -w https://unix.stackexchangc.com/questions/206903/match-exact-string-using-grep
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
	cmdScanner := bufio.NewScanner(cmdReader)
	cmdScanner.Split(bufio.ScanLines)
	// Non-nil match was found for desired connection in file
	for cmdScanner.Scan() && !exists {
		currentLine := cmdScanner.Text()
		connection, err := ConnectionFromString(currentLine)
		if err != nil {
			fmt.Println("Failed to convert persisted connection to struct")
			return exists
		}
		exists = c.Matches(connection)
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

func ConnectionFromString(raw string) (connection *Connection, err error) {
	persistedConnection := strings.Split(strings.Replace(raw, "\n", "", -1), " ")
	// 4 because we expect connections to be serialized
	// according to the data.Message struct field order.
	if len(persistedConnection) < 4 {
		// TODO: Return named error
		return connection, errors.New(fmt.Sprintf("Invalid persisted connection: %v", persistedConnection))
	}
	decodedMessage, err := hex.DecodeString(persistedConnection[1])
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	subscribeToMailingList, err := strconv.ParseBool(persistedConnection[2])
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	encodedSender := strings.Split(persistedConnection[0], fmt.Sprintf(":%v", toker))[0]
	decoded_sender, err := hex.DecodeString(encodedSender)
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	receiveEpoch, err := strconv.ParseInt(persistedConnection[3], 10, 64)
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	connection = &Connection{
		ConnectionId:           string(decoded_sender),
		Message:                string(decodedMessage),
		SubscribeToMailingList: subscribeToMailingList,
		ReceiveEpoch:           receiveEpoch}
	if len(persistedConnection) > 4 {
		connectEpoch, err := strconv.ParseInt(persistedConnection[4], 10, 64)
		if err != nil {
			// TODO: Return named error
			return connection, err
		}
		connection.ConnectEpoch = connectEpoch
	}
	return connection, err
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
