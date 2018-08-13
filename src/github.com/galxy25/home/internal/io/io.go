package internal

import (
	"bufio"
	forEach "github.com/galxy25/home/internal/forEach"
	"os"
)

// SerializedLFiles are
// line delimited files of serialized values
// Trying a more idiomatic golang approach: Fire interfaces at will!
// https://blog.chewxy.com/2018/03/18/golang-interfaces/
// https://github.com/golang/go/wiki/CodeReviewComments#interfaces
type SerializedLFile struct {
	FilePath    string
	Serialize   func(deserialized interface{}) (serialized []byte, err error)
	Deserialize func(serialized []byte) (deserialized interface{}, err error)
}

// All lazily iterates over all values of a SerializedLFile
// yielding deserialized values until no more values exist
// or a message is sent on the cancel channel
// returning error(if any)
func (s *SerializedLFile) All(cancel <-chan struct{}) (all chan forEach.Each, err error) {
	all = make(chan forEach.Each)
	file, err := os.OpenFile(s.FilePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		close(all)
		return all, err
	}
	go func() {
		defer close(all)
		defer file.Close()
		reader := bufio.NewReader(file)
		for {
			currentLine, readErr := reader.ReadBytes('\n')
			if readErr != nil {
				return
			}
			deserialized, desErr := s.Deserialize(currentLine)
			select {
			case <-cancel:
				return
			case all <- forEach.Each{
				Item: deserialized,
				Err:  desErr}:
			}
		}
	}()
	return all, err
}

// Store serializes and stores item in a SerializedLFile
// returning stored bytes and serialization error(if any)
func (s *SerializedLFile) Store(item interface{}) (stored []byte, err error) {
	file, err := os.OpenFile(s.FilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()
	stored, err = s.Serialize(item)
	if err != nil {
		return nil, err
	}
	file.Write(stored)
	return stored, err
}
