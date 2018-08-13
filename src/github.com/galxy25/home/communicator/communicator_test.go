package communicator

import (
// "github.com/galxy25/home/data"
// helper "github.com/galxy25/home/internal/test"
// "os"
// "testing"
)

// func defaultConnections() (connections []*data.Connection) {
// 	connections = []*data.Connection{
// 		&data.Connection{
// 			Message:                "Salutations, Body, Farewell",
// 			ConnectionId:           "tester@test.com",
// 			SubscribeToMailingList: false,
// 			ReceiveEpoch:           1531622217,
// 		},
// 		&data.Connection{
// 			Message:                "Farewell, Salutations, Body",
// 			ConnectionId:           "tester@test.com",
// 			SubscribeToMailingList: false,
// 			ReceiveEpoch:           1531622299,
// 		},
// 		&data.Connection{
// 			Message:                helper.RandomString(1000000),
// 			ConnectionId:           "tester@test.com",
// 			SubscribeToMailingList: false,
// 			ReceiveEpoch:           1531622200,
// 		},
// 		&data.Connection{
// 			Message:                "wow, whoa, well",
// 			ConnectionId:           "rando@randos.com",
// 			SubscribeToMailingList: true,
// 			ReceiveEpoch:           1531622369,
// 		},
// 	}
// 	return connections
// }

// func TestConnectMakesConnections(t *testing.T) {
// 	desired, current := "TestConnectMakesConnections.desired", "TestConnectMakesConnections.current"
// 	defer os.Remove(desired)
// 	defer os.Remove(current)
// 	desiredConnections := NewConnectionFile(desired)
// 	seedData := defaultConnections()
// 	err := desiredConnections.WriteConnections(seedData)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	saved := snsPublisher
// 	defer func() { snsPublisher = saved }()
// 	snsPublisher = func(message string) (resp interface{}, err error) {
// 		return resp, err
// 	}
// 	newConnectionsQueue := make(chan *data.Connection)
// 	defer close(newConnectionsQueue)
// 	Connect(desired, current, newConnectionsQueue)
// 	currentConnections := NewConnectionFile(current)
// 	tries := 3
// 	for tries > 0 {
// 		wrote, err := currentConnections.FindConnections(seedData)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if len(wrote) == len(seedData) {
// 			break
// 		}
// 	}
// 	for _, seed := range seedData {
// 		detected, err := currentConnections.FindConnection(seed)
// 		if !detected {
// 			t.Errorf("failed to find %v in %v\n", seed, current)
// 		}
// 		if err != nil {
// 			t.Errorf("error %v while trying to detect %v in %v\n ", err, seed, current)
// 		}
// 	}
// }

// func TestConnectConnectsNewConnections(t *testing.T) {
// 	desired, current := "TestConnectConnectsNewConnections.desired", "TestConnectConnectsNewConnections.current"
// 	defer os.Remove(desired)
// 	defer os.Remove(current)
// 	saved := snsPublisher
// 	defer func() { snsPublisher = saved }()
// 	snsPublisher = func(message string) (resp interface{}, err error) {
// 		return resp, err
// 	}
// 	seedData := defaultConnections()
// 	newConnectionsQueue := make(chan *data.Connection)
// 	defer close(newConnectionsQueue)
// 	Connect(desired, current, newConnectionsQueue)
// 	desiredConnections := NewConnectionFile(desired)
// 	err := desiredConnections.WriteConnections(seedData)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	for _, seed := range seedData {
// 		newConnectionsQueue <- &data.Connection{
// 			Message:                seed.Message,
// 			ConnectionId:           seed.ConnectionId,
// 			SubscribeToMailingList: seed.SubscribeToMailingList,
// 			ReceiveEpoch:           seed.ReceiveEpoch,
// 		}
// 	}
// 	currentConnections := NewConnectionFile(current)
// 	tries := 3
// 	for tries > 0 {
// 		wrote, err := currentConnections.FindConnections(seedData)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if len(wrote) == len(seedData) {
// 			break
// 		}
// 	}
// 	for _, seed := range seedData {
// 		detected, err := currentConnections.FindConnection(seed)
// 		if !detected {
// 			t.Errorf("failed to find %v in %v\n", seed, current)
// 		}
// 		if err != nil {
// 			t.Errorf("error %v while trying to detect %v in %v\n ", err, seed, current)
// 		}
// 	}
// }
