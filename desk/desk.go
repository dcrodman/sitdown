package desk

import (
	"github.com/stianeikeland/go-rpio"
	"sync"
	"time"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
)

var (
	moveMux = &sync.Mutex{}
)

func Setup() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonDown.Output()
	pinButtonDown.PullUp()
}

func Cleanup() {
	defer rpio.Close()
	pinButtonUp.PullOff()
	pinButtonDown.PullOff()
}

func lock() {
	moveMux.Lock()
}

func unlock() {
	moveMux.Unlock()
}

func RaiseForDuration(duration int) {
	lock()
	defer unlock()
	raise()
	sleep(duration)
	stop()
}

func LowerForDuration(duration int) {
	lock()
	defer unlock()
	raise()
	sleep(duration)
	stop()
}

func raise() {
	pinButtonUp.Low()
}

func lower() {
	pinButtonDown.Low()
}

func stop() {
	pinButtonUp.High()
	pinButtonDown.High()
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
