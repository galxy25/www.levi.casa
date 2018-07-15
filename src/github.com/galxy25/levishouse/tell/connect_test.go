package tell

import (
	"bufio"
	xip "github.com/galxy25/levishouse/xip"
	"os"
	"testing"
)

// Test that connections that are present
// in done file are swept from todo file
func TestSweepConnectionsSweepsActuatedConnections(t *testing.T) {
	desired, current := "TestSweepConnectionsSweepsActuatedConnections.desired", "TestSweepConnectionsSweepsActuatedConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	desired_file, err := os.OpenFile(desired, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Unable to open file: %v\n", desired_file)
	}
	defer desired_file.Close()
	current_file, err := os.OpenFile(current, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Unable to open file: %v\n", current_file)
	}
	defer current_file.Close()
	done := make(chan struct{})
	connected := make(chan string)
	defer close(done)
	defer close(connected)
	// Construct desired connections
	desired_connections := []xip.EmailConnect{
		xip.EmailConnect{
			EmailConnect:           "Salutations,Body,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622217",
		},
		xip.EmailConnect{
			EmailConnect:           "Farewell, Salutations,Body",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622299",
		},
		xip.EmailConnect{
			EmailConnect:           "Body, Salutations,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622200",
		},
	}
	for _, desired_connection := range desired_connections {
		desired_file.WriteString(desired_connection.ToString())
		desired_connection.ConnectEpoch = "2531622299"
		current_file.WriteString(desired_connection.ToString())
	}
	SweepConnections(desired, current, done, connected)
	// Assert desired file is empty of
	// connections that should've been swept
	unswept := false
	desired_file, _ = os.OpenFile(desired, os.O_RDONLY, 0644)
	desired_scanner := bufio.NewScanner(desired_file)
	desired_scanner.Split(bufio.ScanLines)
	for desired_scanner.Scan() {
		leftover_desired_connection, err := xip.EmailConnectFromString(desired_scanner.Text())
		if err != nil {
			continue
		}
		for _, desired_connection := range desired_connections {
			if leftover_desired_connection.Matches(&desired_connection) {
				unswept = true
				break
			}
		}
		if unswept {
			t.Errorf("Failed to sweep: %v\n", desired_scanner.Text())
		}
	}
}

// Test that new connections sent to
// SweepConnections connected channel are swept
func TestSweepConnectionsSweepsNewConnections(t *testing.T) {
	desired, current := "TestSweepConnectionsSweepsNewConnections.desired", "TestSweepConnectionsSweepsNewConnections.current"
	done := make(chan struct{})
	connected := make(chan string)
	defer close(done)
	defer close(connected)
	SweepConnections(desired, current, done, connected)
	defer os.Remove(desired)
	defer os.Remove(current)
	desired_file, err := os.OpenFile(desired, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Unable to open file: %v\n", desired_file)
	}
	defer desired_file.Close()
	current_file, err := os.OpenFile(current, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Unable to open file: %v\n", current_file)
	}
	defer current_file.Close()
	// Construct desired connections
	desired_connections := []xip.EmailConnect{
		xip.EmailConnect{
			EmailConnect:           "Salutations,Body,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622217",
		},
		xip.EmailConnect{
			EmailConnect:           "Farewell, Salutations,Body",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622299",
		},
		xip.EmailConnect{
			EmailConnect:           "Body, Salutations,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622200",
		},
	}
	for _, desired_connection := range desired_connections {
		desired_file.WriteString(desired_connection.ToString())
		desired_connection.ConnectEpoch = "2531622299"
		current_file.WriteString(desired_connection.ToString())
		desired_connection.ConnectEpoch = ""
		connected <- desired_connection.ToString()
	}
	// Assert desired file is empty of
	// connections that should've been swept
	unswept := false
	desired_file, _ = os.OpenFile(desired, os.O_RDONLY, 0644)
	desired_scanner := bufio.NewScanner(desired_file)
	desired_scanner.Split(bufio.ScanLines)
	for desired_scanner.Scan() {
		leftover_desired_connection, err := xip.EmailConnectFromString(desired_scanner.Text())
		if err != nil {
			continue
		}
		for _, desired_connection := range desired_connections {
			if leftover_desired_connection.Matches(&desired_connection) {
				unswept = true
				break
			}
		}
		if unswept {
			t.Errorf("Failed to sweep: %v\n", desired_scanner.Text())
		}
	}
}
