// Package xip contains common interfaces
// and eXecute In Place data structures
// for www.levi.casa servers
package xip

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
	ReceiveEpoch string `json:receive_epoch`
	// Time message was sent to the receiver
	ConnectEpoch string `json:connect_epoch`
}

// Connections is the XIP for
// collections of EmailConnect's
type Connections struct {
	EmailConnections []EmailConnect `json:"email_connections"`
}
