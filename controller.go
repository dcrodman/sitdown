package main

import (
	"github.com/stianeikeland/go-rpio"
	"net/http"
	"net/url"
	"fmt"
	"strconv"
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

	http.HandleFunc("/move", HandleMove)
	http.ListenAndServe(":8080", nil)
}

func HandleMove(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	height, err := strconv.Atoi(vals["height"][0])
	if err != nil || height < 0 || height > 100 {
		fmt.Printf("Invalid height: %d\n", height)
		return
	}

	fmt.Printf("Received move command: %d\n", height)
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
