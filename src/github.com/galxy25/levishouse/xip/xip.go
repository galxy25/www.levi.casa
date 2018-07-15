// Package xip contains common interfaces
// and eXecute In Place data structures
// for www.levi.casa servers
package xip

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
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

// EmailConnect is the XIP for a
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

func (e *EmailConnect) ToString() (stringy string) {
	if isEmpty(&e.EmailConnectId) {
		e.EmailConnectId = ANON_TOKER
	}
	encoded_message := base64.StdEncoding.EncodeToString([]byte(e.EmailConnect))
	// We really care about whitespace because
	// we're matching off of the string via grep
	if isEmpty(&e.ConnectEpoch) {
		stringy = fmt.Sprintf("%v:%v %v %t %v\n", e.EmailConnectId,
			toker,
			encoded_message,
			e.SubscribeToMailingList,
			e.ReceiveEpoch)
	} else {
		stringy = fmt.Sprintf("%v:%v %v %t %v %v\n", e.EmailConnectId,
			toker,
			encoded_message,
			e.SubscribeToMailingList,
			e.ReceiveEpoch,
			e.ConnectEpoch)
	}
	return stringy
}

func EmailConnectFromString(raw string) (email_connection *EmailConnect, err error) {
	// Read persisted connection
	// into an EmailConnection interface
	// TODO, add this as an interface method
	persisted_connection := strings.Split(raw, " ")
	fmt.Printf("Persisted connection: %v \n", persisted_connection)
	// HACKs to stop runtime panics
	// due to blindly reading any garbage line in desired
	// as a valid connection
	if len(persisted_connection) < 3 {
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

// Connections is the XIP for
// collections of EmailConnect's
type Connections struct {
	EmailConnections []EmailConnect `json:"email_connections"`
}
