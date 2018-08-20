// Package communicator provides the ability to link, record, report, and reconcile connections.
package communicator

import (
	"fmt"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

// Address for receiving email communications.
var homeEmail = os.Getenv("HOME_EMAIL")

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

// Link attempts to make a connection,
// returning made connection and error (if any).
func (c *Communicator) Link(newConnection *data.Connection) (madeConnection *data.Connection, err error) {
	// XXX: only email's for the moment
	// coming soon: SMS, Slack, SnapChat
	// FB messenger, twitter, ...
	email := &Email{
		Message:   newConnection.Message,
		Sender:    newConnection.Sender,
		Receivers: []string{homeEmail},
		Subject:   fmt.Sprintf("%v -> www.levi.casa", newConnection.Sender),
	}
	err = email.Send()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"executor":           "#Link.Email.#Send",
			"command_parameters": email,
			"error":              err.Error(),
			"connection":         newConnection,
		}).Error("error making connection")
		return madeConnection, err
	}
	connectedTimestamp := time.Now()
	connectEpoch := connectedTimestamp.Unix()
	newConnection.ConnectEpoch = connectEpoch
	packageLogger.WithFields(log.Fields{
		"executor":   "#Link",
		"connection": newConnection,
	}).Info("successfully linked connection")
	err = c.currentConnections.WriteConnection(newConnection)
	return newConnection, err
}

// Record records new connection
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
	stop := make(chan struct{})
	defer close(stop)
	desired, err := c.Received(stop)
	if err != nil {
		return reconciled, err
	}
	current, err := c.Sent(stop)
	if err != nil {
		return reconciled, err
	}
	var linked, maybeLinked, unlinked []*data.Connection
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
	for _, connection := range unlinked {
		connected, err := c.Link(connection)
		if err != nil {
			packageLogger.WithFields(log.Fields{
				"executor":   "#Reconcile.#Link",
				"connection": connection,
				"err":        err,
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
