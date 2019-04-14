// Package data contains data structures
// for use by www.levi.casa servers and clients
package data

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// AnonToken is the anonymous user's token, e.g a token's opposite.
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

// ConnectionFromString attempts to parse
// and return a connection object from a string
// returning parsed connection and error (if any)
// TODO: refactor as a scan line function for bufio
// https://golang.org/pkg/bufio/#SplitFunc
// or replace with a managed key
// value backend (e.g. a database)
func ConnectionFromString(raw string) (*Connection, error) {
	persistedConnection := strings.Split(strings.Replace(raw, "\n", "", -1), " ")
	// 4 because we expect connections to be serialized
	// according to the data.Message struct field order.
	if len(persistedConnection) < 4 {
		// TODO: Return named error
		return nil, fmt.Errorf("Invalid persisted connection: %v", persistedConnection)
	}
	encodedSender := persistedConnection[0]
	decodedSender, err := hex.DecodeString(encodedSender)
	if err != nil {
		// TODO: Return named error
		return nil, err
	}
	encodedReciever := persistedConnection[1]
	decodedReciever, err := hex.DecodeString(encodedReciever)
	if err != nil {
		// TODO: Return named error
		return nil, err
	}
	sendEpoch, err := strconv.ParseInt(persistedConnection[2], 10, 64)
	if err != nil {
		// TODO: Return named error
		return nil, err
	}
	decodedMessage, err := hex.DecodeString(persistedConnection[3])
	if err != nil {
		// TODO: Return named error
		return nil, err
	}
	connection := Connection{
		Sender:    string(decodedSender),
		Receiver:  string(decodedReciever),
		SendEpoch: sendEpoch,
		Message:   string(decodedMessage)}
	if len(persistedConnection) > 4 {
		receiveEpoch, err := strconv.ParseInt(persistedConnection[4], 10, 64)
		if err != nil {
			// TODO: Return named error
			return nil, err
		}
		connection.ReceiveEpoch = receiveEpoch
	}
	return &connection, err
}

// Phenomenon are a series of
// cohesive digital events such as a
// chat, video, or application
type Phenomenon struct {
	ID   string `json:"id"`
	Data []byte `json:"data"`
}

// A Link is a retrievable
// piece of content
type Link struct {
	ID       string   `json:"id"`
	URI      string   `json:"uri"`
	Mirrors  []Link   `json:"mirrors"`
	Inbound  []Link   `json:"inbound"`
	Outbound []Link   `json:"outbound"`
	Memories []string `json:"memories"`
}

// A Comment is a comment on a memory
// by a commenter at at a certain time
// piece of
type Comment struct {
	ID           string `json:"id"`
	Parent       string `json:"parent"`
	Commenter    string `json:"commenter"`
	CommentEpoch int64  `json:"comment_epoch"`
	Comment      string `json:"comment"`
}

// A Tag is a user defined key value identifier
type Tag struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Tagger   string `json:"tagger"`
	TagEpoch int64  `json:"tag_epoch"`
}

// Context contains a collection of digital media
type Context struct {
	Links         []Link       `json:"links"`
	Comments      []Comment    `json:"comments"`
	Tags          []Tag        `json:"tags"`
	Phenomenons   []Phenomenon `json:"phenomenons"`
	ScheduleEpoch int64        `json:"schedule_epoch"`
}

type Entity struct {
}

// A Memory represents a
// relationship between time,
// entities (people, places, things)
// and some digital context.
type Memory struct {
	ID           string   `json:"id"`
	PublishEpoch int64    `json:"publish_epoch"`
	Publisher    string   `json:"Publisher"`
	Entities     []string `json:"entities"`
	Context      Context  `json:"context"`
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
