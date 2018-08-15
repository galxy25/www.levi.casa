package internal

import (
	"time"
)

// Ready is a type of function that reports
// readiness of some state or action, returning
// bool for readiness and error (if any).
type Ready func() (ready bool, err error)

// Await waits until the ready function is ready
// or errors, returning success and error (if any).
// To stop waiting, send on the stop channel.z
func Await(ready Ready, stop <-chan struct{}) (success bool, err error) {
	for tries := 0; ; tries++ {
		select {
		case <-stop:
			return success, err
		default:
			success, err := ready()
			if err != nil {
				return success, err
			}
			if !success {
				// exponentially back off before the next attempt
				// https://github.com/adonovan/gopl.io/blob/77e9f810f3c2502e9c641b97e09f9721424090f5/ch5/wait/wait.go#L30
				time.Sleep((10 * time.Millisecond) << uint(tries))
				continue
			}
			return success, err
		}
	}
}
