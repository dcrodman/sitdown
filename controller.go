package main

import (
	"fmt"
	"github.com/stianeikeland/go-rpio"
)

func main() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	defer rpio.Close()
	defer pin.PullOff()

	pin := rpio.Pin(21)
	pin.Output()

	for i := 0; i < 5; i++ {
		pin.PullDown()
	}

	pin.PullOff()
	fmt.Println("It works")
}
