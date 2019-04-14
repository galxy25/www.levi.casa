package memory

import (
	"github.com/galxy25/www.levi.casa/home/data"
	"testing"
)

type TestMapper struct {
	data map[string]interface{}
}

func (tm *TestMapper) Map(point Point, value interface{}) (error, Point) {
	tm.data[point.Partition+point.Key] = value
	return nil, Point{}
}
func (tm *TestMapper) Fetch(point Point) (error, interface{}) {
	return nil, nil
}
func (tm *TestMapper) Update(point Point, path string, op Operation, value interface{}) (error, interface{}) {
	return nil, nil
}
func (tm *TestMapper) Delete(point Point) error {
	return nil
}
func (tm *TestMapper) Query(index, params interface{}) (error, []interface{}) {
	return nil, nil
}
func (tm *TestMapper) View(name string, params interface{}) (error, []interface{}) {
	return nil, nil
}
func (tm *TestMapper) Configure(settings interface{}) error {
	return nil
}

func TestRecalledMemoryEqualsMemorizedMemory(t *testing.T) {
	memory := Memory{
		mapper: &TestMapper{},
	}
	newMemory := data.Memory{
		PublishEpoch: 123,
	}
	_, point := memory.Memorize(&newMemory)
	_, oldMemory := memory.Recall(point)
	if oldMemory.PublishEpoch != newMemory.PublishEpoch {
		t.Errorf("Expected memorized value: %v to equal recalled value: %v", newMemory, oldMemory)
	}
}
