package internal

import (
	"bufio"
	"os"
)

// SerializableLFiles are
// serializable line delimited files
// Trying a more idiomatic golang approach: Fire interfaces at will!
// https://blog.chewxy.com/2018/03/18/golang-interfaces/
// https://github.com/golang/go/wiki/CodeReviewComments#interfaces
type SerializableLFile struct {
	FilePath    string
	Serialize   func(deserialized interface{}) (serialized []byte, err error)
	Deserialize func(serialized []byte) (deserialized interface{}, err error)
}

// All returns lazy iterator for SerializableLFile values,
// iteration stopper, and iteration error(if any)
// to cancel iteration send on the cancel channel
// iteration stops on the first iteration error
func (s *SerializableLFile) All() (all chan interface{}, cancel chan struct{}, exit chan error) {
	all = make(chan interface{}, 1)
	cancel = make(chan struct{}, 1)
	exit = make(chan error, 1)
	go func() {
		var err error
		defer close(all)
		defer close(exit)
		file, err := os.OpenFile(s.FilePath, os.O_RDONLY|os.O_CREATE, 0644)
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for err == nil && scanner.Scan() {
			select {
			case <-cancel:
				return
			default:
				deserialized, err := s.Deserialize(scanner.Bytes())
				if err != nil {
					exit <- err
					return
				}
				all <- deserialized
			}
		}
	}()
	return all, cancel, exit
}

// Store stores a given item in a SerializableLFile
// returning stored bytes and serialization error(if any)
func (s *SerializableLFile) Store(item interface{}) (stored []byte, err error) {
	file, err := os.OpenFile(s.FilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()
	stored, err = s.Serialize(item)
	if err != nil {
		return nil, err
	}
	file.Write(stored)
	return stored, err
}
