// Package communicator provides the ability to link, record, report, and reconcile connections.
package communicator

import (
	"fmt"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"time"
)

// Configure package logging context
var packageLogger = log.WithFields(log.Fields{
	"package": "home/communicator",
})

// Communicator implements functionality
// to make and verify bi-directional connections.
type Communicator struct {
	desiredConnections ConnectionFile
	currentConnections ConnectionFile
}

// Sender implements sending a connection over
// the senders protocol.
type Sender interface {
	Send() (err error)
}

// Link attempts to make a connection using the send()
// protocol of the provided sender
// returning made connection and send error (if any).
func (c *Communicator) Link(newConnection *data.Connection, sender Sender) (madeConnection *data.Connection, err error) {
	err = sender.Send()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"executor":    "#Link.sender.#Send",
			"error":       err.Error(),
			"connection":  newConnection,
			"sender_type": fmt.Sprintf("%T", sender),
		}).Error("error making connection")
		return madeConnection, err
	}
	connectedTimestamp := time.Now()
	connectEpoch := connectedTimestamp.Unix()
	newConnection.ReceiveEpoch = connectEpoch
	packageLogger.WithFields(log.Fields{
		"executor":    "#Link",
		"connection":  newConnection,
		"sender_type": fmt.Sprintf("%T", sender),
	}).Info("successfully linked connection")
	err = c.currentConnections.WriteConnection(newConnection)
	return newConnection, err
}

// Record records a new connection
// returning error (if any).
func (c *Communicator) Record(newConnection *data.Connection) (err error) {
	err = c.desiredConnections.WriteConnection(newConnection)
	return err
}

// Sent reports all linked connections
// for a communicator, returning linked connections and error (if any).
// To stop an in progress report, send on the finish channel.
func (c *Communicator) Sent(finish <-chan struct{}) (linked chan *data.Connection, err error) {
	linked, err = c.currentConnections.Each(finish)
	return linked, err
}

// Unsent returns all connections that have been
// received but not sent, and error (if any).
func (c *Communicator) Unsent() (unlinked []*data.Connection, err error) {
	stop := make(chan struct{})
	defer close(stop)
	desired, err := c.Received(stop)
	if err != nil {
		return unlinked, err
	}
	current, err := c.Sent(stop)
	if err != nil {
		return unlinked, err
	}
	var linked, maybeLinked []*data.Connection
	for connection := range desired {
		maybeLinked = append(maybeLinked, connection)
	}
	for connection := range current {
		linked = append(linked, connection)
	}
	var match bool
	for _, connection := range maybeLinked {
		match = false
		for _, madeConnection := range linked {
			if connection.Equals(madeConnection) {
				match = true
				break
			}
		}
		if match {
			continue
		}
		unlinked = append(unlinked, connection)
	}
	return unlinked, err
}

// Received reports all received connections
// for a communicator, returning those connections
// and error (if any). Received connections will also appear
// in the list of connections reported by Communicator.Sent
// if the connection has been linked.
// To stop an in progress report, send on the finish channel.
func (c *Communicator) Received(finish <-chan struct{}) (unlinked chan *data.Connection, err error) {
	unlinked, err = c.desiredConnections.Each(finish)
	return unlinked, err
}

// Reconcile attempts to link all
// unconnected connections, returning
// reconciled connections and error (if any).
func (c *Communicator) Reconcile() (reconciled []*data.Connection, err error) {
	unlinked, err := c.Unsent()
	if err != nil {
		return reconciled, err
	}
	for _, connection := range unlinked {
		sender, err := Translate(connection)
		if err != nil {
			continue
		}
		connected, err := c.Link(connection, sender)
		if err != nil {
			packageLogger.WithFields(log.Fields{
				"executor":    "#Reconcile.#Link",
				"connection":  connection,
				"err":         err,
				"sender_type": fmt.Sprintf("%T", sender),
			}).Error("failed to link connection")
			// TODO: return [] of unreconciled?
			continue
		}
		reconciled = append(reconciled, connected)
	}
	return reconciled, err
}

// NewCommunicator returns a new communicator
// that uses the provided file paths to record
// and report connections as they are initiated
// and linked.
func NewCommunicator(desiredConnectionsFilePath string, currentConnectionsFilePath string) (communicator Communicator) {
	communicator = Communicator{
		desiredConnections: NewConnectionFile(desiredConnectionsFilePath),
		currentConnections: NewConnectionFile(currentConnectionsFilePath),
	}
	return communicator
}
