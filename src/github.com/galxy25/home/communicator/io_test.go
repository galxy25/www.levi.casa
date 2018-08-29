package communicator

import (
	"bufio"
	"fmt"
	"github.com/galxy25/home/data"
	helper "github.com/galxy25/home/internal/test"
	"os"
	"testing"
)

var randomConnections = []*data.Connection{
	helper.RandomEmailConnection(),
	helper.RandomSmsConnection(),
	helper.RandomEmailConnection(),
	helper.RandomSmsConnection(),
	helper.RandomEmailConnection(),
	helper.RandomSmsConnection(),
}

func TestSerializeConnectionReturnsSameConnectionAsBytes(t *testing.T) {
	connection := helper.RandomEmailConnection()
	serializedConnection, err := SerializeConnection(connection)
	if err != nil {
		t.Errorf("error serializing connection %v:%v\n", connection, err)
	}
	if string(serializedConnection) != fmt.Sprintf("%v\n", connection.String()) {
		t.Errorf("got %v\n wanted %v\n", string(serializedConnection), connection.String())
	}
}

func TestDeserializeConnectionReturnsSameConnectionAsInterface(t *testing.T) {
	connection := helper.RandomEmailConnection()
	serializedConnection, err := SerializeConnection(connection)
	if err != nil {
		t.Errorf("error serializing connection %v:%v\n", connection, err)
	}
	deserializedConnection, err := DeserializeConnection(serializedConnection)
	if err != nil {
		t.Errorf("error deserializing connection %v:%v\n", connection, err)
	}
	castConnection, err := castAsConnectionPtr(deserializedConnection)
	if err != nil {
		t.Errorf("deserializedConnection is not a connection pointer: %v\n", err)
	}
	if castConnection.String() != connection.String() {
		t.Errorf("got %v\n wanted %v\n", castConnection.String(), connection.String())
	}
}

func TestWriteConnectionWritesSerializedConnectionToConnectionFile(t *testing.T) {
	connectionFilePath := "TestWriteConnectionWritesSerializedConnectionToConnectionFile.txt"
	defer os.Remove(connectionFilePath)
	connectionFile := NewConnectionFile(connectionFilePath)
	connection := helper.RandomEmailConnection()
	err := connectionFile.WriteConnection(connection)
	if err != nil {
		t.Errorf("%v failed writing connection %v to %v\n", err, connection, connectionFilePath)
	}
	rawFile, err := os.OpenFile(connectionFilePath, os.O_RDONLY, 0644)
	if err != nil {
		t.Error(err)
	}
	reader := bufio.NewReader(rawFile)
	wroteConnection, err := reader.ReadBytes('\n')
	if err != nil {
		t.Error(err)
	}
	if string(wroteConnection) != fmt.Sprintf("%v\n", connection.String()) {
		t.Errorf("expected %v got %v\n", connection.String(), string(wroteConnection))
	}
}

func TestWriteConnectionsWritesSerializedConnectionsToConnectionFile(t *testing.T) {
	connectionFilePath := "TestWriteConnectionsWritesSerializedConnectionsToConnectionFile.txt"
	defer os.Remove(connectionFilePath)
	connectionFile := NewConnectionFile(connectionFilePath)
	err := connectionFile.WriteConnections(randomConnections)
	if err != nil {
		t.Errorf("%v failed writing connections %v to %v\n", err, randomConnections, connectionFilePath)
	}
	rawFile, err := os.OpenFile(connectionFilePath, os.O_RDONLY, 0644)
	if err != nil {
		t.Error(err)
	}
	reader := bufio.NewReader(rawFile)
	for _, connection := range randomConnections {
		wroteConnection, err := reader.ReadBytes('\n')
		if err != nil {
			t.Error(err)
		}
		if string(wroteConnection) != fmt.Sprintf("%v\n", connection.String()) {
			t.Errorf("expected %v got %v\n", connection.String(), string(wroteConnection))
		}
	}
}

func TestFindConnectionFindsWroteConnectionInConnectionFile(t *testing.T) {
	connectionFilePath := "TestFindConnectionFindsWroteConnectionInConnectionFile.txt"
	defer os.Remove(connectionFilePath)
	connectionFile := NewConnectionFile(connectionFilePath)
	connection := helper.RandomEmailConnection()
	err := connectionFile.WriteConnection(connection)
	if err != nil {
		t.Errorf("%v failed writing connection %v to %v\n", err, connection, connectionFilePath)
	}
	connectionWrote, err := connectionFile.FindConnection(connection)
	if err != nil {
		t.Error(err)
	}
	if !connectionWrote {
		t.Errorf("failed to find %v in %v", connection, connectionFilePath)
	}
}

func TestFindConnectionsFindsWroteConnectionsInConnectionFile(t *testing.T) {
	connectionFilePath := "TestFindConnectionsFindsWroteConnectionsInConnectionFile.txt"
	defer os.Remove(connectionFilePath)
	connectionFile := NewConnectionFile(connectionFilePath)
	err := connectionFile.WriteConnections(randomConnections)
	if err != nil {
		t.Errorf("%v failed writing connections %v to %v\n", err, randomConnections, connectionFilePath)
	}
	wroteConnections, errs := connectionFile.FindConnections(randomConnections)
	for err := range errs {
		t.Error(err)
	}
	var match bool
	for _, connection := range randomConnections {
		match = false
		for _, wroteConnection := range wroteConnections {
			if connection.Equals(wroteConnection) {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("failed to find %v in wroteConnections %v", connection, wroteConnections)
		}
	}
}

func TestEachReturnsAllWroteConnectionsInConnectionFile(t *testing.T) {
	connectionFilePath := "TestEachReturnsAllWroteConnectionsInConnectionFile.txt"
	defer os.Remove(connectionFilePath)
	connectionFile := NewConnectionFile(connectionFilePath)
	err := connectionFile.WriteConnections(randomConnections)
	if err != nil {
		t.Errorf("%v failed writing connections %v to %v\n", err, randomConnections, connectionFilePath)
	}
	stop := make(chan struct{})
	defer close(stop)
	each, err := connectionFile.Each(stop)
	if err != nil {
		t.Error(err)
	}
	var wroteConnections []*data.Connection
	for wroteConnection := range each {
		wroteConnections = append(wroteConnections, wroteConnection)
	}
	if len(wroteConnections) != len(randomConnections) {
		t.Errorf("expecting %v wrote connections, got %v", len(randomConnections), len(wroteConnections))
	}
	var match bool
	for _, connection := range randomConnections {
		match = false
		for _, wroteConnection := range wroteConnections {
			if connection.Equals(wroteConnection) {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("failed to find %v in wroteConnections %v", connection, wroteConnections)
		}
	}
}
