package communicator

import (
	"errors"
	"fmt"
	"github.com/galxy25/www.levi.casa/home/data"
	forEach "github.com/galxy25/www.levi.casa/home/internal/forEach"
	io "github.com/galxy25/www.levi.casa/home/internal/io"
)

// SerializeConnection attempts to serialize
// a interface to the serialized representation
// of a connection object, returning the
// serialized byte representation and error (if any).
func SerializeConnection(deserialized interface{}) ([]byte, error) {
	connection, err := castAsConnectionPtr(deserialized)
	if err != nil {
		return nil, err
	}
	serialized := []byte(fmt.Sprintf("%v\n", connection.String()))
	return serialized, nil
}

// SerializeConnection attempts to deserialize
// bytes to a connection object, returning the
// deserialized connection and error (if any).
func DeserializeConnection(serialized []byte) (interface{}, error) {
	connection, err := data.ConnectionFromString(string(serialized))
	if err != nil {
		return nil, err
	}
	deserialized := connection
	return deserialized, err
}

// ConnectionFile represents a file of
// line delimited serialized connections.
type ConnectionFile struct {
	*io.SerializedLFile
}

// NewConnectionFile returns a handle
// to a connectionFile at the specified file path,
// file will lazily be created the first time
// a read or write is attempted on it.
func NewConnectionFile(filePath string) ConnectionFile {
	sf := &io.SerializedLFile{
		FilePath:    filePath,
		Serialize:   SerializeConnection,
		Deserialize: DeserializeConnection,
	}
	return ConnectionFile{sf}
}

// WriteConnection writes a connection to a
// ConnectionFile, returning error (if any).
func (c *ConnectionFile) WriteConnection(connection *data.Connection) (err error) {
	_, err = c.Store(connection)
	return err
}

// WriteConnections writes connections to a
// ConnectionFile, returning error(if any).
func (c *ConnectionFile) WriteConnections(connections []*data.Connection) (err error) {
	for _, connection := range connections {
		err = c.WriteConnection(connection)
		if err != nil {
			return err
		}
	}
	return err
}

// FindConnection returns bool indicating whether
// connection was found in ConnectionFile
// additionally returning error (if any).
func (c *ConnectionFile) FindConnection(connection *data.Connection) (detected bool, err error) {
	_, err = forEach.Detect(c.All, func(item interface{}) (predicate bool, err error) {
		connectionItem, err := castAsConnectionPtr(item)
		if err != nil {
			return predicate, err
		}
		if connectionItem.Equals(connection) {
			detected = true
			predicate = true
		}
		return predicate, err
	})
	return detected, err
}

// FindConnections finds all connections
// in ConnectionFile that match any
// connection in the list of connections to find
// returning all matches and errors (if any).
func (c *ConnectionFile) FindConnections(connections []*data.Connection) (found []*data.Connection, errs []error) {
	finder, findErr := forEach.Select(c.All, func(item interface{}) (predicate bool, err error) {
		connectionItem, err := castAsConnectionPtr(item)
		if err != nil {
			return predicate, err
		}
		for _, connection := range connections {
			if connectionItem.Equals(connection) {
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
		connectionItem, err := castAsConnectionPtr(find.Item)
		if err != nil {
			continue
		}
		found = append(found, connectionItem)

	}
	return found, errs
}

// Each lazily returns each connection
// in a ConnectionFile, returning lazy iterator
// and err (if any).
// Send on the finish channel to
// terminate an in progress iteration.
func (c *ConnectionFile) Each(finish <-chan struct{}) (connections chan *data.Connection, err error) {
	connections = make(chan *data.Connection)
	connectionsIterator, err := c.All(finish)
	if err != nil {
		return connections, err
	}
	go func() {
		defer close(connections)
		for {
			select {
			case <-finish:
				return
			case item, more := <-connectionsIterator:
				if !more {
					return
				}
				if item.Err != nil {
					err = item.Err
					return
				}
				connection, err := castAsConnectionPtr(item.Item)
				if err != nil {
					return
				}
				connections <- connection
			}
		}
	}()
	return connections, err
}

// castAsConnectionPtr attempts to cast a interface
// to a connection object,
// returning connection and cast error (if any).
func castAsConnectionPtr(a interface{}) (connection *data.Connection, err error) {
	connection, ok := a.(*data.Connection)
	if !ok {
		// TODO: return named error
		return connection, errors.New(fmt.Sprintf("Unable to cast %v to a *data.Connection\n", a))
	}
	return connection, err
}
