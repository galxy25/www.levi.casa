package communicator

import (
	"bufio"
	"errors"
	"fmt"
	data "github.com/galxy25/home/data"
	forEach "github.com/galxy25/home/internal/forEach"
	io "github.com/galxy25/home/internal/io"
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
	connected := make(chan *data.EmailConnect)
	defer close(done)
	defer close(connected)
	// Construct desired connections
	desired_connections := []data.EmailConnect{
		data.EmailConnect{
			EmailConnect:           "Farewell, Salutations,Body",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622299",
		},
		data.EmailConnect{
			EmailConnect:           "Body, Salutations,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622200",
		},
		data.EmailConnect{
			EmailConnect:           "Salutations,Body,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622217",
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
		leftover_desired_connection, err := data.EmailConnectFromString(desired_scanner.Text())
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
	connected := make(chan *data.EmailConnect)
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
	desired_connections := []*data.EmailConnect{
		&data.EmailConnect{
			EmailConnect:           "Salutations,Body,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622217",
		},
		&data.EmailConnect{
			EmailConnect:           "Body, Salutations,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622200",
		},
		&data.EmailConnect{
			EmailConnect:           "Farewell, Salutations,Body",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622299",
		},
	}
	for _, desired_connection := range desired_connections {
		desired_file.WriteString(desired_connection.BaseString())
		// Fake that we made the connection
		test_connected_at := "2531622299"
		current_file.WriteString(fmt.Sprintf("%v %v\n", desired_connection.BaseString(), test_connected_at))
		// Send the faux made connection to the sweeper
		connected <- desired_connection
	}
	// Assert desired file is empty of
	// connections that should've been swept
	unswept := false
	desired_file, _ = os.OpenFile(desired, os.O_RDONLY, 0644)
	desired_scanner := bufio.NewScanner(desired_file)
	desired_scanner.Split(bufio.ScanLines)
	for desired_scanner.Scan() {
		leftover_desired_connection, err := data.EmailConnectFromString(desired_scanner.Text())
		if err != nil {
			continue
		}
		for _, desired_connection := range desired_connections {
			if leftover_desired_connection.Matches(desired_connection) {
				unswept = true
				break
			}
		}
		if unswept {
			t.Errorf("Failed to sweep: %v\n", desired_scanner.Text())
		}
	}
}

func TestConnectMakesConnections(t *testing.T) {
	desired, current := "TestConnectMakesConnections.desired", "TestConnectMakesConnections.current"
	defer os.Remove(desired)
	defer os.Remove(current)
	desired_file, err := os.OpenFile(desired, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Unable to open file: %v\n", desired_file)
	}
	defer desired_file.Close()
	// Construct desired connections
	desired_connections := []data.EmailConnect{
		data.EmailConnect{
			EmailConnect:           "Salutations,Body,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622217",
		},
		data.EmailConnect{
			EmailConnect:           "Farewell, Salutations,Body",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622299",
		},
		data.EmailConnect{
			EmailConnect:           "Body, Salutations,Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622200",
		},
	}
	for _, desired_connection := range desired_connections {
		desired_file.WriteString(desired_connection.ToString())
	}
	saved := sns_publisher
	defer func() { sns_publisher = saved }()
	sns_publisher = func(message string) (resp interface{}, err error) {
		return resp, err
	}
	connected := make(chan *data.EmailConnect, 10)
	defer close(connected)
	Connect(desired, current, connected)
	current_file, err := os.OpenFile(current, os.O_RDONLY, 0644)
	if err != nil {
		t.Errorf("Unable to open file: %v\n", current_file)
	}
	defer current_file.Close()
	current_scanner := bufio.NewScanner(current_file)
	current_scanner.Split(bufio.ScanLines)
	for current_scanner.Scan() {
		current_connection, err := data.EmailConnectFromString(current_scanner.Text())
		if err != nil {
			t.Errorf("Invalid current connection: %v\n", current_connection)
			continue
		}
		for index, desired_connection := range desired_connections {
			if desired_connection.Matches(current_connection) {
				desired_connections = append(desired_connections[:index], desired_connections[index+1:]...)
				break
			}
		}
	}
	if len(desired_connections) > 0 {
		t.Errorf("Failed to make connections: %v\n", desired_connections)
	}
}

func SerializeConnection(deserialized interface{}) (serialized []byte, err error) {
	connection, ok := deserialized.(data.EmailConnect)
	if !ok {
		return serialized, errors.New(fmt.Sprintf("Unable to serialize %v\n to an  data.EmailConnect", deserialized))
	}
	serialized = []byte(connection.ToString())
	return serialized, nil
}

func DeserializeConnection(serialized []byte) (deserialized interface{}, err error) {
	deserialized, err = data.EmailConnectFromString(string(serialized))
	return deserialized, err
}

type ConnectionFile struct {
	*io.SerializableLFile
}

func NewConnectionFile(filePath string) (file ConnectionFile) {
	sf := &io.SerializableLFile{
		FilePath:    filePath,
		Serialize:   SerializeConnection,
		Deserialize: DeserializeConnection,
	}
	return ConnectionFile{sf}
}

func (c *ConnectionFile) WriteConnections(connections []interface{}) (err error) {
	for connection := range connections {
		_, err := c.Store(connection)
		if err != nil {
			break
		}
	}
	return err
}

func (c *ConnectionFile) FindConnections(connections []interface{}) (found []interface{}, err error) {
	// selected, selectErr := forEach.Select(c.All, func(item interface{}) (predicate bool, err error) {
	// 	connection, ok := (item).(*data.EmailConnect)
	// 	if !ok {
	// 		predicate = false
	// 		return predicate, errors.New(fmt.Sprintf("Unable to cast %v\n to data.EmailConnect", item))
	// 	}
	// 	if connection.Matches(&seedData[0]) {
	// 		predicate = true
	// 	}
	// 	return predicate, err
	// })
	// for selected := range selected
	return
}

func TestConnectReportsSweepableConnections(t *testing.T) {
	filePath := "fun.functions"
	defer os.Remove(filePath)
	a := io.SerializableLFile{
		FilePath:    filePath,
		Serialize:   SerializeConnection,
		Deserialize: DeserializeConnection,
	}
	seedData := []data.EmailConnect{
		data.EmailConnect{
			EmailConnect:           "Salutations, Body, Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622217",
		},
		data.EmailConnect{
			EmailConnect:           "Farewell, Salutations, Body",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622299",
		},
		data.EmailConnect{
			EmailConnect:           "Body, Salutations, Farewell",
			EmailConnectId:         "tester@test.com",
			SubscribeToMailingList: false,
			ReceiveEpoch:           "1531622200",
		},
		data.EmailConnect{
			EmailConnect:           "wow, whoa, well",
			EmailConnectId:         "rando@randos.com",
			SubscribeToMailingList: true,
			ReceiveEpoch:           "1531622369",
		},
	}
	for _, seed := range seedData {
		_, err := a.Store(seed)
		if err != nil {
			t.Error(err)
		}
	}
	all, _, exit := a.All()
	for stuff := range all {
		t.Log(stuff)
	}
	err := <-exit
	if err != nil {
		t.Error(err)
	}
	extracted, exit := forEach.Detect(a.All, func(item interface{}) (predicate bool, err error) {
		connection, ok := (item).(*data.EmailConnect)
		if !ok {
			predicate = false
			return predicate, errors.New(fmt.Sprintf("Unable to cast %v\n to data.EmailConnect", item))
		}
		if connection.Matches(&seedData[1]) {
			predicate = true
		}
		return predicate, err
	})
	t.Log(extracted)
	if extracted == nil {
		t.Error("Failed to detect seeded connection")
	}
	for err := range exit {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestConnectConnectsNewConnections(t *testing.T) {}
