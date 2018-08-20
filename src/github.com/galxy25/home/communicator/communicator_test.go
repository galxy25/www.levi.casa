package communicator

import (
	"github.com/galxy25/home/data"
	helper "github.com/galxy25/home/internal/test"
	"os"
	"testing"
)

var realSnsPublisher = snsPublisher
var mockSnsPublisher = func(message string) (resp interface{}, err error) {
	return resp, err
}

func TestSuccesfulLinkRecordsLinkedConnection(t *testing.T) {
	snsPublisher = mockSnsPublisher
	defer func() {
		snsPublisher = realSnsPublisher
	}()
	desired, current := "TestSuccesfulLinkRecordsLinkedConnection.desired", "TestSuccesfulLinkRecordsLinkedConnection.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := NewCommunicator(desired, current)
	connection := helper.RandomConnection()
	linked, err := comm.Link(connection)
	if err != nil {
		t.Error(err)
	}
	currentConnectionFile := NewConnectionFile(current)
	found, err := currentConnectionFile.FindConnection(linked)
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Errorf("failed to record linked connection %v", linked)
	}
}

func TestRecordRecordsConnection(t *testing.T) {
	desired, current := "TestRecordRecordsConnection.desired", "TestRecordRecordsConnection.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := NewCommunicator(desired, current)
	connection := helper.RandomConnection()
	err := comm.Record(connection)
	if err != nil {
		t.Error(err)
	}
	currentConnectionFile := NewConnectionFile(desired)
	found, err := currentConnectionFile.FindConnection(connection)
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Errorf("failed to record connection %v", connection)
	}
}

func TestReceivedReportsAllUnlinkedConnections(t *testing.T) {
	desired, current := "TestReceivedReportsAllUnlinkedConnections.desired", "TestReceivedReportsAllUnlinkedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := NewCommunicator(desired, current)
	connections := []*data.Connection{
		helper.RandomConnection(),
		helper.RandomConnection(),
		helper.RandomConnection(),
	}
	for _, connection := range connections {
		err := comm.Record(connection)
		if err != nil {
			t.Error(err)
		}
	}
	var reported []*data.Connection
	stop := make(chan struct{})
	defer close(stop)
	unlinkReporter, err := comm.Received(stop)
	if err != nil {
		t.Error(err)
	}
	for unlinkedConnection := range unlinkReporter {
		reported = append(reported, unlinkedConnection)
	}
	if len(reported) != len(connections) {
		t.Errorf("expected %v unlinked connections, got %v", len(connections), len(reported))
	}
	var match bool
	for _, connection := range connections {
		match = false
		for _, unlinked := range reported {
			if connection.Equals(unlinked) {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("failed to find %v in reported unlinked connections %v\n", connection, reported)
		}
	}
}

func TestSentReportsAllLinkedConnections(t *testing.T) {
	snsPublisher = mockSnsPublisher
	defer func() {
		snsPublisher = realSnsPublisher
	}()
	desired, current := "TestSentReportsAllLinkedConnections.desired", "TestSentReportsAllLinkedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := NewCommunicator(desired, current)
	connections := []*data.Connection{
		helper.RandomConnection(),
		helper.RandomConnection(),
		helper.RandomConnection(),
	}
	for _, connection := range connections {
		_, err := comm.Link(connection)
		if err != nil {
			t.Error(err)
		}
	}
	var reported []*data.Connection
	stop := make(chan struct{})
	defer close(stop)
	linkReporter, err := comm.Sent(stop)
	if err != nil {
		t.Error(err)
	}
	for linkedConnection := range linkReporter {
		reported = append(reported, linkedConnection)
	}
	if len(reported) != len(connections) {
		t.Errorf("expected %v linked connections, got %v", len(connections), len(reported))
	}
	var match bool
	for _, connection := range connections {
		match = false
		for _, linked := range reported {
			if connection.Equals(linked) {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("failed to find %v in reported linked connections %v\n", connection, reported)
		}
	}
}

func TestReconcileLinksAllUnlinkedConnections(t *testing.T) {
	snsPublisher = mockSnsPublisher
	defer func() {
		snsPublisher = realSnsPublisher
	}()
	desired, current := "TestReconcileLinksAllUnlinkedConnections.desired", "TestReconcileLinksAllUnlinkedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := NewCommunicator(desired, current)
	connections := []*data.Connection{
		helper.RandomConnection(),
		helper.RandomConnection(),
		helper.RandomConnection(),
	}
	for _, connection := range connections {
		_, err := comm.Link(connection)
		if err != nil {
			t.Error(err)
		}
	}
	unmadeConnections := []*data.Connection{
		helper.RandomConnection(),
		helper.RandomConnection(),
		helper.RandomConnection(),
	}
	for _, connection := range unmadeConnections {
		err := comm.Record(connection)
		if err != nil {
			t.Error(err)
		}
	}
	reconciled, err := comm.Reconcile()
	if err != nil {
		t.Error(err)
	}
	if len(reconciled) != len(unmadeConnections) {
		t.Errorf("expected %v linked connections, got %v", len(unmadeConnections), len(reconciled))
	}
	var match bool
	for _, connection := range unmadeConnections {
		match = false
		for _, linked := range reconciled {
			if connection.Equals(linked) {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("failed to find %v in reported reconciled connections %v\n", connection, reconciled)
		}
	}
}
