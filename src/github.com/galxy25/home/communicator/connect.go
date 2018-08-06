// Package communicator provides the ability to record, report, and link connections.
package communicator

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	"file":    "connect.go",
})

// SweepConnections reconciles program state between desired and current connections
// between desired and current connections via algorithm:
//      for each connection in desiredConnections
//          verify it is not in currentConnections
//      if it is
//          remove it from desiredConnections
// When SweepConnections returns
// all past connections are swept
// and future connections can be swept
// by sending them to the madeConnections channel.
func SweepConnections(desiredConnectionsFilePath string, currentConnectionsFilePath string, madeConnections <-chan *data.Connection) {
	desiredConnections, err := os.OpenFile(desiredConnectionsFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	defer desiredConnections.Close()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#SweepConnections",
			"error":    err,
			"io":       desiredConnectionsFilePath,
		}).Fatal("failed to open file")
	}
	desiredConnectionsScanner := bufio.NewScanner(desiredConnections)
	desiredConnectionsScanner.Split(bufio.ScanLines)
	for desiredConnectionsScanner.Scan() {
		// Check to see if desired connection
		// is in the list of current connections
		rawConnection := desiredConnectionsScanner.Text()
		desiredConnection, err := data.ConnectionFromString(rawConnection)
		if err != nil {
			packageLogger.WithFields(log.Fields{
				"connection": rawConnection,
				"executor":   "#SweepConnections",
				"err":        err,
			}).Info("skipping sweeping invalid desired connection")
			continue
		}
		connected := desiredConnection.ExistsInFile(currentConnectionsFilePath)
		if !connected {
			continue
		}
		// Sweep the made connection
		sweepErr := sweepConnection(desiredConnection, desiredConnectionsFilePath)
		if sweepErr == nil {
			packageLogger.WithFields(log.Fields{
				"swept":      desiredConnection,
				"swept_from": desiredConnectionsFilePath,
				"executor":   "#SweepConnections",
			}).Info("Swept!")
		} else {
			packageLogger.WithFields(log.Fields{
				"to_sweep":   desiredConnection,
				"sweep_from": desiredConnectionsFilePath,
				"executor":   "#SweepConnections",
				"err":        sweepErr,
			}).Error("failed to sweep")
		}
	}
	// runs until close is called on madeConnections
	go sweeperD(desiredConnectionsFilePath, madeConnections)
}

// Connect sends messages from one person to another
// for each desired connection in desiredConnections
//      attempt to make the connection
//      if connection successful
//          write it to currentConnections
//      else
//          no-op
func Connect(desiredConnectionsFilePath string, currentConnectionsFilePath string, madeConnections chan<- *data.Connection, newConnections <-chan *data.Connection) {
	desiredConnections, err := os.OpenFile(desiredConnectionsFilePath, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer desiredConnections.Close()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#Connect",
			"error":    err,
			"io":       desiredConnectionsFilePath,
		}).Fatal("Failed to open file")
	}
	desiredConnectionsScanner := bufio.NewScanner(desiredConnections)
	desiredConnectionsScanner.Split(bufio.ScanLines)
	// Iterate over each desired connection
	// and attempt to make the connection
	for desiredConnectionsScanner.Scan() {
		currentLine := desiredConnectionsScanner.Text()
		if currentLine == "" {
			packageLogger.WithFields(log.Fields{
				"resource": "io/file",
				"io":       desiredConnectionsFilePath,
				"executor": "#Connect",
			}).Info("Skipping empty connection")
			continue
		} else {
			connection, err := data.ConnectionFromString(currentLine)
			if err != nil {
				packageLogger.WithFields(log.Fields{
					"connection": currentLine,
					"executor":   "#Connect",
					"error":      err,
				}).Error("Unable to de-serialize desired connection")
				continue
			}
			go doConnection(connection, currentConnectionsFilePath, madeConnections)
		}
	}
	// runs until close is called on newConnections
	go connectD(currentConnectionsFilePath, madeConnections, newConnections)
}

// Connect connects new connections as they appear, runs until close is called on newConnections
func connectD(currentConnectionsFilePath string, madeConnections chan<- *data.Connection, newConnections <-chan *data.Connection) {
	for {
		select {
		case newConnection, more := <-newConnections:
			if !more {
				return
			}
			go doConnection(newConnection, currentConnectionsFilePath, madeConnections)
		}
	}
}

// sweeperD sweeps new connections as they occur, runs until close is called on madeConnections
func sweeperD(desiredConnectionsFilePath string, madeConnections <-chan *data.Connection) {
	for {
		select {
		case madeConnection, more := <-madeConnections:
			if !more {
				return
			}
			// Sweep the made connection
			sweepErr := sweepConnection(madeConnection, desiredConnectionsFilePath)
			if sweepErr == nil {
				packageLogger.WithFields(log.Fields{
					"swept":      madeConnection,
					"swept_from": desiredConnectionsFilePath,
					"executor":   "#SweepConnections",
				}).Info("Swept!")
			} else {
				packageLogger.WithFields(log.Fields{
					"to_sweep":   madeConnection,
					"sweep_from": desiredConnectionsFilePath,
					"executor":   "#sweeperD",
					"err":        sweepErr,
				}).Error("failed to sweep")
			}
		}
	}
}

// sweepConnection sweeps a connection from a file
// returning error if any
func sweepConnection(connection *data.Connection, desiredConnectionsFilePath string) (err error) {
	sedCommand := "sed"
	// Sanitize raw connection string for sed'ding by removing trailing newline
	sedSafeConnection := strings.Replace(connection.ToString(), "\n", "", 1)
	sedSweepArgs := []string{"-i", fmt.Sprintf("s/%v//g", sedSafeConnection), desiredConnectionsFilePath}
	if runtime.GOOS == "darwin" {
		// cuz, BSD ain't GNU
		sedSweepArgs = []string{"-i", "", fmt.Sprintf("s/%v//g", sedSafeConnection), desiredConnectionsFilePath}
	}
	//  Remove the realized from the desired
	_, err = exec.Command(sedCommand, sedSweepArgs...).Output()
	return err
}

// doConnection makes connections by
// emailing me the connection via AWS SNS
func doConnection(connection *data.Connection, currentConnectionsFilePath string, madeConnections chan<- *data.Connection) {
	packageLogger.WithFields(log.Fields{
		"executor":           "#doConnection",
		"command_parameters": connection,
	}).Info("Processing connection to initiate")
	message := connection.Message
	resp, err := snsPublisher(message)
	if err != nil {
		// Cast err to awserr.Error
		// to get the Code and Message from an error.
		packageLogger.WithFields(log.Fields{
			"executor":           "#doConnection.#snsPublisher",
			"command_parameters": message,
			"error":              err.Error(),
		}).Error("error making connection")
		return
	} else {
		// Record the time this connection was made
		soConnectTimestamp := time.Now()
		soConnectEpoch := soConnectTimestamp.Unix()
		connection.ConnectEpoch = soConnectEpoch
		// Send processed email connection to
		// next stage in pipeline
		packageLogger.WithFields(log.Fields{
			"executor":           "#doConnection.#snsPublisher",
			"command_parameters": connection,
			"command_response":   resp,
		}).Info("Successfully made connection")
		soConnection(connection, currentConnectionsFilePath, madeConnections)
	}
}

// soConnection does the things we do after a connection is made
// answering the question: "You made a connection. So what?"
func soConnection(connection *data.Connection, currentConnectionsFilePath string, madeConnections chan<- *data.Connection) {
	currentConnections, err := os.OpenFile(currentConnectionsFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer currentConnections.Close()
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource": "io/file",
			"executor": "#soConnection",
			"io":       currentConnectionsFilePath,
		}).Fatal("failed to open file")
	}
	packageLogger.WithFields(log.Fields{
		"resource": connection,
		"executor": "#soConnection",
	}).Info("Processing connection")
	connectionData := connection.ToString()
	_, err = currentConnections.WriteString(connectionData)
	if err != nil {
		packageLogger.WithFields(log.Fields{
			"resource":           "io/file",
			"executor":           "#soConnection",
			"command_parameters": connectionData,
			"io":                 currentConnectionsFilePath,
		}).Fatal("failed to persist connection")
	}
	packageLogger.WithFields(log.Fields{
		"resource":           "io/file",
		"executor":           "#soConnection",
		"command_parameters": connectionData,
		"io":                 currentConnectionsFilePath,
	}).Info("Successfully persisted connection")
	madeConnections <- connection
}
