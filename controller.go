package main

import (
	"fmt"
	"github.com/dcrodman/sitdown/desk"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func main() {
	for _, arg := range os.Args {
		switch arg {
		case "-c":
			EnterCommandMode()
			os.Exit(0)
		case "-t", "--test":
			InitializePins()
			move("up", 2000)
			move("down", 2000)
			os.Exit(0)
		}
	}

	InitializePins()

	PubNubSubscribe()
	http.HandleFunc("/move", HandleMove)
	http.HandleFunc("/set", HandleSet)
	http.ListenAndServe(":8080", nil)
}

func InitializePins() {
	desk.Setup()
	defer desk.Cleanup()
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
	duration, err := strconv.Atoi(vals["time"][0])
	if err != nil || duration < 0 || duration > 10000 {
		fmt.Printf("Invalid time: %d\n", duration)
		return
	}

	fmt.Printf("Received move command: %s %d\n", direction, duration)
	move(direction, duration)
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
	switch direction {
	case "up":
		desk.RaiseForDuration(time)
	case "down":
		desk.LowerForDuration(time)
	}
}
