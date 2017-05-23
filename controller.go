package main

import (
	"github.com/stianeikeland/go-rpio"
	"time"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
)

func main() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	defer rpio.Close()

	raise()
	sleep(2000)
	stopRaising()

	lower()
	sleep(2000)
	stopLowering()

	pinButtonUp.PullOff()
	pinButtonDown.PullOff()
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func raise() {
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.Low()
}

func stopRaising() {
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.High()
}

func lower() {
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.Low()
}

func stopLowering() {
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.High()
}
