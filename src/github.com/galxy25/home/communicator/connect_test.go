package communicator

import (
	"errors"
	"fmt"
	data "github.com/galxy25/home/data"
	forEach "github.com/galxy25/home/internal/forEach"
	io "github.com/galxy25/home/internal/io"
	"os"
	"testing"
	"time"
)

func defaultConnections() (connections []data.Connection) {
	connections = []data.Connection{
		data.Connection{
			Connection:             "Salutations, Body, Farewell",
			ConnectionId:           "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           1531622217,
		},
		data.Connection{
			Connection:             "Farewell, Salutations, Body",
			ConnectionId:           "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           1531622299,
		},
		data.Connection{
			Connection:             "Body, Salutations, Farewell",
			ConnectionId:           "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           1531622200,
		},
		data.Connection{
			Connection:             "wow, whoa, well",
			ConnectionId:           "rando@randos.com",
			SubscribeToMailingList: true,
			ReceiveEpoch:           1531622369,
		},
	}
	return connections
}

func castToConnection(a interface{}) (connection data.Connection, err error) {
	connection, ok := a.(data.Connection)
	if !ok {
		return connection, errors.New(fmt.Sprintf("Unable to cast %v to a data.Connection\n", a))
	}
	return connection, nil
}

func SerializeConnection(deserialized interface{}) (serialized []byte, err error) {
	connection, err := castToConnection(deserialized)
	if err != nil {
		return serialized, err
	}
	serialized = []byte(connection.ToString())
	return serialized, nil
}

func DeserializeConnection(serialized []byte) (deserialized interface{}, err error) {
	connection, err := data.ConnectionFromString(string(serialized))
	if err != nil {
		return deserialized, err
	}
	deserialized = *connection
	return deserialized, err
}

type ConnectionFile struct {
	*io.SerializedLFile
}

func NewConnectionFile(filePath string) (file ConnectionFile) {
	sf := &io.SerializedLFile{
		FilePath:    filePath,
		Serialize:   SerializeConnection,
		Deserialize: DeserializeConnection,
	}
	return ConnectionFile{sf}
}

func (c *ConnectionFile) WriteConnections(connections []data.Connection) (err error) {
	for _, connection := range connections {
		_, err = c.Store(connection)
		if err != nil {
			return err
		}
	}
	return err
}

func (c *ConnectionFile) FindConnections(connections []data.Connection) (found []data.Connection, errs []error) {
	finder, findErr := forEach.Select(c.All, func(item interface{}) (predicate bool, err error) {
		connectionItem, err := castToConnection(item)
		if err != nil {
			return predicate, err
		}
		for _, connection := range connections {
			if connectionItem.Matches(&connection) {
				predicate = true
				break
			}
		}
		return predicate, err
	})
	if findErr != nil {
		errs = append(errs, findErr)
		return found, errs
	}
	for find := range finder {
		if find.Err != nil {
			errs = append(errs, find.Err)
		}
		connectionItem, err := castToConnection(find.Item)
		if err != nil {
			continue
		}
		found = append(found, connectionItem)

	}
	return found, errs
}

func (c *ConnectionFile) DetectConnection(connection data.Connection) (detected bool, err error) {
	found, err := forEach.Detect(c.All, func(item interface{}) (predicate bool, err error) {
		connectionItem, err := castToConnection(item)
		if err != nil {
			return predicate, err
		}
		if connectionItem.Matches(&connection) {
			predicate = true
		}
		return predicate, err
	})
	if found != nil {
		detected = true
	}
	return detected, err
}

func TestSweepConnectionsSweepsMadeConnections(t *testing.T) {
	desired, current := "TestSweepConnectionsSweepsMadeConnections.desired", "TestSweepConnectionsSweepsMadeConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	desiredConnections := NewConnectionFile(desired)
	currentConnections := NewConnectionFile(current)
	seedData := defaultConnections()
	err := desiredConnections.WriteConnections(seedData)
	if err != nil {
		t.Error(err)
	}
	connected := make(chan *data.Connection)
	defer close(connected)
	for _, desiredConnection := range seedData {
		desiredConnections.Store(desiredConnection)
		// fake a made connection
		desiredConnection.ConnectEpoch = 2531622299
		currentConnections.Store(desiredConnection)
	}
	SweepConnections(desired, current, connected)
	unConnected, errs := desiredConnections.FindConnections(seedData)
	if len(unConnected) != 0 {
		t.Errorf("failed to make these connections %v\n", unConnected)
	}
	for err := range errs {
		t.Errorf("error %v while searching %v for %v", err, desired, seedData)
	}
}

func TestSweepConnectionsSweepsNewlyMadeConnections(t *testing.T) {
	desired, current := "TestSweepConnectionsSweepsNewlyMadeConnections.desired", "TestSweepConnectionsSweepsNewlyMadeConnections.current"
	connected := make(chan *data.Connection)
	defer close(connected)
	SweepConnections(desired, current, connected)
	defer os.Remove(desired)
	defer os.Remove(current)
	desiredConnections := NewConnectionFile(desired)
	currentConnections := NewConnectionFile(current)
	seedData := defaultConnections()
	for _, desiredConnection := range seedData {
		desiredConnections.Store(desiredConnection)
		desiredConnection.ConnectEpoch = 2531622299
		currentConnections.Store(desiredConnection)
		// Send the faux made connection to the sweeper
		connected <- &data.Connection{
			Connection:             desiredConnection.Connection,
			ConnectionId:           desiredConnection.ConnectionId,
			SubscribeToMailingList: desiredConnection.SubscribeToMailingList,
			ReceiveEpoch:           desiredConnection.ReceiveEpoch,
		}
	}
	unConnected, errs := desiredConnections.FindConnections(seedData)
	for tries := 10; tries > 0 && len(unConnected) != 0; tries-- {
		time.Sleep(10 * time.Millisecond)
		unConnected, errs = desiredConnections.FindConnections(seedData)
	}
	if len(unConnected) != 0 {
		t.Errorf("failed to sweep these connections %v\n", unConnected)
	}
	for err := range errs {
		t.Errorf("error %v while searching %v for %v", err, desired, seedData)
	}
}

func TestConnectMakesConnections(t *testing.T) {
	desired, current := "TestConnectMakesConnections.desired", "TestConnectMakesConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	desiredConnections := NewConnectionFile(desired)
	seedData := defaultConnections()
	err := desiredConnections.WriteConnections(seedData)
	if err != nil {
		t.Error(err)
	}
	saved := sns_publisher
	defer func() { sns_publisher = saved }()
	sns_publisher = func(message string) (resp interface{}, err error) {
		return resp, err
	}
	connected := make(chan *data.Connection, len(seedData))
	newConnectionsQueue := make(chan *data.Connection)
	defer close(newConnectionsQueue)
	defer close(connected)
	Connect(desired, current, connected, newConnectionsQueue)
	currentConnections := NewConnectionFile(current)
	for _, seed := range seedData {
		detected, err := currentConnections.DetectConnection(seed)
		if !detected {
			t.Errorf("failed to find %v in %v\n", seed, current)
		}
		if err != nil {
			t.Errorf("error %v while trying to detect %v in %v\n ", err, seed, current)
		}
	}
	// Prevent data race of closing
	// connected channel after the connection
	// is persisted to file but not enqueued
	for _, _ = range seedData {
		<-connected
	}
}

func TestConnectReportsMadeConnections(t *testing.T) {
	desired, current := "TestConnectReportsMadeConnections.desired", "TestConnectReportsMadeConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	desiredConnections := NewConnectionFile(desired)
	currentConnections := NewConnectionFile(current)
	seedData := defaultConnections()
	err := desiredConnections.WriteConnections(seedData)
	if err != nil {
		t.Error(err)
	}
	connected := make(chan *data.Connection, len(seedData))
	newConnectionsQueue := make(chan *data.Connection)
	defer close(newConnectionsQueue)
	saved := sns_publisher
	defer func() { sns_publisher = saved }()
	sns_publisher = func(message string) (resp interface{}, err error) {
		return resp, err
	}
	Connect(desired, current, connected, newConnectionsQueue)
	for _, seed := range seedData {
		exists := seed.ExistsInFile(current)
		if !exists {
			t.Errorf("failed to find %v in %v\n", seed, current)
		}
	}
	var connectionAlerts []data.Connection
	for i := 0; i < len(seedData); i++ {
		connectionAlerts = append(connectionAlerts, *(<-connected))
	}
	found, errs := currentConnections.FindConnections(connectionAlerts)
	if len(found) != len(connectionAlerts) {
		t.Errorf("failed to find all of %v\n in %v\n", connectionAlerts, found)
	}
	for _, find := range found {
		alerted := false
		for _, alert := range connectionAlerts {
			if alert.Matches(&find) {
				alerted = true
				break
			}
		}
		if !alerted {
			t.Errorf("no alert sent for connection %v\n", find)
		}
	}
	if len(errs) > 0 {
		t.Error(errs)
	}
}

func TestConnectConnectsNewConnections(t *testing.T) {
	desired, current := "TestConnectConnectsNewConnections.desired", "TestConnectConnectsNewConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	saved := sns_publisher
	defer func() { sns_publisher = saved }()
	sns_publisher = func(message string) (resp interface{}, err error) {
		return resp, err
	}
	seedData := defaultConnections()
	connected := make(chan *data.Connection, len(seedData))
	newConnectionsQueue := make(chan *data.Connection)
	defer close(newConnectionsQueue)
	defer close(connected)
	Connect(desired, current, connected, newConnectionsQueue)
	desiredConnections := NewConnectionFile(desired)
	err := desiredConnections.WriteConnections(seedData)
	if err != nil {
		t.Error(err)
	}
	for _, seed := range seedData {
		newConnectionsQueue <- &data.Connection{
			Connection:             seed.Connection,
			ConnectionId:           seed.ConnectionId,
			SubscribeToMailingList: seed.SubscribeToMailingList,
			ReceiveEpoch:           seed.ReceiveEpoch,
		}
	}
	for _, _ = range seedData {
		<-connected
	}
	currentConnections := NewConnectionFile(current)
	for _, seed := range seedData {
		detected, err := currentConnections.DetectConnection(seed)
		if !detected {
			t.Errorf("failed to find %v in %v\n", seed, current)
		}
		if err != nil {
			t.Errorf("error %v while trying to detect %v in %v\n ", err, seed, current)
		}
	}
}
