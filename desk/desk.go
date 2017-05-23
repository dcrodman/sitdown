package desk

import (
	"github.com/stianeikeland/go-rpio"
	"sync"
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
}

func Cleanup() {
	rpio.Close()
	pinButtonUp.PullOff()
	pinButtonDown.PullOff()
}

func Lock() {
	moveMux.Lock()
}

func Unlock() {
	moveMux.Unlock()
}

func Raise() {
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.Low()
}

func StopRaising() {
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.High()
}

func Lower() {
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.Low()
}

func StopLowering() {
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.High()
}