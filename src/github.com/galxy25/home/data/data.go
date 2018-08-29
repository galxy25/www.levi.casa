// Package data contains data structures
// for use by www.levi.casa servers and clients
package data

import (
	"encoding/hex"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

// default anonymous user token, c.g a token's antonym
const AnonToken = "antonym"

// Connection represents a static bidirectional communication
// e.g. Congress.
type Connection struct {
	// Address of the sender
	Sender string `json:"sender"`
	// Address of the receiver
	Receiver string `json:"receiver"`
	// Time message was received from the sender
	SendEpoch int64 `json:"send_epoch"`
	// Contents of the connection
	Message string `json:"message"`
	// Time message was sent to the receiver
	ReceiveEpoch int64 `json:"receive_epoch"`
}

// Connections are an array
// of connections, useful in list responses
type Connections struct {
	Connections []*Connection `json:"connections"`
}

func (c *Connection) baseString() (stringy string) {
	encodedMessage := hex.EncodeToString([]byte(c.Message))
	if c.Sender == "" {
		c.Sender = AnonToken
	}
	encodedSender := hex.EncodeToString([]byte(c.Sender))
	encodedReciever := hex.EncodeToString([]byte(c.Receiver))
	stringy = fmt.Sprintf("%v %v %v %v", encodedSender,
		encodedReciever,
		c.SendEpoch,
		encodedMessage)
	return stringy
}

func (c *Connection) String() (stringy string) {
	base := c.baseString()
	stringy = fmt.Sprintf("%v %v", base, c.ReceiveEpoch)
	return stringy
}

// Equals returns bool indicating whether the
// connection matches the other connection
// Matches on 4-tuple of:
// sender, receiver, send time, and message.
func (c *Connection) Equals(other *Connection) (equal bool) {
	equal = c.Sender == other.Sender && c.Receiver == other.Receiver && c.SendEpoch == other.SendEpoch && c.Message == other.Message
	return equal
}

func ConnectionFromString(raw string) (connection *Connection, err error) {
	persistedConnection := strings.Split(strings.Replace(raw, "\n", "", -1), " ")
	// 4 because we expect connections to be serialized
	// according to the data.Message struct field order.
	if len(persistedConnection) < 4 {
		// TODO: Return named error
		return connection, errors.New(fmt.Sprintf("Invalid persisted connection: %v", persistedConnection))
	}
	encodedSender := persistedConnection[0]
	decodedSender, err := hex.DecodeString(encodedSender)
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	encodedReciever := persistedConnection[1]
	decodedReciever, err := hex.DecodeString(encodedReciever)
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	sendEpoch, err := strconv.ParseInt(persistedConnection[2], 10, 64)
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	decodedMessage, err := hex.DecodeString(persistedConnection[3])
	if err != nil {
		// TODO: Return named error
		return connection, err
	}
	connection = &Connection{
		Sender:    string(decodedSender),
		Receiver:  string(decodedReciever),
		SendEpoch: sendEpoch,
		Message:   string(decodedMessage)}
	if len(persistedConnection) > 4 {
		receiveEpoch, err := strconv.ParseInt(persistedConnection[4], 10, 64)
		if err != nil {
			// TODO: Return named error
			return connection, err
		}
		connection.ReceiveEpoch = receiveEpoch
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
