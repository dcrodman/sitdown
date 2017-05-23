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

	pin := rpio.Pin(21)
	for i := 0; i < 5; i++ {
		pin.PullDown()
	}

	fmt.Println("It works")
}
