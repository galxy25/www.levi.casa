package memory

import (
	"gopkg.in/couchbase/gocb.v1"
)

type Couchdb struct {
	Username string
	Password string
	URI      string
	bucket   gocb.Bucket
}

func (c *Couchdb) Map(point Point, item interface{}) (error, Point) {
	return nil, Point{}
}

func (c *Couchdb) Fetch(point Point) (error, interface{}) {
	return nil, nil
}

func (c *Couchdb) Update(point Point, path string, op Operation, value interface{}) (error, interface{}) {
	return nil, nil

}

func (c *Couchdb) Delete(point Point) error {
	return nil
}

func (c *Couchdb) Query(index, params interface{}) (error, []interface{}) {
	return nil, nil

}

func (c *Couchdb) View(name string, params interface{}) (error, []interface{}) {
	return nil, nil

}

func (c *Couchdb) Configure(settings interface{}) error {
	return nil
}

func NewCouchdb(role Role, uri string) (error, Couchdb) {
	return nil, Couchdb{}
}
