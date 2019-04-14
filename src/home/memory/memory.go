// Package memory implements
// remembering of digital events.
package memory

import (
	"github.com/galxy25/www.levi.casa/home/data"
)

type Memory struct {
	mapper Mapper
}

type Role struct {
	Name     string
	Password string
}

func (m *Memory) Memorize(memory *data.Memory) (error, Point) {
	return nil, Point{}
}

func (m *Memory) Recall(point Point) (error, *data.Memory) {
	return nil, &data.Memory{}
}

func New(role Role, endpoint string, settings interface{}) (error, Memory) {
	var memory Memory
	// if endpoint is like couchbase://
	err, couchdb := NewCouchdb(role, endpoint)
	if err != nil {
		return err, memory
	}
	memory = Memory{
		mapper: &couchdb,
	}
	return nil, memory
}

type Mapper interface {
	Map(point Point, value interface{}) (error, Point)
	Fetch(point Point) (error, interface{})
	Update(point Point, path string, op Operation, value interface{}) (error, interface{})
	Delete(point Point) error
	Query(index, params interface{}) (error, []interface{})
	View(name string, params interface{}) (error, []interface{})
	Configure(settings interface{}) error
}

type Point struct {
	Partition string
	Key       string
}

type Operation int

const (
	INSERT Operation = iota
	UPSERT
	APPEND
	PREPEND
	REMOVE
)
