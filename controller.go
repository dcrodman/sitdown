package main

import (
	"github.com/stianeikeland/go-rpio"
	"net/http"
	"net/url"
	"fmt"
	"strconv"
	"time"
	"sync"
	"os"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
)

var (
	moveMux = &sync.Mutex{}
	server = false
)

func main() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	defer rpio.Close()

	for _, arg := range os.Args {
		switch (arg) {
		case "-s", "--server":
			server = true
		}
	}

	raise()
	sleep(2000)
	stopRaising()

	lower()
	sleep(2000)
	stopLowering()

	if server {
		http.HandleFunc("/move", HandleMove)
		http.HandleFunc("/set", HandleSet)
		http.ListenAndServe(":8080", nil)
	}

	pinButtonUp.PullOff()
	pinButtonDown.PullOff()
}

func HandleMove(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	direction := vals["direction"][0]
	if direction != "down" && direction != "up" {
		fmt.Printf("Invalid direction: %s\n", direction)
		return
	}
	time, err := strconv.Atoi(vals["time"][0])
	if err != nil || time < 0 || time > 10000 {
		fmt.Printf("Invalid time: %d\n", time)
		return
	}

	fmt.Printf("Received move command: %s %d\n", direction, time)
}

func HandleSet(responseWriter http.ResponseWriter, request *http.Request) {
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

	fmt.Printf("Received set command: %d\n", height)
}

func move(direction string, time int) {
	moveMux.Lock()
	defer moveMux.Unlock()
	switch(direction) {
	case "up":
		raise()
		sleep(time)
		stopRaising()
	case "down":
		lower()
		sleep(time)
		stopLowering()
	}
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
