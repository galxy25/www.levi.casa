// Package communicator provides the ability to link, record, report, and reconcile connections.
package communicator

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

// endpoint URI for receiving connections
var snsTopicARN = os.Getenv("HOME_SNS_TOPIC")

// 🙄 only a "variable" so we can stub it in tests
var snsPublisher = func(message string) (resp interface{}, err error) {
	sess := session.Must(session.NewSession())
	svc := sns.New(sess)
	params := &sns.PublishInput{
		Message:  aws.String(message),
		TopicArn: aws.String(snsTopicARN),
		Subject:  aws.String("Message to www.levi.casa"),
	}
	resp, err = svc.Publish(params)
	return resp, err
}

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
	message := newConnection.Message
	resp, err := snsPublisher(message)
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"executor":           "#Link.#snsPublisher",
			"command_parameters": message,
			"error":              err.Error(),
			"connection":         newConnection,
		}).Error("error making connection")
		return madeConnection, err
	}
	connectedTimestamp := time.Now()
	connectEpoch := connectedTimestamp.Unix()
	newConnection.ConnectEpoch = connectEpoch
	packageLogger.WithFields(log.Fields{
		"executor":         "#Link",
		"connection":       newConnection,
		"publish_response": resp,
	}).Info("Successfully made connection")
	err = c.currentConnections.WriteConnection(newConnection)
	return newConnection, err
}

// Record records new connection
// returning error (if any).
func (c *Communicator) Record(newConnection *data.Connection) (err error) {
	err = c.desiredConnections.WriteConnection(newConnection)
	return err
}

// ReportLinked reports all linked connections
// for a communicator, returning linked connections and error (if any).
// To stop an in progress report, send on the finish channel.
func (c *Communicator) ReportLinked(finish <-chan struct{}) (linked chan *data.Connection, err error) {
	linked, err = c.currentConnections.Each(finish)
	return linked, err
}

// ReportUnlinked reports all unlinked connections
// for a communicator, returning unlinked connections and error (if any).
// To stop an in progress report, send on the finish channel.
func (c *Communicator) ReportUnlinked(finish <-chan struct{}) (unlinked chan *data.Connection, err error) {
	unlinked, err = c.desiredConnections.Each(finish)
	return unlinked, err
}

// Reconcile attempts to link all
// unconnected connections, returning
// reconciled connections and error (if any).
func (c *Communicator) Reconcile() (reconciled []*data.Connection, err error) {
	stop := make(chan struct{})
	defer close(stop)
	desired, err := c.ReportUnlinked(stop)
	if err != nil {
		return reconciled, err
	}
	current, err := c.ReportLinked(stop)
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
