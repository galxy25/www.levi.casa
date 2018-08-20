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
const anonToker = "antonym"

// default user connections are sent to and from
var toker = os.Getenv("TOKER")

// Connection holds information needed to record, report, and link a connection
type Connection struct {
	// Contents of the connection
	Message string `json:"message"`
	// Address of the sender
	Sender string `json:"sender"`
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
	Connections []*Connection `json:"connections"`
}

func (c *Connection) baseString() (stringy string) {
	encodedMessage := hex.EncodeToString([]byte(c.Message))
	if c.Sender == "" {
		c.Sender = fmt.Sprintf("%v:%v", anonToker, toker)
	}
	encodedConnectId := hex.EncodeToString([]byte(c.Sender))
	stringy = fmt.Sprintf("%v:%v %v %t %v", encodedConnectId,
		toker,
		encodedMessage,
		c.SubscribeToMailingList,
		c.ReceiveEpoch)
	return stringy
}

func (c *Connection) String() (stringy string) {
	base := c.baseString()
	stringy = fmt.Sprintf("%v %v", base, c.ConnectEpoch)
	return stringy
}

// Equals returns bool indicating whether the
// connection matches the other connection
// Matches on 3-tuple of:
// message, sender, receive time.
func (c *Connection) Equals(other *Connection) (equal bool) {
	equal = c.Message == other.Message && c.Sender == other.Sender && c.ReceiveEpoch == other.ReceiveEpoch
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
		Sender:                 string(decoded_sender),
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
