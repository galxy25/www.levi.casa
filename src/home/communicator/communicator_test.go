package communicator

import (
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/galxy25/www.levi.casa/home/data"
	helper "github.com/galxy25/www.levi.casa/home/internal/test"
	"os"
	"testing"
)

var realSesPublisher = sesPublisher
var mockSesPublisher = func(email *Email) (response *ses.SendEmailOutput, err error) {
	return response, err
}
var realSmsPublisher = smsPublisher
var mockSmsPublisher = func(sms *SMS) (err error) {
	return err
}

func TestSuccesfulLinkRecordsLinkedConnection(t *testing.T) {
	sesPublisher = mockSesPublisher
	smsPublisher = mockSmsPublisher
	defer func() {
		sesPublisher = realSesPublisher
		smsPublisher = realSmsPublisher
	}()
	desired, current := "TestSuccesfulLinkRecordsLinkedConnection.desired", "TestSuccesfulLinkRecordsLinkedConnection.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := New(desired, current)
	var sender Sender
	var err error
	for connectionType, connectionGenerator := range helper.ConnectionGenerators {
		connection := connectionGenerator()
		switch connectionType {
		case "email":
			sender, err = EmailFromConnection(connection)
			if err != nil {
				t.Error(err)
			}
		case "sms":
			sender, err = SmsFromConnection(connection)
			if err != nil {
				t.Error(err)
			}
		}
		linked, err := comm.Link(connection, sender)
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
}

func TestRecordRecordsConnection(t *testing.T) {
	desired, current := "TestRecordRecordsConnection.desired", "TestRecordRecordsConnection.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := New(desired, current)
	for _, connectionGenerator := range helper.ConnectionGenerators {
		connection := connectionGenerator()
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
}

func TestReceivedReportsAllUnlinkedConnections(t *testing.T) {
	desired, current := "TestReceivedReportsAllUnlinkedConnections.desired", "TestReceivedReportsAllUnlinkedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := New(desired, current)
	connections := []*data.Connection{
		helper.RandomEmailConnection(),
		helper.RandomSmsConnection(),
		helper.RandomEmailConnection(),
		helper.RandomSmsConnection(),
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
	sesPublisher = mockSesPublisher
	smsPublisher = mockSmsPublisher
	defer func() {
		sesPublisher = realSesPublisher
		smsPublisher = realSmsPublisher
	}()
	desired, current := "TestSentReportsAllLinkedConnections.desired", "TestSentReportsAllLinkedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := New(desired, current)
	var sender Sender
	var err error
	var allConnections []*data.Connection
	for connectionType, connectionGenerator := range helper.ConnectionGenerators {
		connections := []*data.Connection{
			connectionGenerator(),
			connectionGenerator(),
			connectionGenerator(),
		}
		allConnections = append(allConnections, connections...)
		for _, connection := range connections {
			switch connectionType {
			case "email":
				sender, err = EmailFromConnection(connection)
				if err != nil {
					t.Error(err)
				}
			case "sms":
				sender, err = SmsFromConnection(connection)
				if err != nil {
					t.Error(err)
				}
			}
			_, err := comm.Link(connection, sender)
			if err != nil {
				t.Error(err)
			}
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
	if len(reported) != len(allConnections) {
		t.Errorf("expected %v linked connections, got %v", len(allConnections), len(reported))
	}
	var match bool
	for _, connection := range allConnections {
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
	sesPublisher = mockSesPublisher
	smsPublisher = mockSmsPublisher
	defer func() {
		sesPublisher = realSesPublisher
		smsPublisher = realSmsPublisher
	}()
	desired, current := "TestReconcileLinksAllUnlinkedConnections.desired", "TestReconcileLinksAllUnlinkedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	comm := New(desired, current)
	var allUnmadeConnections []*data.Connection
	var sender Sender
	var err error
	for connectionType, connectionGenerator := range helper.ConnectionGenerators {
		connections := []*data.Connection{
			connectionGenerator(),
			connectionGenerator(),
			connectionGenerator(),
		}
		for _, connection := range connections {
			switch connectionType {
			case "email":
				sender, err = EmailFromConnection(connection)
				if err != nil {
					t.Error(err)
				}
			case "sms":
				sender, err = SmsFromConnection(connection)
				if err != nil {
					t.Error(err)
				}
			}
			_, err := comm.Link(connection, sender)
			if err != nil {
				t.Error(err)
			}
		}
		unmadeConnections := []*data.Connection{
			connectionGenerator(),
			connectionGenerator(),
			connectionGenerator(),
		}
		allUnmadeConnections = append(allUnmadeConnections, unmadeConnections...)
		for _, connection := range unmadeConnections {
			err := comm.Record(connection)
			if err != nil {
				t.Error(err)
			}
		}
	}
	reconciled, err := comm.Reconcile()
	if err != nil {
		t.Error(err)
	}
	if len(reconciled) != len(allUnmadeConnections) {
		t.Errorf("expected %v linked connections, got %v", len(allUnmadeConnections), len(reconciled))
	}
	var match bool
	for _, connection := range allUnmadeConnections {
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
